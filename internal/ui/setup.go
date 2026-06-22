package ui

import (
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

// showSetup is the first-run welcome/setup dialog: choose demo data or a fresh
// document with a chosen currency. "Get Started" then prompts for where to save
// the new file.
func showSetup() {
	if ctl == nil || ctl.win == nil {
		return
	}

	mode := widget.NewRadioGroup([]string{setupDemo, setupScratch}, nil)
	mode.SetSelected(setupDemo)

	currency := widget.NewSelect([]string{"Rp", "$", "€", "£"}, nil)
	currency.SetSelected("Rp")
	currency.Disable() // only relevant for "start from scratch"

	mode.OnChanged = func(v string) {
		if v == setupScratch {
			currency.Enable()
		} else {
			currency.Disable()
		}
	}

	body := container.NewVBox(
		txt("Welcome to Financy", colText, 20, true),
		txt("Choose how to begin, then pick where to save your file.", colTextDim, 12, false),
		spacerH(10),
		mode,
		spacerH(4),
		container.NewBorder(nil, nil,
			container.NewGridWrap(fyne.NewSize(80, 32), txt("Currency", colTextDim, 12, false)),
			nil, currency),
	)

	var d *dialog.CustomDialog
	cont := primaryButton("Continue…", nil, func() {
		demo := mode.Selected == setupDemo
		cur := currency.Selected
		d.Hide()
		promptCreateDocument(demo, cur)
	})

	content := container.NewVBox(body, spacerH(16), container.NewCenter(cont))
	d = dialog.NewCustomWithoutButtons("Setup", content, ctl.win)
	d.Resize(fyne.NewSize(480, 340))
	d.Show()
}

// promptCreateDocument asks where to save, then creates the new document.
func promptCreateDocument(demo bool, currency string) {
	fd := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			// Cancelled — leave the welcome state so the user can retry via File.
			return
		}
		path := w.URI().Path()
		w.Close() // creates an empty file; NewDocument initializes the schema

		s, e := core.NewDocument(path)
		if e != nil {
			dialog.ShowError(e, ctl.win)
			return
		}
		if demo {
			core.SeedDemo(s, "$")
		} else {
			s.SetSettings("", currency, s.Year())
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
