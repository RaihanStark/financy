package view

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// completionEntry is a text field that drops a live-filtered list of known values
// directly beneath itself as the user types. It only *suggests* — the user is
// always free to keep typing a value that isn't listed (e.g. a brand-new payee),
// and ignoring the list simply commits whatever was typed.
//
// The suggestion list is rendered inline (part of the normal widget tree) rather
// than in a floating overlay. An overlay would take over the canvas focus, which
// stops keystrokes from reaching the entry — so inline rendering is what keeps
// typing working while the dropdown is open.
type completionEntry struct {
	widget.BaseWidget

	input *completionInput
	box   *fyne.Container // vertical stack of suggestionRow, hidden when closed

	options  []string
	matches  []string
	sel      int  // highlighted row index, or -1 for none
	suppress bool // guards re-entrant OnChanged fired by our own SetText
}

const completionMaxRows = 8

func newCompletionEntry(options []string) *completionEntry {
	c := &completionEntry{
		input:   &completionInput{},
		box:     container.NewVBox(),
		options: options,
		sel:     -1,
	}
	c.input.ExtendBaseWidget(c.input)
	c.input.OnChanged = c.onChanged
	c.input.onKey = c.onKey
	c.box.Hide()
	c.ExtendBaseWidget(c)
	return c
}

func (c *completionEntry) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewVBox(c.input, c.box))
}

// ---- text proxy API used by the forms ----

// Text returns the current entry contents.
func (c *completionEntry) Text() string { return c.input.Text }

// SetText replaces the entry contents without opening the dropdown.
func (c *completionEntry) SetText(s string) {
	c.suppress = true
	c.input.SetText(s)
	c.suppress = false
	c.closeList()
}

// SetPlaceHolder sets the entry placeholder.
func (c *completionEntry) SetPlaceHolder(s string) { c.input.SetPlaceHolder(s) }

// ---- filtering ----

func (c *completionEntry) onChanged(s string) {
	if c.suppress {
		return
	}
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
				if len(c.matches) >= completionMaxRows {
					break
				}
			}
		}
	}
	c.sel = -1
	if len(c.matches) == 0 {
		c.closeList()
		return
	}
	c.openList()
}

func (c *completionEntry) openList() {
	rows := make([]fyne.CanvasObject, len(c.matches))
	for i, m := range c.matches {
		m := m // capture
		rows[i] = newSuggestionRow(m, func() { c.commit(m) })
	}
	c.box.Objects = rows
	c.box.Show()
	c.box.Refresh()
	c.Refresh()
}

func (c *completionEntry) closeList() {
	if !c.box.Visible() && len(c.box.Objects) == 0 {
		return
	}
	c.box.Hide()
	c.box.Objects = nil
	c.box.Refresh()
	c.Refresh()
}

// commit fills the entry with the chosen suggestion, closes the list and keeps
// focus on the entry so the user can carry on.
func (c *completionEntry) commit(s string) {
	c.suppress = true
	c.input.SetText(s)
	c.input.CursorColumn = len([]rune(s))
	c.suppress = false
	c.closeList()
	if cnv := fyne.CurrentApp().Driver().CanvasForObject(c.input); cnv != nil {
		cnv.Focus(c.input)
	}
}

// onKey handles dropdown navigation while it is open. It returns true when the
// key was consumed so the entry does not also act on it.
func (c *completionEntry) onKey(k *fyne.KeyEvent) bool {
	if !c.box.Visible() {
		return false
	}
	switch k.Name {
	case fyne.KeyEscape:
		c.closeList()
		return true
	case fyne.KeyDown:
		c.moveSel(1)
		return true
	case fyne.KeyUp:
		c.moveSel(-1)
		return true
	case fyne.KeyReturn, fyne.KeyEnter:
		if c.sel >= 0 && c.sel < len(c.matches) {
			c.commit(c.matches[c.sel])
			return true
		}
		c.closeList()
	case fyne.KeyTab:
		c.closeList()
	}
	return false
}

func (c *completionEntry) moveSel(d int) {
	n := len(c.matches)
	if n == 0 {
		return
	}
	c.sel += d
	if c.sel < 0 {
		c.sel = 0
	}
	if c.sel >= n {
		c.sel = n - 1
	}
	for i, o := range c.box.Objects {
		if row, ok := o.(*suggestionRow); ok {
			row.setHighlight(i == c.sel)
		}
	}
}

// completionInput is the editable field of a completionEntry. It forwards key
// events to the parent first so dropdown navigation can intercept them.
type completionInput struct {
	widget.Entry
	onKey func(*fyne.KeyEvent) bool
}

func (e *completionInput) TypedKey(k *fyne.KeyEvent) {
	if e.onKey != nil && e.onKey(k) {
		return
	}
	e.Entry.TypedKey(k)
}

// suggestionRow is one clickable, hover/keyboard-highlightable suggestion line.
type suggestionRow struct {
	widget.BaseWidget

	label     *widget.Label
	bg        *canvas.Rectangle
	onTap     func()
	hovered   bool
	highlight bool
}

func newSuggestionRow(text string, onTap func()) *suggestionRow {
	r := &suggestionRow{
		label: widget.NewLabel(text),
		bg:    canvas.NewRectangle(color.Transparent),
		onTap: onTap,
	}
	r.label.Truncation = fyne.TextTruncateEllipsis
	r.ExtendBaseWidget(r)
	return r
}

func (r *suggestionRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(r.bg, r.label))
}

func (r *suggestionRow) Tapped(*fyne.PointEvent) {
	if r.onTap != nil {
		r.onTap()
	}
}

func (r *suggestionRow) MouseIn(*desktop.MouseEvent)    { r.hovered = true; r.applyBG() }
func (r *suggestionRow) MouseMoved(*desktop.MouseEvent) {}
func (r *suggestionRow) MouseOut()                      { r.hovered = false; r.applyBG() }
func (r *suggestionRow) Cursor() desktop.Cursor         { return desktop.PointerCursor }

func (r *suggestionRow) setHighlight(b bool) { r.highlight = b; r.applyBG() }

func (r *suggestionRow) applyBG() {
	if r.hovered || r.highlight {
		r.bg.FillColor = colSurfaceHi
	} else {
		r.bg.FillColor = color.Transparent
	}
	r.bg.Refresh()
}
