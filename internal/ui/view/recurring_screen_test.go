package view

import (
	"testing"

	"github.com/raihanstark/financy/internal/core"
)

// addDemoRecurring seeds one monthly expense template, due in the past so it
// counts as pending. Returns it.
func addDemoRecurring(s *core.Store, due int) core.Recurring {
	r := core.Recurring{
		Kind: "Expense", AcctA: "checking", AcctB: "housing",
		Amount: 140000, Payee: "Rent", Freq: "Monthly",
		NextDue: due, Enabled: true,
	}
	s.AddRecurring(r)
	rs := s.Recurrings()
	return rs[len(rs)-1]
}

// With no templates the recurring screen shows its empty state. The empty store
// (categories only) seeds no recurring templates.
func TestScreenRecurringEmptyState(t *testing.T) {
	_, w := emptyViewTest(t)
	screen := mount(w, ScreenRecurring())
	assertText(t, screen, "Recurring", "No recurring transactions yet")
}

// A template renders as a row with payee, frequency and amount; a past-due one
// surfaces the due banner.
func TestScreenRecurringWithDueItem(t *testing.T) {
	s, w := newViewTest(t)
	addDemoRecurring(s, todaySerial-1) // due yesterday

	screen := mount(w, ScreenRecurring())
	assertText(t, screen, "Rent", "Monthly", "due", "Review & Post")
}

// A future-dated, enabled template shows its next-due date, not the due banner.
func TestScreenRecurringFutureItem(t *testing.T) {
	s, w := newViewTest(t)
	addDemoRecurring(s, todaySerial+10)

	screen := mount(w, ScreenRecurring())
	assertText(t, screen, "Rent", "next ·")
}

// Adding a template through RecurringForm persists it to the store.
func TestRecurringFormAdd(t *testing.T) {
	s, w := newViewTest(t)
	before := len(s.Recurrings())

	RecurringForm(nil)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("recurring form did not open")
	}
	selects := findSelects(d)
	entries := findEntries(d)
	// selects: [0]=Type [1]=acctA [2]=acctB [3]=Freq ; entries include amount/payee/memo/date
	if len(selects) < 4 {
		t.Fatalf("expected >=4 selects, got %d", len(selects))
	}
	selects[0].SetSelected("Expense")
	selects[1].SetSelected(acctLabel("Checking"))
	selects[2].SetSelected(acctLabel("Housing"))
	selects[3].SetSelected("Monthly")
	for _, e := range entries {
		if e.Text == "" {
			e.SetText("90000")
			break
		}
	}
	if !tapButton(d, "Save") {
		t.Fatal("no Save button in recurring form")
	}
	if got := len(s.Recurrings()); got != before+1 {
		t.Fatalf("recurrings = %d, want %d", got, before+1)
	}
}

// postDueNow with nothing pending shows the "Nothing due" info dialog.
func TestPostDueNowNothingDue(t *testing.T) {
	_, w := newViewTest(t)
	postDueNow()
	d := dialogContent(w)
	if d == nil {
		t.Fatal("expected an info dialog")
	}
	assertText(t, d, "Nothing due")
}

// The recurring context menu leads with the early-post action and a delete.
func TestRecurringMenuItems(t *testing.T) {
	s, _ := newViewTest(t)
	r := addDemoRecurring(s, todaySerial+5)
	m := recurringMenu(r)
	if len(m) < 2 {
		t.Fatalf("recurring menu too short: %d", len(m))
	}
	if m[0].Label != "Post now (early)…" {
		t.Errorf("first item = %q", m[0].Label)
	}
}

// amountLabel signs the amount per kind; plural picks the right word.
func TestAmountLabelAndPlural(t *testing.T) {
	_, _ = newViewTest(t)
	if got := plural(1, "entry", "entries"); got != "entry" {
		t.Errorf("plural(1) = %q", got)
	}
	if got := plural(3, "entry", "entries"); got != "entries" {
		t.Errorf("plural(3) = %q", got)
	}
	if amountLabel("Income", 1000) == "" || amountLabel("Expense", 1000) == "" {
		t.Error("amountLabel returned empty")
	}
}
