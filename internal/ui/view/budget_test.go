package view

import (
	"testing"
)

// Covering an overspent envelope from another category zeroes the shortfall
// and leaves no envelope in the red (the demo data ships with one overspent
// debt envelope: money paid to a debt that was never assigned).
func TestCoverOverspending(t *testing.T) {
	s, w := newViewTest(t)
	prevMonth := budgetMonth
	budgetMonth = currentMonthKey()
	t.Cleanup(func() { budgetMonth = prevMonth })

	bm := s.BudgetFor(budgetMonth)
	var over *BudgetCategory
	for i := range bm.Categories {
		if bm.Categories[i].Available < 0 {
			over = &bm.Categories[i]
			break
		}
	}
	if over == nil {
		t.Fatal("demo data has no overspent category to cover")
	}

	coverOverspendDialog(*over)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("cover dialog did not open")
	}
	// Defaults: richest source preselected, amount prefilled with the
	// shortfall — so a bare "Move" fully covers the envelope.
	if !tapButton(d, "Move") {
		t.Fatal("no Move button in cover dialog")
	}

	after := s.BudgetFor(budgetMonth)
	for _, c := range after.Categories {
		if c.ID == over.ID && c.Available != 0 {
			t.Fatalf("covered category Available = %d, want 0", c.Available)
		}
		if c.Available < 0 {
			t.Fatalf("category %q still overspent after cover", c.Name)
		}
	}
}

// A locked past month refuses the cover flow.
func TestCoverOverspendingLockedMonth(t *testing.T) {
	_, w := newViewTest(t)
	prevMonth := budgetMonth
	budgetMonth = shiftMonth(currentMonthKey(), -1)
	t.Cleanup(func() { budgetMonth = prevMonth })

	coverOverspendDialog(BudgetCategory{ID: "x", Name: "X", Available: -100})
	if d := dialogContent(w); d != nil {
		t.Fatal("cover dialog opened for a locked month")
	}
}
