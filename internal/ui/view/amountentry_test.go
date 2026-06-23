package view

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/raihanstark/financy/internal/core"
)

func TestAmountEntryLiveGrouping(t *testing.T) {
	test.NewApp()
	core.SetCurrencySymbol("$") // 2 decimals, "," thousands, "." decimal
	defer core.SetCurrencySymbol("Rp")

	e := newAmountEntry()
	w := test.NewWindow(e)
	defer w.Close()

	test.Type(e, "1000000")
	if e.Text != "1,000,000" {
		t.Fatalf("typing 1000000 → %q, want 1,000,000", e.Text)
	}

	// A decimal amount, and EU number format mid-session.
	e2 := newAmountEntry()
	test.NewWindow(e2)
	test.Type(e2, "1234.50")
	if e2.Text != "1,234.50" {
		t.Fatalf("typing 1234.50 → %q, want 1,234.50", e2.Text)
	}

	core.ApplyNumberFormat("1.234,56") // EU separators
	e3 := newAmountEntry()
	test.NewWindow(e3)
	test.Type(e3, "1234567")
	if e3.Text != "1.234.567" {
		t.Fatalf("EU typing 1234567 → %q, want 1.234.567", e3.Text)
	}
}
