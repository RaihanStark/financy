package ui

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

// newShellTest spins up the full application shell (toolbar + status bar +
// content) wired to a demo-seeded document, exactly as Run does, and returns the
// controller and its window. Package globals are reset on cleanup.
func newShellTest(t *testing.T) (*appController, fyne.Window) {
	t.Helper()
	a := test.NewApp()
	a.Settings().SetTheme(style.Theme{})

	initNav()

	w := a.NewWindow("financy-test")
	w.Resize(fyne.NewSize(1320, 860))

	prevCtl, prevStore, prevPrefs := ctl, store, prefs
	prefs = nil
	store = nil
	ctl = &appController{win: w}
	w.SetContent(assembleShell(ctl))

	s := core.NewStore()
	core.SeedDemo(s, "$")
	core.SetCurrencySymbol("$")
	useStore(s, "")

	t.Cleanup(func() {
		w.Close()
		ctl, store, prefs = prevCtl, prevStore, prevPrefs
		core.SetCurrencySymbol("Rp")
	})
	return ctl, w
}

func uiText(obj fyne.CanvasObject) string {
	var sb strings.Builder
	seen := map[fyne.CanvasObject]bool{}
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || seen[o] {
			return
		}
		seen[o] = true
		switch v := o.(type) {
		case *canvas.Text:
			sb.WriteString(v.Text + "\n")
		case *widget.Label:
			sb.WriteString(v.Text + "\n")
		case *widget.Button:
			sb.WriteString(v.Text + "\n")
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
		case fyne.Widget:
			func() {
				defer func() { _ = recover() }()
				if r := test.WidgetRenderer(v); r != nil {
					for _, c := range r.Objects() {
						walk(c)
					}
				}
			}()
		}
	}
	walk(obj)
	return sb.String()
}

// Opening a document lands on the Accounts screen with the account data shown.
func TestShellOpensOnAccounts(t *testing.T) {
	c, _ := newShellTest(t)
	if c.currentUID != "accounts" {
		t.Fatalf("startup screen = %q, want accounts", c.currentUID)
	}
	got := uiText(c.content)
	if !strings.Contains(got, "NET WORTH") {
		t.Errorf("accounts screen not rendered; text:\n%s", got)
	}
}

// Every navigation entry builds and renders its screen without panicking.
func TestShellNavigatesAllScreens(t *testing.T) {
	c, _ := newShellTest(t)
	want := map[string]string{
		"accounts":     "NET WORTH",
		"transactions": "Journal",
		"analytics":    "SAVINGS RATE",
		"recurring":    "Recurring",
	}
	for uid, marker := range want {
		c.show(uid)
		if c.currentUID != uid {
			t.Errorf("show(%q): currentUID = %q", uid, c.currentUID)
		}
		if got := uiText(c.content); !strings.Contains(got, marker) {
			t.Errorf("screen %q missing marker %q", uid, marker)
		}
	}
}

// The status bar reflects the active screen and the live net-worth figure.
func TestShellStatusBar(t *testing.T) {
	c, _ := newShellTest(t)
	c.show("transactions")
	if !strings.Contains(c.statusLeft.Text, "Transactions") {
		t.Errorf("status left = %q, want it to mention Transactions", c.statusLeft.Text)
	}
	if !strings.Contains(c.statusRight.Text, "Net Worth") {
		t.Errorf("status right = %q, want a Net Worth figure", c.statusRight.Text)
	}
	if !strings.Contains(c.statusRight.Text, fmtMoney(store.NetWorth())) {
		t.Errorf("status right net worth doesn't match store")
	}
}

// Unknown nav keys are ignored rather than crashing the shell.
func TestShellUnknownScreenIgnored(t *testing.T) {
	c, _ := newShellTest(t)
	c.show("accounts")
	c.show("does-not-exist")
	if c.currentUID != "accounts" {
		t.Errorf("unknown screen changed currentUID to %q", c.currentUID)
	}
}

// The toolbar quick-add always opens the add-transaction form, regardless of
// the current screen.
func TestShellQuickAdd(t *testing.T) {
	c, w := newShellTest(t)

	for _, uid := range []string{"accounts", "budget", "transactions"} {
		c.show(uid)
		quickAdd(c)
		d := w.Canvas().Overlays().Top()
		if d == nil {
			t.Fatalf("quickAdd on %s opened no dialog", uid)
		}
		if !strings.Contains(uiText(d), "Add Transaction") {
			t.Errorf("quickAdd on %s didn't open the transaction form", uid)
		}
		w.Canvas().Overlays().Remove(d)
	}
}

// With no document open the shell shows the welcome screen and an empty status.
func TestShellWelcomeWhenNoStore(t *testing.T) {
	c, _ := newShellTest(t)
	useStore(nil, "")
	if c.currentUID != "" {
		t.Errorf("currentUID = %q after closing document, want empty", c.currentUID)
	}
	if !strings.Contains(c.statusLeft.Text, "No file open") {
		t.Errorf("status left = %q, want 'No file open'", c.statusLeft.Text)
	}
}

// A store change notifies the controller, which re-renders the active screen.
func TestShellRefreshOnStoreChange(t *testing.T) {
	c, _ := newShellTest(t)
	c.show("accounts")

	store.AddAccount(core.Account{ID: "brokerage", Name: "Brokerage Zeta", Type: core.Asset})
	// Subscribe(ctl.refresh) should have re-rendered with the new account.
	if got := uiText(c.content); !strings.Contains(got, "Brokerage Zeta") {
		t.Errorf("new account not reflected after store change")
	}
}
