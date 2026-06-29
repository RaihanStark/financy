package view

import (
	"testing"
)

// The account context menu offers the expected actions.
func TestAccountMenuItems(t *testing.T) {
	s, _ := newViewTest(t)
	a := s.AccountByName("Checking")
	items := accountMenuItems(*a)
	labels := map[string]bool{}
	for _, it := range items {
		labels[it.Label] = true
	}
	for _, want := range []string{"New Transaction…", "View Register", "Reconcile…", "Edit Account…", "Delete Account"} {
		if !labels[want] {
			t.Errorf("account menu missing %q", want)
		}
	}
}

// The transaction context menu offers edit/duplicate/delete, plus an "Open
// <account>" jump for entries tied to a money account.
func TestTxnMenuItems(t *testing.T) {
	s, _ := newViewTest(t)
	var expense Transaction
	for _, tx := range s.Transactions() {
		if deriveView(tx).kind == "Expense" {
			expense = tx
			break
		}
	}
	if expense.ID == "" {
		t.Skip("no expense transaction")
	}
	items := txnMenuItems(expense)
	labels := map[string]bool{}
	for _, it := range items {
		labels[it.Label] = true
	}
	if !labels["Edit…"] || !labels["Duplicate"] || !labels["Delete"] {
		t.Errorf("txn menu missing core actions: %v", labels)
	}
}

// toggleSelect flips a transaction's selection state.
func TestToggleSelect(t *testing.T) {
	resetTxnFilters(t)
	_, _ = newViewTest(t)
	toggleSelect("txn-1")
	if !selected["txn-1"] {
		t.Fatal("toggleSelect did not select")
	}
	toggleSelect("txn-1")
	if selected["txn-1"] {
		t.Fatal("toggleSelect did not deselect")
	}
}

// Posting a due recurring entry as a new transaction adds it and advances the
// schedule.
func TestPostRecurringNowCreates(t *testing.T) {
	s, w := newViewTest(t)
	r := addDemoRecurring(s, todaySerial-1)
	before := len(s.Transactions())

	postRecurringNow(r)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("post-now dialog did not open")
	}
	// The dialog defaults to "create a new transaction"; Confirm posts it.
	if !tapButton(d, "Confirm") {
		t.Fatal("no Confirm button in post-now dialog")
	}
	if got := len(s.Transactions()); got != before+1 {
		t.Fatalf("transactions = %d, want %d after posting recurring", got, before+1)
	}
	if updated := s.RecurringByID(r.ID); updated == nil || updated.NextDue <= r.NextDue {
		t.Error("recurring schedule did not advance after posting")
	}
}

// CheckRecurringDue prompts with the due-review dialog when entries are pending.
func TestCheckRecurringDuePrompts(t *testing.T) {
	s, w := newViewTest(t)
	addDemoRecurring(s, todaySerial-1) // due

	CheckRecurringDue()
	d := dialogContent(w)
	if d == nil {
		t.Fatal("expected a due-review dialog")
	}
	assertText(t, d, "due", "Apply", "Later")
}

// candidateSelect always offers the post-new option plus a browse-all entry.
func TestCandidateSelect(t *testing.T) {
	s, _ := newViewTest(t)
	checking := s.AccountByName("Checking")
	sel := candidateSelect(postNewTodayOption, nil, checking.ID, false)
	if !sel.isPostNew() {
		t.Errorf("default selection = %q, want post-new", sel.widget.Selected)
	}
	if !optContains(sel.widget.Options, browseAllOption) {
		t.Error("candidate select missing the browse-all option")
	}
}
