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
// in a floating popup beneath itself as the user types. It only *suggests* — the
// user is always free to keep typing a value that isn't listed (e.g. a brand-new
// payee), and ignoring the popup simply commits whatever was typed.
//
// The suggestions float in a canvas overlay so they sit on top of the
// surrounding UI instead of pushing it around. A canvas overlay takes over the
// keyboard focus manager, which would normally stop keystrokes from reaching the
// entry, so while the popup is open we keep the entry focused and forward typed
// runes/keys to it through the canvas' typed-event fallback.
type completionEntry struct {
	widget.BaseWidget

	input *completionInput
	box   *fyne.Container // vertical stack of suggestionRow, hosted by popup
	popup *widget.PopUp

	options  []string
	matches  []string
	sel      int  // highlighted row index, or -1 for none
	open     bool // whether the popup is currently shown
	suppress bool // guards re-entrant OnChanged fired by our own SetText

	fwd      bool // whether the typed-event forwarders are installed
	prevRune func(rune)
	prevKey  func(*fyne.KeyEvent)
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
	c.input.onFocusLost = c.closeList
	c.ExtendBaseWidget(c)
	return c
}

// CreateRenderer renders only the entry — the suggestion popup lives in a canvas
// overlay, so the widget keeps the height of a plain field and never reflows the
// form when the dropdown opens.
func (c *completionEntry) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.input)
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
	cnv := fyne.CurrentApp().Driver().CanvasForObject(c.input)
	if cnv == nil {
		return
	}

	rows := make([]fyne.CanvasObject, len(c.matches))
	for i, m := range c.matches {
		m := m // capture
		rows[i] = newSuggestionRow(m, func() { c.commit(m) })
	}
	c.box.Objects = rows
	c.box.Refresh()

	if c.popup == nil {
		c.popup = widget.NewPopUp(c.box, cnv)
	}
	m := c.box.MinSize()
	w := c.input.Size().Width
	if m.Width > w {
		w = m.Width
	}
	c.popup.Resize(fyne.NewSize(w, m.Height))
	c.popup.ShowAtPosition(c.popupPos())
	c.open = true

	c.installForwarder(cnv)
	// Keep the entry focused so the caret stays visible; typed events then route
	// to it through the forwarders below.
	cnv.Focus(c.input)
}

func (c *completionEntry) popupPos() fyne.Position {
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(c.input)
	return pos.Add(fyne.NewPos(0, c.input.Size().Height))
}

func (c *completionEntry) closeList() {
	if !c.open {
		return
	}
	c.open = false
	if c.popup != nil {
		c.popup.Hide()
	}
	c.uninstallForwarder()
}

// commit fills the entry with the chosen suggestion, closes the popup and keeps
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
	if !c.open {
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

// ---- typed-event forwarding ----
//
// While the overlay popup is open the canvas focus manager belongs to the
// overlay, so the entry no longer receives keystrokes directly. We install
// canvas-level fallbacks that funnel typing back into the entry, and tear them
// down again as soon as the popup goes away.

func (c *completionEntry) installForwarder(cnv fyne.Canvas) {
	if c.fwd {
		return
	}
	c.prevRune = cnv.OnTypedRune()
	c.prevKey = cnv.OnTypedKey()
	cnv.SetOnTypedRune(c.fwdRune)
	cnv.SetOnTypedKey(c.fwdKey)
	c.fwd = true
}

func (c *completionEntry) uninstallForwarder() {
	if !c.fwd {
		return
	}
	c.fwd = false
	if cnv := fyne.CurrentApp().Driver().CanvasForObject(c.input); cnv != nil {
		cnv.SetOnTypedRune(c.prevRune)
		cnv.SetOnTypedKey(c.prevKey)
	}
	c.prevRune, c.prevKey = nil, nil
}

func (c *completionEntry) fwdRune(r rune) {
	if c.open {
		c.input.TypedRune(r)
		return
	}
	prev := c.prevRune
	c.uninstallForwarder()
	if prev != nil {
		prev(r)
	}
}

func (c *completionEntry) fwdKey(k *fyne.KeyEvent) {
	if c.open {
		c.input.TypedKey(k)
		return
	}
	prev := c.prevKey
	c.uninstallForwarder()
	if prev != nil {
		prev(k)
	}
}

// completionInput is the editable field of a completionEntry. It forwards key
// events and focus loss to the parent so the popup can be driven and dismissed.
type completionInput struct {
	widget.Entry
	onKey       func(*fyne.KeyEvent) bool
	onFocusLost func()
}

func (e *completionInput) TypedKey(k *fyne.KeyEvent) {
	if e.onKey != nil && e.onKey(k) {
		return
	}
	e.Entry.TypedKey(k)
}

func (e *completionInput) FocusLost() {
	e.Entry.FocusLost()
	if e.onFocusLost != nil {
		e.onFocusLost()
	}
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
