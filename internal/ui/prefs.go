package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// appPrefs is small app-level config (NOT document data): which file to reopen
// and the recent-files list. Stored in the OS config dir.
type appPrefs struct {
	LastPath string   `json:"last_path"`
	Recent   []string `json:"recent"`
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

