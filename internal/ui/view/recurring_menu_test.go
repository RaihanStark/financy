package view

import (
	"testing"

	"github.com/raihanstark/financy/internal/core"
)

func TestRecurringMenuHasPostNow(t *testing.T) {
	r := core.Recurring{ID: "r1", Kind: "Expense", Payee: "Rent", Freq: "Monthly", NextDue: core.TodaySerial + 5}
	m := recurringMenu(r)
	if len(m) == 0 || m[0].Label != "Post now (early)…" {
		t.Fatalf("first recurring menu item = %q, want 'Post now (early)…'", m[0].Label)
	}
}
