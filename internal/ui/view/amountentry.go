package view

import "fyne.io/fyne/v2/widget"

// newAmountEntry returns a money input that live-formats with thousands grouping
// as the user types, following the document's number format (e.g. 1,000 / 1.000).
// Callers attach their own Validator. Prefill it with fmtMoneyInput(minor).
func newAmountEntry() *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder("0")
	var busy bool // guard the SetText-triggered re-entry into OnChanged
	e.OnChanged = func(s string) {
		if busy {
			return
		}
		g := groupTyped(s)
		if g == s {
			return
		}
		busy = true
		e.SetText(g)
		e.CursorColumn = len([]rune(g)) // keep the caret at the end while typing
		busy = false
		e.Refresh()
	}
	return e
}
