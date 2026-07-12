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

// With no templates the recurring screen shows its explainer welcome. The
// empty store (categories only) seeds no recurring templates.
func TestScreenRecurringEmptyState(t *testing.T) {
	_, w := emptyViewTest(t)
	screen := mount(w, ScreenRecurring())
	assertText(t, screen, "Recurring", "autopilot", "Nothing posts without your confirmation")
}

// A template renders as a row with payee, frequency and amount; a past-due one
// surfaces the due banner, the overdue bucket, and the commitment stats.
func TestScreenRecurringWithDueItem(t *testing.T) {
	s, w := newViewTest(t)
	addDemoRecurring(s, todaySerial-1) // due yesterday

	screen := mount(w, ScreenRecurring())
	assertText(t, screen, "Rent", "Monthly", "due", "Review & Post",
		"MONTHLY BILLS", "MONTHLY INCOME", "NET PER MONTH", "OVERDUE & DUE", "All templates")
}

// A future-dated, enabled template shows its next-due date, not the due banner.
func TestScreenRecurringFutureItem(t *testing.T) {
	s, w := newViewTest(t)
	addDemoRecurring(s, todaySerial+10)

	screen := mount(w, ScreenRecurring())
	assertText(t, screen, "Rent", "next ·")
}

// The 30-day timeline buckets occurrences: overdue/due, this week, and later
// this month, each with the payee and a distance-from-today label.
func TestUpcomingPanelBuckets(t *testing.T) {
	s, w := newViewTest(t)
	addDemoRecurring(s, todaySerial-1) // overdue
	s.AddRecurring(core.Recurring{Kind: "Income", AcctA: "checking", AcctB: "salary",
		Amount: 450000, Payee: "Payroll", Freq: "Monthly", NextDue: todaySerial + 3, Enabled: true})
	s.AddRecurring(core.Recurring{Kind: "Expense", AcctA: "checking", AcctB: "housing",
		Amount: 5000, Payee: "Cloud", Freq: "Monthly", NextDue: todaySerial + 20, Enabled: true})

	screen := mount(w, ScreenRecurring())
	assertText(t, screen, "OVERDUE & DUE", "THIS WEEK", "LATER THIS MONTH",
		"Rent", "Payroll", "Cloud", "1 day overdue", "in 3 days", "in 20 days")
}

// The commitment stats normalize each frequency to a monthly figure.
func TestRecurringSummaryStats(t *testing.T) {
	s, w := newViewTest(t)
	for _, r := range s.Recurrings() { // drop demo templates so the total is ours
		s.DeleteRecurring(r.ID)
	}
	s.AddRecurring(core.Recurring{Kind: "Expense", AcctA: "checking", AcctB: "housing",
		Amount: 120000, Payee: "Insurance", Freq: "Yearly", NextDue: todaySerial + 40, Enabled: true})

	screen := mount(w, ScreenRecurring())
	// 120000/12 = 10000 minor units → $100.00 monthly bills.
	assertText(t, screen, "MONTHLY BILLS", "100.00", "ACTIVE")
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

// RecheckRecurringDue prompts when something is due, stays quiet while the due
// set is unchanged (Later means later), skips when a dialog is already open,
// and prompts again once a new occurrence becomes due.
func TestRecheckRecurringDue(t *testing.T) {
	s, w := newViewTest(t)
	lastDueSig = ""
	t.Cleanup(func() { lastDueSig = "" })

	// Nothing due → no prompt, signature stays clear.
	RecheckRecurringDue()
	if dialogContent(w) != nil {
		t.Fatal("prompted with nothing due")
	}

	addDemoRecurring(s, todaySerial-1)
	RecheckRecurringDue()
	d := dialogContent(w)
	if d == nil {
		t.Fatal("expected a due prompt")
	}

	// A dialog is open → the next tick must not stack another prompt.
	RecheckRecurringDue()
	if !tapButton(dialogContent(w), "Later") {
		t.Fatal("no Later button on due prompt")
	}
	if dialogContent(w) != nil {
		t.Fatal("Later did not dismiss the prompt")
	}

	// Unchanged due set → deferred, no re-prompt.
	RecheckRecurringDue()
	if dialogContent(w) != nil {
		t.Fatal("re-prompted for an unchanged due set")
	}

	// A second template becomes due → the set changed, prompt again.
	s.AddRecurring(core.Recurring{Kind: "Expense", AcctA: "checking", AcctB: "housing",
		Amount: 5000, Payee: "Cloud", Freq: "Monthly", NextDue: todaySerial, Enabled: true})
	RecheckRecurringDue()
	if dialogContent(w) == nil {
		t.Fatal("expected a prompt after a new occurrence became due")
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
