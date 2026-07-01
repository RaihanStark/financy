package mobileui

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"

	"github.com/raihanstark/financy/internal/core"
)

// The mobile app keeps a single auto-saved document in the app's private
// storage. There is no file picker (a content:// URI can't be opened as a
// SQLite file), so New / Load-demo replace this one slot in place.

func docPath() string {
	if a := fyne.CurrentApp(); a != nil {
		if st := a.Storage(); st != nil {
			if root := st.RootURI(); root != nil && root.Path() != "" {
				return filepath.Join(root.Path(), "financy.db")
			}
		}
	}
	// Desktop-preview fallback (FINANCY_MOBILE=1).
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "financy", "mobile.financy")
}

// startup opens the single document, creating an empty one on first run. An
// encrypted document (e.g. one imported from a password-protected file) is a
// FINCRYPT blob on disk, not SQLite, so it must be detected and unlocked rather
// than opened with the plain opener.
func (m *mobileApp) startup() {
	path := docPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)

	if _, e := os.Stat(path); e != nil {
		m.showWelcome() // first run — let the user choose how to start
		return
	}

	if enc, err := core.IsEncrypted(path); err != nil {
		m.showWelcome()
		return
	} else if enc {
		m.openEncryptedSandbox(path)
		return
	}

	s, err := core.OpenStore(path)
	if err != nil {
		m.showWelcome() // unreadable/corrupt — offer a fresh start
		return
	}
	m.useStore(s)
}

// useStore wires a store into the shell and renders the Home tab. It subscribes
// so any mutation (add transaction, etc.) re-renders the current screen.
func (m *mobileApp) useStore(s *core.Store) {
	if m.store != nil {
		_ = m.store.Close()
	}
	m.store = s
	s.SetErrorHandler(m.fatal)
	m.showShell() // leave any welcome/unlock page and show the tabbed app (clears nav stack)
	s.Subscribe(func() {
		// Plain documents write through on every change; encrypted ones only
		// persist on Save, so flush them here to keep the single-doc auto-saved.
		if s.Encrypted() {
			_ = s.Save()
		}
		m.syncBack() // mirror to the linked external file, if any
		m.render()
	})
	m.selectTab(0)
}

// replaceDocument discards the current document and creates a fresh one, seeded
// with demo data when demo is true. Used by the settings sheet.
func (m *mobileApp) replaceDocument(demo bool) {
	path := docPath()
	m.sourceURI = nil // a brand-new document isn't linked to any external file
	if m.store != nil {
		_ = m.store.Close()
		m.store = nil
	}
	_ = os.Remove(path)

	s, err := core.NewDocument(path)
	if err != nil {
		m.fatal(err)
		return
	}
	if demo {
		core.SeedDemo(s, "$")
	}
	m.useStore(s)
}
