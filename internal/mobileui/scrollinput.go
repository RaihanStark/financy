package mobileui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Fyne's mobile driver routes a touch-drag to the top-most Draggable/Touchable
// widget under the finger. A widget.Entry (draggable for text selection) or a
// widget.Select (touchable) therefore swallows a scroll gesture that happens to
// start on it, so the form won't scroll — it selects text or opens a dropdown
// instead. The wrappers below keep the fields inline but forward drags to the
// enclosing scroll, so a scroll gesture scrolls the page wherever it lands while
// a genuine tap still edits the field.

// scrollForwarder is a form control that can be told which scroll to drive.
type scrollForwarder interface {
	setScroll(*container.Scroll)
}

// forwardDrag scrolls s by a drag delta (Scrolled clamps to the content bounds).
func forwardDrag(s *container.Scroll, ev *fyne.DragEvent) {
	s.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DX: ev.Dragged.DX, DY: ev.Dragged.DY}})
}

// scrollEntry is a widget.Entry that forwards drags to its host scroll instead
// of starting a text selection.
type scrollEntry struct {
	widget.Entry
	scroll *container.Scroll
}

func newEntry() *scrollEntry {
	e := &scrollEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func newPasswordEntry() *scrollEntry {
	e := &scrollEntry{}
	e.Password = true
	e.ExtendBaseWidget(e)
	return e
}

func (e *scrollEntry) setScroll(s *container.Scroll) { e.scroll = s }

func (e *scrollEntry) Dragged(ev *fyne.DragEvent) {
	if e.scroll == nil {
		e.Entry.Dragged(ev) // not hosted in a form scroll → normal text selection
		return
	}
	forwardDrag(e.scroll, ev)
}

func (e *scrollEntry) DragEnd() {
	if e.scroll == nil {
		e.Entry.DragEnd()
	}
}

// scrollSelect is a widget.Select made draggable so a scroll gesture starting on
// it scrolls the form (a plain Select is only Touchable, which the driver drops).
type scrollSelect struct {
	widget.Select
	scroll *container.Scroll
}

func newSelect(options []string, changed func(string)) *scrollSelect {
	s := &scrollSelect{}
	s.Options = options
	s.OnChanged = changed
	s.ExtendBaseWidget(s)
	return s
}

func (s *scrollSelect) setScroll(sc *container.Scroll) { s.scroll = sc }

func (s *scrollSelect) Dragged(ev *fyne.DragEvent) {
	if s.scroll != nil {
		forwardDrag(s.scroll, ev)
	}
}

func (s *scrollSelect) DragEnd() {}

// wireScroll walks a form body and points every scroll-forwarding control at the
// scroll that hosts it, so drags on those fields scroll the page.
func wireScroll(o fyne.CanvasObject, s *container.Scroll) {
	switch v := o.(type) {
	case scrollForwarder:
		v.setScroll(s)
	case *fyne.Container:
		for _, c := range v.Objects {
			wireScroll(c, s)
		}
	}
}
