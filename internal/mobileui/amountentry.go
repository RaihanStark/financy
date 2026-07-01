package mobileui

import (
	"github.com/raihanstark/financy/internal/core"
)

// newAmountEntry returns a money input that live-formats with thousands grouping
// as the user types, following the document's number format (e.g. 1,000 vs
// 1.000). Prefill it with core.FmtMoneyInput(minor); read it with
// core.ParseAmount(entry.Text). Like every form field it forwards scroll-drags
// to its host scroll (see scrollEntry).
func newAmountEntry() *scrollEntry {
	e := newEntry()
	e.SetPlaceHolder("0")
	var busy bool // guard the SetText-triggered re-entry into OnChanged
	e.OnChanged = func(s string) {
		if busy {
			return
		}
		g := core.GroupTyped(s)
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
