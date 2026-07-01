package mobileui

import (
	"errors"
	"io"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"

	"github.com/raihanstark/financy/internal/core"
)

// openEncryptedSandbox prompts (full-screen) for the passphrase of the encrypted
// sandbox document and opens it. Cancelling returns to the welcome screen so the
// user can retry, open another file, or start over.
func (m *mobileApp) openEncryptedSandbox(path string) {
	m.showUnlockPage(
		"Unlock document",
		"This document is password-protected. Enter its passphrase to open it.",
		func(pass string) (*core.Store, error) { return core.OpenStoreEncrypted(path, pass) },
		func() { m.showWelcome() },
	)
}

// importFile lets the user pick a .financy file to load. The picker returns a
// content:// URI on Android, so we read its bytes and copy them into the app
// sandbox (a real path SQLite can open) rather than opening the URI directly.
func (m *mobileApp) importFile() {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, m.win)
			return
		}
		if r == nil {
			return // cancelled
		}
		src := r.URI()
		data, e := io.ReadAll(r)
		_ = r.Close()
		if e != nil {
			dialog.ShowError(e, m.win)
			return
		}
		m.installImported(data, src)
	}, m.win)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".financy"}))
	d.Show()
}

// installImported validates the picked bytes (prompting for a passphrase if the
// document is encrypted), swaps them into the single sandbox slot, and links
// src for session write-back so later edits flow back to the picked file.
func (m *mobileApp) installImported(data []byte, src fyne.URI) {
	if len(data) == 0 {
		dialog.ShowError(errors.New("that file is empty"), m.win)
		return
	}
	path := docPath()
	tmp := path + ".import"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		dialog.ShowError(err, m.win)
		return
	}

	enc, err := core.IsEncrypted(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		dialog.ShowError(err, m.win)
		return
	}

	open := func(p, pass string) (*core.Store, error) {
		if enc {
			return core.OpenStoreEncrypted(p, pass)
		}
		return core.OpenStore(p)
	}

	// prepare validates (and migrates) the temp copy, then swaps it into the
	// sandbox slot and returns the live store. It only touches the current
	// document after the temp opens cleanly, so a wrong passphrase is harmless.
	prepare := func(pass string) (*core.Store, error) {
		st, e := open(tmp, pass)
		if e != nil {
			return nil, e
		}
		_ = st.Close()
		if m.store != nil {
			_ = m.store.Close()
			m.store = nil
		}
		_ = os.Remove(path)
		if err := os.Rename(tmp, path); err != nil {
			return nil, err
		}
		s, e := open(path, pass)
		if e != nil {
			return nil, e
		}
		m.sourceURI = src // link for session write-back
		return s, nil
	}

	if enc {
		m.showUnlockPage(
			"Open encrypted file",
			"This file is password-protected. Enter its passphrase to open it.",
			prepare,
			func() { _ = os.Remove(tmp) },
		)
		return
	}
	s, e := prepare("")
	if e != nil {
		_ = os.Remove(tmp)
		dialog.ShowError(friendlyOpenErr(e), m.win)
		return
	}
	m.useStore(s)
}

// exportFile writes a copy of the current document to a location the user picks
// (a content:// URI on Android). The sandbox file is already a complete
// .financy document (plain SQLite or an encrypted blob), so we copy its bytes.
func (m *mobileApp) exportFile() {
	if m.store == nil {
		return
	}
	if m.store.Encrypted() {
		_ = m.store.Save() // flush the in-memory encrypted doc to disk first
	}
	data, err := os.ReadFile(docPath())
	if err != nil {
		dialog.ShowError(err, m.win)
		return
	}
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, m.win)
			return
		}
		if w == nil {
			return
		}
		uri := w.URI()
		if _, e := w.Write(data); e != nil {
			_ = w.Close()
			dialog.ShowError(e, m.win)
			return
		}
		_ = w.Close()
		m.sourceURI = uri // keep syncing to the file we just saved
		dialog.ShowInformation("Exported", "Saved a copy — changes will now sync to it while the app is open.", m.win)
	}, m.win)
	d.SetFileName("financy.financy")
	d.Show()
}

// syncBack copies the current document back to the linked external file. It runs
// asynchronously and coalesces bursts of changes, so cloud-provider I/O never
// blocks the UI. If the OS has revoked access (e.g. after a restart), it quietly
// unlinks — the sandbox copy is still saved either way.
func (m *mobileApp) syncBack() {
	if m.sourceURI == nil {
		return
	}
	m.syncMu.Lock()
	if m.syncPending {
		m.syncMu.Unlock()
		return
	}
	m.syncPending = true
	m.syncMu.Unlock()

	go func() {
		time.Sleep(700 * time.Millisecond) // coalesce a burst of edits
		m.syncMu.Lock()
		m.syncPending = false
		uri := m.sourceURI
		m.syncMu.Unlock()
		if uri == nil {
			return
		}
		data, err := os.ReadFile(docPath())
		if err != nil {
			return
		}
		w, err := storage.Writer(uri)
		if err != nil {
			m.sourceURI = nil // access lost; fall back to sandbox-only
			return
		}
		defer func() { _ = w.Close() }()
		_, _ = w.Write(data)
	}()
}

func friendlyOpenErr(e error) error {
	switch {
	case errors.Is(e, core.ErrFileTooNew):
		return errors.New("this file was made by a newer version of Financy — update the app first")
	case errors.Is(e, core.ErrBadPassphrase):
		return errors.New("wrong passphrase, or the file is damaged")
	default:
		return e
	}
}
