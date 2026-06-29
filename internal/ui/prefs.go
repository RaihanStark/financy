package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// appPrefs is small app-level config (NOT document data): which file to reopen
// and the recent-files list. Stored in the OS config dir.
type appPrefs struct {
	LastPath string   `json:"last_path"`
	Recent   []string `json:"recent"`
	Dark     bool     `json:"dark"`

	// Update-check state. The automatic check is on by default; the zero value
	// of DisableUpdateCheck (false) keeps it enabled.
	DisableUpdateCheck bool   `json:"disable_update_check,omitempty"`
	SkipVersion        string `json:"skip_version,omitempty"`
	LastUpdateCheck    string `json:"last_update_check,omitempty"` // RFC3339 UTC
}

const maxRecent = 8

func prefsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "financy", "prefs.json")
}

func loadPrefs() *appPrefs {
	p := &appPrefs{}
	data, err := os.ReadFile(prefsPath())
	if err == nil {
		_ = json.Unmarshal(data, p)
	}
	return p
}

func (p *appPrefs) save() {
	path := prefsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		_ = os.Rename(tmp, path)
	}
}

// dueForUpdateCheck reports whether the automatic update check should run: it's
// due when never run before, when the stored timestamp is unparseable, or when
// the throttle interval has elapsed.
func (p *appPrefs) dueForUpdateCheck(now time.Time) bool {
	if p.LastUpdateCheck == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339, p.LastUpdateCheck)
	if err != nil {
		return true
	}
	return now.Sub(last) >= updateCheckInterval
}

// remember records path as the current/most-recent file (deduped, capped).
func (p *appPrefs) remember(path string) {
	p.LastPath = path
	out := []string{path}
	for _, r := range p.Recent {
		if r != path && len(out) < maxRecent {
			out = append(out, r)
		}
	}
	p.Recent = out
	p.save()
}
