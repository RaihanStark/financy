package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// showForm presents a modal form; onConfirm runs only when the user saves.
func showForm(title string, items []*widget.FormItem, onConfirm func()) {
	if win == nil {
		return
	}
	d := dialog.NewForm(title, "Save", "Cancel", items, func(ok bool) {
		if ok {
			onConfirm()
		}
	}, win)
	d.Resize(fyne.NewSize(460, 420))
	d.Show()
}

// showContextMenu pops up a desktop-style menu at the given screen position.
func showContextMenu(pos fyne.Position, items ...*fyne.MenuItem) {
	if win == nil {
		return
	}
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), win.Canvas(), pos)
}

func showInfo(title, msg string) {
	if win == nil {
		return
	}
	dialog.ShowInformation(title, msg, win)
}

func confirmDelete(what string, onYes func()) {
	if win == nil {
		return
	}
	dialog.ShowConfirm("Delete", "Delete "+what+"? This cannot be undone.", func(ok bool) {
		if ok {
			onYes()
		}
	}, win)
}
