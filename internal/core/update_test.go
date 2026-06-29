package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.6.0", "0.6.0", 0},
		{"0.6", "0.6.0", 0},    // missing trailing segment counts as zero
		{"v0.6.0", "0.6.0", 0}, // leading "v" ignored
		{"0.6.1", "0.6.0", 1},
		{"0.6.0", "0.7.0", -1},
		{"1.0.0", "0.9.9", 1},
		{"0.10.0", "0.9.0", 1},     // numeric, not lexicographic
		{"v1.2.3-rc1", "1.2.3", 0}, // pre-release suffix dropped
		{"", "0.1.0", -1},
		{"0.1.0", "", 1},
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestLatestRelease(t *testing.T) {
	const body = `{
		"tag_name": "v0.7.0",
		"html_url": "https://github.com/raihanstark/financy/releases/tag/v0.7.0",
		"body": "## What's new\n- Auto update check"
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("missing User-Agent header")
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	old := releaseAPIURL
	releaseAPIURL = srv.URL
	defer func() { releaseAPIURL = old }()

	rel, err := LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.Version != "0.7.0" {
		t.Errorf("Version = %q, want 0.7.0", rel.Version)
	}
	if rel.URL == "" {
		t.Errorf("URL is empty")
	}
	if rel.Notes == "" {
		t.Errorf("Notes is empty")
	}
}

func TestLatestReleaseBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := releaseAPIURL
	releaseAPIURL = srv.URL
	defer func() { releaseAPIURL = old }()

	if _, err := LatestRelease(context.Background()); err == nil {
		t.Fatal("expected an error for non-200 status")
	}
}

func TestUpdateAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://example.com","body":"x"}`))
	}))
	defer srv.Close()

	old := releaseAPIURL
	releaseAPIURL = srv.URL
	defer func() { releaseAPIURL = old }()

	oldVer := Version
	defer func() { Version = oldVer }()

	// Running an older build -> update available.
	Version = "0.6.0"
	if _, avail, err := UpdateAvailable(context.Background()); err != nil || !avail {
		t.Fatalf("UpdateAvailable older build: avail=%v err=%v, want true/nil", avail, err)
	}

	// Already on the latest -> no update.
	Version = "9.9.9"
	if _, avail, err := UpdateAvailable(context.Background()); err != nil || avail {
		t.Fatalf("UpdateAvailable current build: avail=%v err=%v, want false/nil", avail, err)
	}

	// Development build (empty Version) -> never reports an update, no request made.
	Version = ""
	if _, avail, err := UpdateAvailable(context.Background()); err != nil || avail {
		t.Fatalf("UpdateAvailable dev build: avail=%v err=%v, want false/nil", avail, err)
	}
}
