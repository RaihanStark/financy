package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/ui/component"
)

// All modal surfaces go through component.Modal — the rounded, scrimmed card
// that replaced Fyne's boxy stock dialogs. The helpers here mirror the old
// dialog.* semantics so screens read the same as before.

// newModal wraps content in a bare modal card; the content brings its own
// action row. Callers Show() it themselves.
func newModal(title string, content fyne.CanvasObject) *modal {
	return component.NewModal(win.Canvas(), title, content)
}

// newModalWithClose is a read-only modal with a single dismiss button.
func newModalWithClose(title, closeLabel string, content fyne.CanvasObject) *modal {
	m := newModal(title, content)
	m.SetButtons(secondaryButton(closeLabel, nil, func() { m.Hide() }))
	return m
}

// newModalConfirm mirrors dialog.NewCustomConfirm: cancel/confirm buttons in
// the footer, cb(true) on confirm, cb(false) on cancel.
func newModalConfirm(title, confirmLabel, cancelLabel string, content fyne.CanvasObject, cb func(bool)) *modal {
	m := newModal(title, content)
	m.SetButtons(
		secondaryButton(cancelLabel, nil, func() {
			m.Hide()
			cb(false)
		}),
		primaryButton(confirmLabel, nil, func() {
			m.Hide()
			cb(true)
		}),
	)
	return m
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

// showForm presents a modal form; onConfirm runs only when the user saves.
func showForm(title string, items []*widget.FormItem, onConfirm func()) {
	if win == nil {
		return
	}
	m := newModalForm(title, "Save", "Cancel", items, func(ok bool) {
		if ok {
			onConfirm()
		}
	})
	m.SetCardSize(fyne.NewSize(460, 0))
	m.Show()
}

// dialogWithSave wraps arbitrary content in a Save/Cancel modal; onSave runs
// only when the user saves. The caller shows the returned modal.
func dialogWithSave(title string, content fyne.CanvasObject, onSave func()) *modal {
	m := newModalConfirm(title, "Save", "Cancel", content, func(ok bool) {
		if ok {
			onSave()
		}
	})
	m.SetCardSize(fyne.NewSize(440, 0))
	return m
}

// showDetail presents read-only content with just a Close button.
func showDetail(title string, content fyne.CanvasObject) {
	if win == nil {
		return
	}
	m := newModalWithClose(title, "Close", content)
	m.SetCardSize(fyne.NewSize(440, 0))
	m.Show()
}

// showContextMenu pops up a desktop-style menu at the given screen position.
func showContextMenu(pos fyne.Position, items ...*fyne.MenuItem) {
	if win == nil {
		return
	}
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), win.Canvas(), pos)
}

// showActionConfirm presents arbitrary content with a custom confirm/cancel
// action; onConfirm runs only when the user confirms.
func showActionConfirm(title, confirmLabel string, content fyne.CanvasObject, onConfirm func()) {
	if win == nil {
		return
	}
	m := newModalConfirm(title, confirmLabel, "Cancel", content, func(ok bool) {
		if ok {
			onConfirm()
		}
	})
	m.SetCardSize(fyne.NewSize(460, 0))
	m.Show()
}

// wrapLabel is a word-wrapped message body for info/error modals.
func wrapLabel(msg string) fyne.CanvasObject {
	l := widget.NewLabel(msg)
	l.Wrapping = fyne.TextWrapWord
	return l
}

func showInfo(title, msg string) {
	if win == nil {
		return
	}
	m := newModalWithClose(title, "OK", wrapLabel(msg))
	m.SetCardSize(fyne.NewSize(420, 0))
	m.Show()
}

// showError presents err in a modal alert — the in-app stand-in for
// dialog.ShowError.
func showError(err error) {
	if err == nil || win == nil {
		return
	}
	m := newModalWithClose("Error", "OK", wrapLabel(err.Error()))
	m.SetCardSize(fyne.NewSize(420, 0))
	m.Show()
}

func confirmDelete(what string, onYes func()) {
	if win == nil {
		return
	}
	m := newModal("Delete", wrapLabel("Delete "+what+"? This cannot be undone."))
	danger := widget.NewButton("Delete", nil)
	danger.Importance = widget.DangerImportance
	danger.OnTapped = func() {
		m.Hide()
		onYes()
	}
	m.SetButtons(secondaryButton("Cancel", nil, func() { m.Hide() }), danger)
	m.SetCardSize(fyne.NewSize(420, 0))
	m.Show()
}
