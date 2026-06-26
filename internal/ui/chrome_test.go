package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// The welcome backdrop offers New/Open when no document is open.
func TestWelcomeScreen(t *testing.T) {
	c, w := newShellTest(t)
	useStore(nil, "") // close the document → welcome state
	c.renderWelcome()

	got := uiText(c.content)
	for _, want := range []string{"No file open", "New File…", "Open…"} {
		if !strings.Contains(got, want) {
			t.Errorf("welcome screen missing %q; text:\n%s", want, got)
		}
	}
	_ = w
}

// The Setup wizard dialog opens with its demo / from-scratch choice.
func TestSetupWizardOpens(t *testing.T) {
	_, w := newShellTest(t)
	showSetup()
	d := w.Canvas().Overlays().Top()
	if d == nil {
		t.Fatal("setup wizard did not open")
	}
	got := uiText(d)
	for _, want := range []string{"Welcome to Financy", "Continue…"} {
		if !strings.Contains(got, want) {
			t.Errorf("setup wizard missing %q", want)
		}
	}
}

// File ▸ New asks for a currency before the save dialog.
func TestPromptNewDocument(t *testing.T) {
	_, w := newShellTest(t)
	promptNewDocument()
	d := w.Canvas().Overlays().Top()
	if d == nil {
		t.Fatal("new-document prompt did not open")
	}
	assertUITextContains(t, d, "New document")
}

// The toolbar builds and exposes the navigation buttons.
func TestBuildToolbar(t *testing.T) {
	c, _ := newShellTest(t)
	tb := buildToolbar(c)
	// Tooltips/labels live on the tool buttons; ensure the bar assembles.
	if tb == nil {
		t.Fatal("buildToolbar returned nil")
	}
}

// A tool button shows its tooltip on hover and hides it on mouse-out.
func TestToolButtonTooltip(t *testing.T) {
	_, _ = newShellTest(t)
	b := newToolBtn(theme.ContentAddIcon(), "Add entry", func() {})
	w := test.NewWindow(b)
	defer w.Close()
	w.Resize(fyne.NewSize(120, 60))

	b.MouseIn(nil)
	if ttBox == nil || !ttBox.Visible() {
		t.Error("tooltip not shown on MouseIn")
	}
	b.MouseOut()
	if ttBox != nil && ttBox.Visible() {
		t.Error("tooltip not hidden on MouseOut")
	}
}

// Tapping a tool button fires its action.
func TestToolButtonTapped(t *testing.T) {
	_, _ = newShellTest(t)
	fired := false
	b := newToolBtn(theme.ContentAddIcon(), "Add", func() { fired = true })
	b.Tapped(nil)
	if !fired {
		t.Error("tool button action did not fire on tap")
	}
}

func assertUITextContains(t *testing.T, obj fyne.CanvasObject, want string) {
	t.Helper()
	if got := uiText(obj); !strings.Contains(got, want) {
		t.Errorf("missing %q; text:\n%s", want, got)
	}
}
