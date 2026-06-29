package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Release describes a published GitHub release.
type Release struct {
	Version string // tag with any leading "v" stripped, e.g. "0.7.0"
	URL     string // the release page (html_url)
	Notes   string // the release body (markdown)
}

// releaseAPIURL is the GitHub endpoint queried for the latest release. It's a
// package var so tests can point it at an httptest server.
var releaseAPIURL = "https://api.github.com/repos/raihanstark/financy/releases/latest"

// updateHTTPClient has a short timeout so a slow or unreachable network never
// hangs the caller (the update check runs off the UI thread regardless).
var updateHTTPClient = &http.Client{Timeout: 5 * time.Second}

// LatestRelease fetches the latest published release from GitHub. The provided
// context bounds the request lifetime in addition to the client timeout.
func LatestRelease(ctx context.Context) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPIURL, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "financy/"+Version)

	resp, err := updateHTTPClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("github: unexpected status %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Body    string `json:"body"`
	}
	// Cap the body so a misbehaving endpoint can't exhaust memory.
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return Release{}, err
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return Release{}, fmt.Errorf("github: release has no tag")
	}
	return Release{
		Version: normalizeVersion(payload.TagName),
		URL:     strings.TrimSpace(payload.HTMLURL),
		Notes:   strings.TrimSpace(payload.Body),
	}, nil
}

// UpdateAvailable reports whether the latest GitHub release is newer than the
// running build. Development builds (an empty or unparseable Version) never
// report an update so contributors aren't nagged.
func UpdateAvailable(ctx context.Context) (Release, bool, error) {
	if len(versionParts(Version)) == 0 {
		return Release{}, false, nil
	}
	rel, err := LatestRelease(ctx)
	if err != nil {
		return Release{}, false, err
	}
	return rel, CompareVersions(rel.Version, Version) > 0, nil
}

// CompareVersions compares two dotted version strings numerically, ignoring a
// leading "v" and any pre-release/build suffix. It returns -1 if a < b, 0 if
// they're equal, and 1 if a > b. Missing trailing segments count as zero, so
// "0.6" and "0.6.0" are equal.
func CompareVersions(a, b string) int {
	as, bs := versionParts(a), versionParts(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		switch {
		case av < bv:
			return -1
		case av > bv:
			return 1
		}
	}
	return 0
}

// normalizeVersion trims surrounding space and a leading "v"/"V".
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 0 && (v[0] == 'v' || v[0] == 'V') {
		v = v[1:]
	}
	return v
}

// versionParts splits a version into its numeric segments, dropping a leading
// "v" and any pre-release/build suffix (e.g. "1.2.3-rc1" -> [1 2 3]). Returns
// nil for an empty or non-numeric value.
func versionParts(v string) []int {
	v = normalizeVersion(v)
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return nil
	}
	fields := strings.Split(v, ".")
	out := make([]int, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
