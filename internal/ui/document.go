package ui

import (
	"errors"
	"os"
	"path/filepath"

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

// appTitle builds the window title, including the app version and open file.
func appTitle(path string) string {
	t := "Financy v" + core.Version
	if path != "" {
		t += " — " + filepath.Base(path)
	}
	return t
}

// useStore makes s the active document and refreshes the whole UI.
func useStore(s *core.Store, path string) {
	if store != nil {
		store.Close()
	}
	store = s
	docPath = path
	view.Init(store, ctl.show, ctl.refresh, ctl.win)
	if store != nil {
		store.Subscribe(ctl.refresh)
		store.SetErrorHandler(func(err error) {
			dialog.ShowError(err, ctl.win)
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
// current schema (migrations run on open).
func openDocumentAt(path string) {
	if v, err := core.FileVersion(path); err == nil && v < core.LatestVersion() {
		if data, e := os.ReadFile(path); e == nil {
			_ = os.WriteFile(path+".bak", data, 0o644)
		}
	}
	s, err := core.OpenStore(path)
	if err != nil {
		if errors.Is(err, core.ErrFileTooNew) {
			msg := filepath.Base(path) + " was created by a newer version of Financy than " +
				"this one (you're running v" + core.Version + ").\n\n" +
				"Update Financy to the latest version, then open the file again. " +
				"It has not been changed."
			dialog.ShowInformation("Can't open this file", msg, ctl.win)
			return
		}
		dialog.ShowError(err, ctl.win)
		return
	}
	useStore(s, path)
}

// ---- File menu actions ----

func doNew() { promptNewDocument() }

func doOpen() {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil || r == nil {
			return
		}
		path := r.URI().Path()
		r.Close()
		openDocumentAt(path)
	}, ctl.win)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".financy"}))
	d.Show()
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
		w.Close()
		_ = os.Remove(path) // VACUUM INTO requires a non-existent target
		if err := store.SaveCopy(path); err != nil {
			dialog.ShowError(err, ctl.win)
			return
		}
		dialog.ShowInformation("Saved a Copy", "Copy written to:\n"+path, ctl.win)
	}, ctl.win)
	d.SetFileName("copy.financy")
	d.Show()
}

func doClose() {
	useStore(nil, "")
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
	prefs = loadPrefs()
	if prefs.LastPath != "" {
		if _, err := os.Stat(prefs.LastPath); err == nil {
			openDocumentAt(prefs.LastPath)
			return
		}
	}
	// First run: render the empty backdrop and offer demo / scratch setup.
	ctl.renderWelcome()
	showSetup()
}
