package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
	"github.com/raihanstark/financy/internal/ui/view"
)

var (
	prefs   *appPrefs
	docPath string
)

// appTitle builds the window title, including the app version and open file. An
// encrypted document with unsaved changes is marked with a leading bullet.
func appTitle(path string) string {
	t := "Financy v" + core.Version
	if path != "" {
		name := filepath.Base(path)
		if store != nil && store.IsDirty() {
			name = "• " + name
		}
		t += " — " + name
	}
	return t
}

// useStore makes s the active document and refreshes the whole UI.
func useStore(s *core.Store, path string) {
	if store != nil {
		_ = store.Close()
	}
	store = s
	docPath = path
	view.Init(store, ctl.show, ctl.refresh, ctl.win)
	if store != nil {
		store.Subscribe(ctl.refresh)
		store.Subscribe(updateTitle) // keep the unsaved-changes marker current
		store.SetErrorHandler(func(err error) {
			showError(err)
		})
	}
	if ctl.win != nil {
		ctl.win.SetTitle(appTitle(path))
		refreshMenu()
	}
	if store == nil {
		docPath = ""
		ctl.currentUID = ""
		ctl.renderWelcome()
		ctl.updateStatus()
		return
	}
	if prefs != nil {
		prefs.remember(path)
	}
	ctl.show("accounts")
	view.CheckRecurringDue() // prompt if any recurring entries are due
}

// openDocumentAt opens an existing file, backing it up first if it predates the
// current schema (migrations run on open). The backup's name carries the old
// schema version (e.g. budget.financy.v6.bak) so the user knows which Financy
// version still reads it if they ever need to downgrade.
func openDocumentAt(path string) {
	if enc, err := core.IsEncrypted(path); err != nil {
		showError(err)
		return
	} else if enc {
		openEncryptedAt(path)
		return
	}
	backupName := ""
	backupVersion := 0
	if v, err := core.FileVersion(path); err == nil && v < core.LatestVersion() {
		if data, e := os.ReadFile(path); e == nil {
			backup := path + core.BackupSuffix(v)
			if os.WriteFile(backup, data, 0o644) == nil {
				backupName = filepath.Base(backup)
				backupVersion = v
			}
		}
	}
	s, err := core.OpenStore(path)
	if err != nil {
		if errors.Is(err, core.ErrFileTooNew) {
			msg := filepath.Base(path) + " was created by a newer version of Financy than " +
				"this one (you're running v" + core.Version + ").\n\n" +
				"Update Financy to the latest version, then open the file again. " +
				"It has not been changed."
			showInfo("Can't open this file", msg)
			return
		}
		showError(err)
		return
	}
	useStore(s, path)
	if backupName != "" {
		msg := "This file was upgraded to a newer format. A backup of the previous " +
			"version was saved as:\n\n" + backupName + "\n\n"
		if rel := core.ReleaseForVersion(backupVersion); rel != "" {
			msg += "To downgrade, open that backup with Financy " + rel + " or later " +
				"(earlier versions can't read this format)."
		} else {
			msg += "Keep it if you may want to downgrade to an older version of Financy."
		}
		showInfo("File upgraded", msg)
	}
}

// ---- File menu actions ----

func doNew() { guardUnsaved(promptNewDocument) }

func doOpen() {
	guardUnsaved(func() {
		d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil || r == nil {
				return
			}
			path := r.URI().Path()
			_ = r.Close()
			openDocumentAt(path)
		}, ctl.win)
		d.SetFilter(storage.NewExtensionFileFilter([]string{".financy"}))
		d.Show()
	})
}

func doSaveCopy() {
	if store == nil {
		return
	}
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			return
		}
		path := w.URI().Path()
		_ = w.Close()
		_ = os.Remove(path) // VACUUM INTO requires a non-existent target
		if err := store.SaveCopy(path); err != nil {
			showError(err)
			return
		}
		showInfo("Saved a Copy", "Copy written to:\n"+path)
	}, ctl.win)
	d.SetFileName("copy.financy")
	d.Show()
}

// doExportCSV writes all transactions to a CSV the user chooses.
func doExportCSV() {
	if store == nil {
		return
	}
	data, err := store.ExportCSV()
	if err != nil {
		showError(err)
		return
	}
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			return
		}
		defer func() { _ = w.Close() }()
		if _, err := w.Write(data); err != nil {
			showError(err)
			return
		}
		showInfo("Exported", "Transactions exported to:\n"+w.URI().Path())
	}, ctl.win)
	d.SetFileName(exportFileName())
	d.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
	d.Show()
}

func exportFileName() string {
	base := "financy"
	if docPath != "" {
		base = strings.TrimSuffix(filepath.Base(docPath), ".financy")
	}
	return base + "-export.csv"
}

func doClose() {
	guardUnsaved(func() { useStore(nil, "") })
}

// ---- welcome / empty state ----

func welcomeScreen() fyne.CanvasObject {
	title := txt("No file open", colText, 20, true)
	hint := txt("Create a new document or open an existing .financy file.", colTextDim, 12, false)

	newBtn := primaryButton("New File…", nil, doNew)
	openBtn := secondaryButton("Open…", nil, doOpen)
	buttons := container.NewHBox(newBtn, openBtn)

	box := container.NewVBox(title, hint, spacerH(8), buttons)

	if prefs != nil && len(prefs.Recent) > 0 {
		box.Add(spacerH(16))
		box.Add(sectionTitle("Recent"))
		for _, r := range prefs.Recent {
			r := r
			b := widget.NewButton(filepath.Base(r), func() { openDocumentAt(r) })
			b.Alignment = widget.ButtonAlignLeading
			b.Importance = widget.LowImportance
			box.Add(b)
		}
	}

	return container.NewCenter(panel(container.New(padCell(16, 24), box)))
}

// startup opens the last-used file, or shows the first-run setup dialog.
func startup() {
	if prefs == nil {
		prefs = loadPrefs()
	}
	if prefs.LastPath != "" {
		if _, err := os.Stat(prefs.LastPath); err == nil {
			// Render the welcome backdrop first so that if opening needs a prompt
			// (an encrypted file's passphrase) and the user cancels, they land on a
			// usable screen rather than a blank window.
			ctl.renderWelcome()
			openDocumentAt(prefs.LastPath)
			return
		}
	}
	// First run: render the empty backdrop and offer demo / scratch setup.
	ctl.renderWelcome()
	showSetup()
}
