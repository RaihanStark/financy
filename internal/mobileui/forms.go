package mobileui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// kbAdjust wraps the scrolling field area of a form. When the on-screen keyboard
// appears, Fyne shrinks the content but does not scroll to the focused field, so
// we do: whenever our height shrinks, scroll the currently focused widget above
// the keyboard. Driving this off the resize — not taps or focus events — makes it
// fire reliably however the field was focused, and it works for ANY focused entry
// on the page (no per-field registration needed).
type kbAdjust struct {
	win    fyne.Window
	scroll *container.Scroll
	prevH  float32
}

func (k *kbAdjust) MinSize(objs []fyne.CanvasObject) fyne.Size {
	if len(objs) == 0 {
		return fyne.Size{}
	}
	return objs[0].MinSize()
}

func (k *kbAdjust) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if len(objs) == 0 {
		return
	}
	objs[0].Resize(size)
	objs[0].Move(fyne.NewPos(0, 0))

	if size.Height+1 < k.prevH { // shrank → keyboard appeared
		if f := k.win.Canvas().Focused(); f != nil {
			if obj, ok := f.(fyne.CanvasObject); ok {
				// The focused field's offset within the scrolled content = its
				// absolute Y minus the scroll's absolute Y, plus however far we're
				// already scrolled. Using absolute positions keeps this correct
				// however deeply the field is nested (e.g. inside a bulk-add row
				// card). The -24 leaves a little room so the field's caption stays
				// visible above it.
				drv := fyne.CurrentApp().Driver()
				y := maxF(0, drv.AbsolutePositionForObject(obj).Y-
					drv.AbsolutePositionForObject(k.scroll).Y+k.scroll.Offset.Y-24)
				fyne.Do(func() { // defer off the layout pass to avoid re-entrant refresh
					k.scroll.Offset = fyne.NewPos(0, y)
					k.scroll.Refresh()
				})
			}
		}
	}
	k.prevH = size.Height
}

// formPage is the shared full-screen page template used for every mobile
// "modal" (add transaction, unlock, settings, confirmations). It frames a
// scrollable body with a fixed top bar: a leading ✕ (onClose), a centered
// title, and an optional trailing action button (action=="" hides it). Any
// focused text field in the body automatically scrolls above the keyboard.
func (m *mobileApp) formPage(title, action string, body fyne.CanvasObject, onAction, onClose func()) fyne.CanvasObject {
	scroll := container.NewVScroll(insets(body, 14, 14, 18, 18))
	wireScroll(body, scroll) // let fields forward scroll-drags to this scroll
	middleObj := container.New(&kbAdjust{win: m.win, scroll: scroll}, scroll)

	closeBtn := iconButton(theme.CancelIcon(), onClose)
	var trailing fyne.CanvasObject
	if action != "" && onAction != nil {
		b := widget.NewButton(action, onAction)
		b.Importance = widget.HighImportance
		trailing = b
	}
	titleT := newText(title, colInk, 17, true)
	titleT.Alignment = fyne.TextAlignCenter

	// Center the title across the FULL bar width and overlay the buttons, so the
	// title stays centered whether or not a trailing action is present.
	buttons := container.NewBorder(nil, nil, closeBtn, trailing)
	bar := insets(container.NewStack(container.NewCenter(titleT), buttons), 6, 6, 8, 8)
	head := container.NewStack(canvas.NewRectangle(colCard), container.NewVBox(bar, thinLine()))

	middle := container.NewStack(canvas.NewRectangle(colBg), middleObj)
	return container.NewBorder(head, nil, nil, nil, middle)
}

// pushPage shows a full-screen page over the current content. The builder gets a
// close func (== back) so its controls can dismiss it; the Android Back button
// dismisses it too.
func (m *mobileApp) pushPage(build func(close func()) fyne.CanvasObject) {
	m.pushView(build(m.back))
}

// drilldownBar is the fixed top bar for drill-down detail pages (account,
// transaction): a leading back arrow (Back button) and a centered title.
func (m *mobileApp) drilldownBar(title string) fyne.CanvasObject {
	backBtn := iconButton(theme.NavigateBackIcon(), m.back)
	titleT := newText(ellipsize(title, 24), colInk, 17, true)
	titleT.Alignment = fyne.TextAlignCenter
	buttons := container.NewBorder(nil, nil, backBtn, nil)
	bar := insets(container.NewStack(container.NewCenter(titleT), buttons), 6, 6, 8, 8)
	return container.NewStack(canvas.NewRectangle(colCard), container.NewVBox(bar, thinLine()))
}

// showUnlockPage presents a full-screen passphrase prompt. open decrypts with the
// entered passphrase and returns the store; a wrong passphrase shows an error and
// keeps the page. onCancel runs if the user backs out.
func (m *mobileApp) showUnlockPage(title, subtitle string, open func(pass string) (*core.Store, error), onCancel func()) {
	pw := newPasswordEntry()
	pw.SetPlaceHolder("Passphrase")
	pwField := field("Passphrase", pw)

	body := container.NewVBox(
		wrapText(subtitle),
		gap(12),
		pwField,
	)

	action := func() {
		s, err := open(pw.Text)
		if err != nil {
			dialog.ShowError(friendlyOpenErr(err), m.win)
			return
		}
		m.useStore(s) // switches the window to the shell
	}
	cancel := func() {
		m.back()
		if onCancel != nil {
			onCancel()
		}
	}
	m.pushView(m.formPage(title, "Open", body, action, cancel))
}
