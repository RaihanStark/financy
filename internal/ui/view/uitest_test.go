package view

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
	"github.com/raihanstark/financy/internal/ui/style"
)

// newViewTest wires the view package to a fresh, demo-seeded store and a real
// test window, so screen builders and dialogs behave exactly as in the running
// app. It returns the store and the window; package globals (store, win, render,
// nav) are set as a side effect and restored on test cleanup.
func newViewTest(t *testing.T) (*core.Store, fyne.Window) {
	t.Helper()
	a := test.NewApp()
	a.Settings().SetTheme(style.Theme{}) // real theme so text measurement works

	s := core.NewStore()
	core.SeedDemo(s, "$")
	core.SetCurrencySymbol("$")

	w := test.NewWindow(nil)
	w.Resize(fyne.NewSize(1280, 860))

	prevStore, prevWin, prevRender, prevNav := store, win, render, nav
	var navTarget string
	store = s
	win = w
	render = func() {}
	nav = func(uid string) { navTarget = uid }
	_ = navTarget

	t.Cleanup(func() {
		w.Close()
		store, win, render, nav = prevStore, prevWin, prevRender, prevNav
		core.SetCurrencySymbol("Rp")
	})
	return s, w
}

// emptyViewTest is like newViewTest but the store has no money accounts and no
// transactions (only the seed categories), for exercising empty-state paths.
func emptyViewTest(t *testing.T) (*core.Store, fyne.Window) {
	t.Helper()
	a := test.NewApp()
	a.Settings().SetTheme(style.Theme{}) // real theme so text measurement works

	s := core.NewStore() // categories only; no money accounts, no txns
	core.SetCurrencySymbol("$")

	w := test.NewWindow(nil)
	w.Resize(fyne.NewSize(1280, 860))

	prevStore, prevWin, prevRender, prevNav := store, win, render, nav
	store = s
	win = w
	render = func() {}
	nav = func(string) {}

	t.Cleanup(func() {
		w.Close()
		store, win, render, nav = prevStore, prevWin, prevRender, prevNav
		core.SetCurrencySymbol("Rp")
	})
	return s, w
}

// mount lays a screen out inside the test window so every widget has a renderer,
// then returns it. Collecting text only works reliably on mounted trees.
func mount(w fyne.Window, screen fyne.CanvasObject) fyne.CanvasObject {
	w.SetContent(screen)
	return screen
}

// collectText walks a (mounted) canvas-object tree and returns every visible
// string it finds in canvas.Text, Label, Button, Select, Entry and Hyperlink.
func collectText(obj fyne.CanvasObject) []string {
	var out []string
	seen := map[fyne.CanvasObject]bool{}
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || seen[o] {
			return
		}
		seen[o] = true
		switch v := o.(type) {
		case *canvas.Text:
			out = append(out, v.Text)
		case *widget.Label:
			out = append(out, v.Text)
		case *widget.Button:
			if v.Text != "" {
				out = append(out, v.Text)
			}
		case *widget.Select:
			if v.Selected != "" {
				out = append(out, v.Selected)
			}
			for _, c := range childObjects(v) {
				walk(c)
			}
		case *widget.Entry:
			if v.Text != "" {
				out = append(out, v.Text)
			}
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range childObjects(v) {
				walk(c)
			}
		}
	}
	walk(obj)
	return out
}

// childObjects renders a widget and returns its child objects, guarding against
// renderers that misbehave when the widget isn't fully laid out.
func childObjects(w fyne.Widget) (objs []fyne.CanvasObject) {
	defer func() { _ = recover() }()
	r := test.WidgetRenderer(w)
	if r == nil {
		return nil
	}
	return r.Objects()
}

// joined flattens collectText output for substring assertions.
func joined(obj fyne.CanvasObject) string { return strings.Join(collectText(obj), "") }

// assertText fails the test unless every wanted substring appears somewhere in
// the rendered tree.
func assertText(t *testing.T, obj fyne.CanvasObject, wants ...string) {
	t.Helper()
	hay := strings.Join(collectText(obj), "\n")
	for _, w := range wants {
		if !strings.Contains(hay, w) {
			t.Errorf("rendered tree missing %q\n--- text was ---\n%s", w, hay)
		}
	}
}

// assertContains fails unless sub appears in s (plain-string convenience).
func assertContains(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("%q does not contain %q", s, sub)
	}
}

// assertNoText fails if any unwanted substring is present.
func assertNoText(t *testing.T, obj fyne.CanvasObject, unwanted ...string) {
	t.Helper()
	hay := joined(obj)
	for _, u := range unwanted {
		if strings.Contains(hay, u) {
			t.Errorf("rendered tree unexpectedly contains %q", u)
		}
	}
}

// findButtons returns every *widget.Button in a tree.
func findButtons(obj fyne.CanvasObject) []*widget.Button {
	var out []*widget.Button
	seen := map[fyne.CanvasObject]bool{}
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || seen[o] {
			return
		}
		seen[o] = true
		if b, ok := o.(*widget.Button); ok {
			out = append(out, b)
		}
		switch v := o.(type) {
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range childObjects(v) {
				walk(c)
			}
		}
	}
	walk(obj)
	return out
}

// tapButton finds the first enabled button whose label contains want and taps
// it. It returns false if no such button exists.
func tapButton(obj fyne.CanvasObject, want string) bool {
	for _, b := range findButtons(obj) {
		if strings.Contains(b.Text, want) && !b.Disabled() {
			test.Tap(b)
			return true
		}
	}
	return false
}

// findSelects returns every *widget.Select in a tree, in walk order.
func findSelects(obj fyne.CanvasObject) []*widget.Select {
	var out []*widget.Select
	seen := map[fyne.CanvasObject]bool{}
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || seen[o] {
			return
		}
		seen[o] = true
		if s, ok := o.(*widget.Select); ok {
			out = append(out, s)
		}
		switch v := o.(type) {
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range childObjects(v) {
				walk(c)
			}
		}
	}
	walk(obj)
	return out
}

// findEntries returns every *widget.Entry in a tree, in walk order.
func findEntries(obj fyne.CanvasObject) []*widget.Entry {
	var out []*widget.Entry
	seen := map[fyne.CanvasObject]bool{}
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || seen[o] {
			return
		}
		seen[o] = true
		if e, ok := o.(*widget.Entry); ok {
			out = append(out, e)
		}
		switch v := o.(type) {
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range childObjects(v) {
				walk(c)
			}
		}
	}
	walk(obj)
	return out
}

// dialogContent returns the content tree of the topmost overlay (a shown
// dialog), or nil if no overlay is present.
func dialogContent(w fyne.Window) fyne.CanvasObject {
	return w.Canvas().Overlays().Top()
}
