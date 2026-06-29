package view

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// completionEntry is a text entry that pops up a live-filtered dropdown of known
// values as the user types. It only *suggests* — the user is always free to keep
// typing a value that isn't in the list (e.g. a brand-new payee), and simply
// ignoring the dropdown commits whatever was typed.
type completionEntry struct {
	widget.Entry

	options []string // the full suggestion pool
	matches []string // current filtered matches shown in the popup

	popup    *widget.PopUp
	list     *widget.List
	suppress bool // guards re-entrant OnChanged fired by our own SetText
}

// newCompletionEntry builds an autocomplete entry over the given suggestions.
func newCompletionEntry(options []string) *completionEntry {
	c := &completionEntry{options: options}
	c.ExtendBaseWidget(c)
	c.OnChanged = c.handleChanged
	return c
}

func (c *completionEntry) handleChanged(s string) {
	if c.suppress {
		return
	}
	c.refresh(s)
}

// refresh recomputes the matches for the current text and shows or hides the
// dropdown accordingly.
func (c *completionEntry) refresh(s string) {
	q := strings.ToLower(strings.TrimSpace(s))
	c.matches = c.matches[:0]
	if q != "" {
		for _, o := range c.options {
			ol := strings.ToLower(o)
			if ol == q { // already typed in full — nothing useful to suggest
				continue
			}
			if strings.Contains(ol, q) {
				c.matches = append(c.matches, o)
			}
		}
	}
	if len(c.matches) == 0 {
		c.hide()
		return
	}
	c.show()
}

func (c *completionEntry) ensureList() {
	if c.list != nil {
		return
	}
	c.list = widget.NewList(
		func() int { return len(c.matches) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i >= 0 && i < len(c.matches) {
				o.(*widget.Label).SetText(c.matches[i])
			}
		},
	)
	c.list.OnSelected = func(i widget.ListItemID) {
		if i >= 0 && i < len(c.matches) {
			c.commit(c.matches[i])
		}
	}
}

func (c *completionEntry) show() {
	canvas := fyne.CurrentApp().Driver().CanvasForObject(c)
	if canvas == nil {
		return
	}
	c.ensureList()
	c.list.UnselectAll()
	c.list.Refresh()

	visible := len(c.matches)
	if visible > 6 {
		visible = 6
	}
	itemH := widget.NewLabel("X").MinSize().Height + theme.Padding()*2
	size := fyne.NewSize(c.Size().Width, itemH*float32(visible)+theme.Padding()*2)

	if c.popup == nil {
		c.popup = widget.NewPopUp(c.list, canvas)
	}
	c.popup.Resize(size)
	c.popup.ShowAtPosition(c.popupPos())
}

func (c *completionEntry) popupPos() fyne.Position {
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(c)
	return pos.Add(fyne.NewPos(0, c.Size().Height))
}

func (c *completionEntry) hide() {
	if c.popup != nil {
		c.popup.Hide()
	}
}

// commit fills the entry with the chosen suggestion and dismisses the dropdown
// without retriggering filtering.
func (c *completionEntry) commit(s string) {
	c.suppress = true
	c.SetText(s)
	c.CursorColumn = len([]rune(s))
	c.suppress = false
	c.hide()
	c.Refresh()
}

// TypedKey lets Escape dismiss the dropdown and keeps Tab from leaving it open.
func (c *completionEntry) TypedKey(k *fyne.KeyEvent) {
	if c.popup != nil && c.popup.Visible() {
		switch k.Name {
		case fyne.KeyEscape:
			c.hide()
			return
		case fyne.KeyTab:
			c.hide()
		}
	}
	c.Entry.TypedKey(k)
}
