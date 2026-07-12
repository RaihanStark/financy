package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/ui/component"
)

// Shell-level modal helpers mirroring the view package's — every dialog the
// chrome opens (setup, updates, passwords, alerts) uses the same rounded
// component.Modal card as the screens.

// newModal wraps content in a bare modal card; the content brings its own
// action row. Callers Show() it themselves.
func newModal(title string, content fyne.CanvasObject) *modal {
	return component.NewModal(ctl.win.Canvas(), title, content)
}

// newModalForm mirrors dialog.NewForm: labeled form items with validation —
// confirm only proceeds (and closes) when every item validates.
func newModalForm(title, confirmLabel, cancelLabel string, items []*widget.FormItem, cb func(bool)) *modal {
	form := widget.NewForm(items...)
	m := newModal(title, form)
	m.SetButtons(
		secondaryButton(cancelLabel, nil, func() {
			m.Hide()
			cb(false)
		}),
		primaryButton(confirmLabel, nil, func() {
			if form.Validate() != nil {
				return
			}
			m.Hide()
			cb(true)
		}),
	)
	return m
}

// wrapLabel is a word-wrapped message body for info/error modals.
func wrapLabel(msg string) fyne.CanvasObject {
	l := widget.NewLabel(msg)
	l.Wrapping = fyne.TextWrapWord
	return l
}

func showInfo(title, msg string) {
	if ctl == nil || ctl.win == nil {
		return
	}
	m := newModal(title, wrapLabel(msg))
	m.SetButtons(secondaryButton("OK", nil, func() { m.Hide() }))
	m.SetCardSize(fyne.NewSize(420, 0))
	m.Show()
}

// showError presents err in a modal alert — the in-app stand-in for
// dialog.ShowError.
func showError(err error) {
	if err == nil || ctl == nil || ctl.win == nil {
		return
	}
	m := newModal("Error", wrapLabel(err.Error()))
	m.SetButtons(secondaryButton("OK", nil, func() { m.Hide() }))
	m.SetCardSize(fyne.NewSize(420, 0))
	m.Show()
}

// showConfirm asks a yes/no question; onYes runs only on confirm.
func showConfirm(title, msg, confirmLabel string, onYes func()) {
	if ctl == nil || ctl.win == nil {
		return
	}
	m := newModal(title, wrapLabel(msg))
	m.SetButtons(
		secondaryButton("Cancel", nil, func() { m.Hide() }),
		primaryButton(confirmLabel, nil, func() {
			m.Hide()
			onYes()
		}),
	)
	m.SetCardSize(fyne.NewSize(420, 0))
	m.Show()
}
