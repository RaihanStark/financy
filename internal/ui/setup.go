package ui

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

const (
	setupDemo    = "Load demo data (USD)"
	setupScratch = "Start from scratch"
)

// labeledRow lays a fixed-width caption beside a field, matching the setup form.
func labeledRow(label string, field fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil,
		container.NewGridWrap(fyne.NewSize(80, 32), txt(label, colTextDim, 12, false)),
		nil, field)
}

// showSetup is the first-run welcome/setup dialog: choose demo data or a fresh
// document with a chosen currency and optional password. "Continue" then prompts
// for where to save the new file.
func showSetup() {
	if ctl == nil || ctl.win == nil {
		return
	}

	mode := widget.NewRadioGroup([]string{setupDemo, setupScratch}, nil)
	mode.SetSelected(setupDemo)

	currency := widget.NewSelect([]string{"Rp", "$", "€", "£"}, nil)
	currency.SetSelected("Rp")
	currency.Disable() // only relevant for "start from scratch"

	pw := widget.NewPasswordEntry()
	pw.SetPlaceHolder("Optional — leave blank for no encryption")
	pw.Disable()
	pw2 := widget.NewPasswordEntry()
	pw2.SetPlaceHolder("Repeat passphrase")
	pw2.Disable()

	mode.OnChanged = func(v string) {
		if v == setupScratch {
			currency.Enable()
			pw.Enable()
			pw2.Enable()
		} else {
			currency.Disable()
			pw.Disable()
			pw2.Disable()
		}
	}

	body := container.NewVBox(
		txt("Welcome to Financy", colText, 20, true),
		txt("Choose how to begin, then pick where to save your file.", colTextDim, 12, false),
		spacerH(10),
		mode,
		spacerH(4),
		labeledRow("Currency", currency),
		labeledRow("Password", pw),
		labeledRow("Confirm", pw2),
	)

	var d *dialog.CustomDialog
	cont := primaryButton("Continue…", nil, func() {
		demo := mode.Selected == setupDemo
		cur := currency.Selected
		pass := ""
		if !demo {
			if pw.Text != pw2.Text {
				dialog.ShowError(errors.New("the passwords don't match"), ctl.win)
				return
			}
			pass = pw.Text
		}
		d.Hide()
		promptCreateDocument(demo, cur, pass)
	})
	cancel := secondaryButton("Close", nil, func() { d.Hide() })

	content := container.NewVBox(body, spacerH(16), container.NewCenter(container.NewHBox(cancel, cont)))
	d = dialog.NewCustomWithoutButtons("Setup", content, ctl.win)
	d.Resize(fyne.NewSize(480, 420))
	d.Show()
}

// promptNewDocument is the File ▸ New flow: ask for currency and an optional
// password for the new (empty) document, then continue to the save dialog.
func promptNewDocument() {
	if ctl == nil || ctl.win == nil {
		return
	}

	currency := widget.NewSelect([]string{"Rp", "$", "€", "£"}, nil)
	currency.SetSelected("$")

	pw := widget.NewPasswordEntry()
	pw.SetPlaceHolder("Optional — leave blank for no encryption")
	pw2 := widget.NewPasswordEntry()
	pw2.SetPlaceHolder("Repeat passphrase")

	body := container.NewVBox(
		txt("New document", colText, 18, true),
		txt("Choose the currency and an optional password, then pick where to save it.", colTextDim, 12, false),
		spacerH(10),
		labeledRow("Currency", currency),
		labeledRow("Password", pw),
		labeledRow("Confirm", pw2),
	)

	var d *dialog.CustomDialog
	cont := primaryButton("Continue…", nil, func() {
		if pw.Text != pw2.Text {
			dialog.ShowError(errors.New("the passwords don't match"), ctl.win)
			return
		}
		cur := currency.Selected
		pass := pw.Text
		d.Hide()
		promptCreateDocument(false, cur, pass)
	})
	cancel := secondaryButton("Cancel", nil, func() { d.Hide() })

	content := container.NewVBox(body, spacerH(16), container.NewCenter(container.NewHBox(cancel, cont)))
	d = dialog.NewCustomWithoutButtons("New", content, ctl.win)
	d.Resize(fyne.NewSize(460, 320))
	d.Show()
}

// promptCreateDocument asks where to save, then creates the new document —
// encrypted when a passphrase was supplied.
func promptCreateDocument(demo bool, currency, passphrase string) {
	fd := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			// Cancelled — leave the welcome state so the user can retry via File.
			return
		}
		path := w.URI().Path()
		_ = w.Close() // creates an empty file; the constructors initialize it

		var (
			s *core.Store
			e error
		)
		if passphrase != "" {
			s, e = core.NewEncryptedDocument(path, passphrase)
		} else {
			s, e = core.NewDocument(path)
		}
		if e != nil {
			dialog.ShowError(e, ctl.win)
			return
		}
		if demo {
			core.SeedDemo(s, "$")
		} else {
			s.SetSettings("", currency, s.Year())
		}
		if s.Encrypted() {
			// Persist the seed/currency so the encrypted file on disk is complete.
			if err := s.Save(); err != nil {
				dialog.ShowError(err, ctl.win)
				return
			}
		}
		useStore(s, path)
	}, ctl.win)
	if demo {
		fd.SetFileName("demo.financy")
	} else {
		fd.SetFileName("finances.financy")
	}
	fd.Show()
}
