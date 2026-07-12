package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
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

// The sidebar builds and registers the six navigation items.
func TestBuildSidebar(t *testing.T) {
	c, _ := newShellTest(t)
	sb := buildSidebar(c)
	if sb == nil {
		t.Fatal("buildSidebar returned nil")
	}
	if len(navItems) != 6 {
		t.Fatalf("sidebar registered %d nav items, want 6", len(navItems))
	}
	got := uiText(sb)
	for _, want := range []string{"Financy", "New Transaction", "Accounts", "Transactions",
		"Budget", "Recurring", "Debts", "Analytics", "Categories", "Preferences", "NET WORTH"} {
		if !strings.Contains(got, want) {
			t.Errorf("sidebar missing %q; text:\n%s", want, got)
		}
	}
}

// The floating tooltip layer shows and hides on demand (charts drive it).
func TestTooltipLayer(t *testing.T) {
	_, _ = newShellTest(t)
	showToolTip("hello", fyne.NewPos(10, 10))
	if ttBox == nil || !ttBox.Visible() {
		t.Error("tooltip not shown by showToolTip")
	}
	hideToolTip()
	if ttBox != nil && ttBox.Visible() {
		t.Error("tooltip not hidden by hideToolTip")
	}
}

// Tapping a sidebar item fires its action.
func TestSideItemTapped(t *testing.T) {
	_, _ = newShellTest(t)
	fired := false
	b := newSideItem(theme.ContentAddIcon(), "Add", func() { fired = true })
	b.Tapped(nil)
	if !fired {
		t.Error("sidebar item action did not fire on tap")
	}
}

func assertUITextContains(t *testing.T, obj fyne.CanvasObject, want string) {
	t.Helper()
	if got := uiText(obj); !strings.Contains(got, want) {
		t.Errorf("missing %q; text:\n%s", want, got)
	}
}
