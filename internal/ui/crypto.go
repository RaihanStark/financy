package ui

import (
	"errors"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// ---- password dialogs ----

// askPassword prompts for a single passphrase (opening an existing encrypted
// document). onOK runs only when the user confirms with a non-empty entry.
func askPassword(title, message string, onOK func(pass string)) {
	if ctl == nil || ctl.win == nil {
		return
	}
	entry := widget.NewPasswordEntry()
	entry.SetPlaceHolder("Passphrase")
	var items []*widget.FormItem
	if message != "" {
		// NewForm has no subtitle slot; show the hint as a leading row.
		hint := widget.NewLabel(message)
		hint.Wrapping = fyne.TextWrapWord
		items = append(items, widget.NewFormItem("", hint))
	}
	items = append(items, widget.NewFormItem("Password", entry))
	d := dialog.NewForm(title, "Open", "Cancel", items, func(ok bool) {
		if ok && entry.Text != "" {
			onOK(entry.Text)
		}
	}, ctl.win)
	d.Resize(fyne.NewSize(440, 0))
	d.Show()
	ctl.win.Canvas().Focus(entry)
}

// askNewPassword prompts for a new passphrase with confirmation. confirmLabel is
// the affirmative button text (e.g. "Create", "Set Password"). onOK runs with
// the chosen passphrase once the two entries match and are non-empty.
func askNewPassword(title, confirmLabel string, onOK func(pass string)) {
	if ctl == nil || ctl.win == nil {
		return
	}
	p1 := widget.NewPasswordEntry()
	p1.SetPlaceHolder("Passphrase")
	p2 := widget.NewPasswordEntry()
	p2.SetPlaceHolder("Repeat passphrase")
	warn := widget.NewLabel("If you forget this passphrase, the file can't be recovered.")
	warn.Wrapping = fyne.TextWrapWord
	items := []*widget.FormItem{
		widget.NewFormItem("Password", p1),
		widget.NewFormItem("Confirm", p2),
		widget.NewFormItem("", warn),
	}
	d := dialog.NewForm(title, confirmLabel, "Cancel", items, func(ok bool) {
		if !ok {
			return
		}
		switch {
		case p1.Text == "":
			dialog.ShowError(errors.New("password can't be empty"), ctl.win)
		case p1.Text != p2.Text:
			dialog.ShowError(errors.New("the passwords don't match"), ctl.win)
		default:
			onOK(p1.Text)
		}
	}, ctl.win)
	d.Resize(fyne.NewSize(440, 0))
	d.Show()
	ctl.win.Canvas().Focus(p1)
}

// ---- opening encrypted documents ----

// openEncryptedAt prompts for the passphrase and opens an encrypted document,
// re-prompting on a wrong passphrase.
func openEncryptedAt(path string) {
	askPassword("Open encrypted file", filepath.Base(path)+" is password-protected.", func(pass string) {
		s, err := core.OpenStoreEncrypted(path, pass)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrBadPassphrase):
				dialog.ShowError(errors.New("incorrect password — please try again"), ctl.win)
				openEncryptedAt(path) // retry
			case errors.Is(err, core.ErrFileTooNew):
				dialog.ShowInformation("Can't open this file",
					filepath.Base(path)+" was created by a newer version of Financy than this one "+
						"(you're running v"+core.Version+"). Update Financy, then open it again.", ctl.win)
			default:
				dialog.ShowError(err, ctl.win)
			}
			return
		}
		useStore(s, path)
	})
}

// ---- save & unsaved-changes guard ----

// doSave writes an encrypted document to disk. It's a no-op for plain
// (auto-saved) documents.
func doSave() {
	if store == nil || !store.Encrypted() {
		return
	}
	if err := store.Save(); err != nil {
		dialog.ShowError(err, ctl.win)
		return
	}
	updateTitle()
}

// guardUnsaved runs proceed immediately unless the current document is an
// encrypted document with unsaved changes, in which case it asks the user to
// Save, Discard, or Cancel first.
func guardUnsaved(proceed func()) {
	if store == nil || !store.IsDirty() {
		proceed()
		return
	}
	var d *dialog.CustomDialog
	msg := txt("You have unsaved changes. Save before continuing?", colText, 13, false)
	save := primaryButton("Save", nil, func() {
		d.Hide()
		if err := store.Save(); err != nil {
			dialog.ShowError(err, ctl.win)
			return
		}
		updateTitle()
		proceed()
	})
	discard := secondaryButton("Discard", nil, func() { d.Hide(); proceed() })
	cancel := secondaryButton("Cancel", nil, func() { d.Hide() })
	content := container.NewVBox(msg, spacerH(14),
		container.NewCenter(container.NewHBox(cancel, discard, save)))
	d = dialog.NewCustomWithoutButtons("Unsaved changes", content, ctl.win)
	d.Show()
}

// ---- password management (File menu) ----

// doSetPassword adds or changes a document's password.
func doSetPassword() {
	if store == nil {
		return
	}
	title, label := "Set Password", "Set Password"
	if store.Encrypted() {
		title, label = "Change Password", "Change"
	}
	askNewPassword(title, label, func(pass string) {
		if err := store.SetPassword(pass); err != nil {
			dialog.ShowError(err, ctl.win)
			return
		}
		updateTitle()
		refreshMenu()
		dialog.ShowInformation(title, "This document is now protected by a password.", ctl.win)
	})
}

// doRemovePassword converts an encrypted document back to a plain file.
func doRemovePassword() {
	if store == nil || !store.Encrypted() {
		return
	}
	dialog.ShowConfirm("Remove Password",
		"Remove encryption from this document? It will be stored unencrypted on disk.",
		func(ok bool) {
			if !ok {
				return
			}
			if err := store.RemovePassword(); err != nil {
				dialog.ShowError(err, ctl.win)
				return
			}
			updateTitle()
			refreshMenu()
		}, ctl.win)
}

// updateTitle refreshes the window title so the unsaved-changes marker tracks
// the document's dirty state.
func updateTitle() {
	if ctl != nil && ctl.win != nil {
		ctl.win.SetTitle(appTitle(docPath))
	}
}
