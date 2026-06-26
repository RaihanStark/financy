package view

import (
	"testing"

	"fyne.io/fyne/v2"

	"github.com/raihanstark/financy/internal/core"
)

// Reconciling to a different balance posts an adjustment transaction.
func TestReconcileCreatesAdjustment(t *testing.T) {
	s, w := newViewTest(t)
	checking := s.AccountByName("Checking")
	before := len(s.Transactions())
	current := s.DisplayBalance(*checking)

	reconcileAccount(*checking)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("reconcile form did not open")
	}
	entries := findEntries(d)
	if len(entries) == 0 {
		t.Fatal("reconcile form has no amount field")
	}
	// Set the actual balance well away from current so an adjustment is needed.
	target := current + 50000
	entries[0].SetText(fmtMoneyInput(target))

	if !tapButton(d, "Save") {
		t.Fatal("no Save button in reconcile form")
	}
	if got := len(s.Transactions()); got != before+1 {
		t.Fatalf("transactions = %d, want %d (reconcile adjustment)", got, before+1)
	}
	if bal := s.DisplayBalance(*checking); bal != target {
		t.Fatalf("balance after reconcile = %d, want %d", bal, target)
	}
}

// Reconciling to the same balance posts nothing and reports "Already balanced".
func TestReconcileNoChange(t *testing.T) {
	s, w := newViewTest(t)
	checking := s.AccountByName("Checking")
	before := len(s.Transactions())
	current := s.DisplayBalance(*checking)

	reconcileAccount(*checking)
	d := dialogContent(w)
	entries := findEntries(d)
	entries[0].SetText(fmtMoneyInput(current)) // unchanged
	if !tapButton(d, "Save") {
		t.Fatal("no Save button")
	}
	if got := len(s.Transactions()); got != before {
		t.Fatalf("transactions changed on no-op reconcile: %d → %d", before, got)
	}
	// An "Already balanced" info dialog should now be on top.
	if info := dialogContent(w); info != nil {
		assertText(t, info, "Already balanced")
	}
}

// Bulk recategorize reassigns matching expense transactions to a new category.
func TestBulkRecategorize(t *testing.T) {
	resetTxnFilters(t)
	s, w := newViewTest(t)

	// Select a couple of expense transactions.
	var picked []string
	for _, tx := range s.Transactions() {
		if deriveView(tx).kind == "Expense" {
			picked = append(picked, tx.ID)
		}
		if len(picked) == 2 {
			break
		}
	}
	if len(picked) < 2 {
		t.Skip("not enough expense transactions to recategorize")
	}
	selecting = true
	selected = map[string]bool{picked[0]: true, picked[1]: true}

	bulkRecategorize()
	d := dialogContent(w)
	if d == nil {
		t.Fatal("recategorize form did not open")
	}
	selects := findSelects(d)
	if len(selects) == 0 {
		t.Fatal("no category select in recategorize form")
	}
	selects[0].SetSelected("Shopping")
	if !tapButton(d, "Save") {
		t.Fatal("no Save button")
	}

	// Both picked transactions should now post to the Shopping category.
	shopping := s.AccountByName("Shopping")
	for _, id := range picked {
		tx := s.TxnByID(id)
		if tx == nil {
			t.Fatalf("transaction %s vanished", id)
		}
		hits := false
		for _, p := range tx.Posts {
			if p.AccountID == shopping.ID {
				hits = true
			}
		}
		if !hits {
			t.Errorf("transaction %s not moved to Shopping", id)
		}
	}
}

// postingsFor builds balanced postings (sum to zero) for each kind.
func TestPostingsForBalances(t *testing.T) {
	_, _ = newViewTest(t)
	for _, kind := range []string{"Expense", "Income", "Transfer"} {
		posts := postingsFor(kind, "checking", "groceries", 1000)
		sum := 0
		for _, p := range posts {
			sum += p.Amount
		}
		if sum != 0 {
			t.Errorf("%s postings sum to %d, want 0", kind, sum)
		}
	}
}

// The dialog helpers are no-ops (not panics) when no window is wired — the path
// the app hits before any document is open.
func TestDialogsNilWindowSafe(t *testing.T) {
	prev := win
	win = nil
	defer func() { win = prev }()
	showInfo("t", "m")
	confirmDelete("x", func() {})
	showForm("t", nil, func() {})
	showContextMenu(fyne.NewPos(0, 0), fyne.NewMenuItem("x", nil))
	_ = core.TodaySerial
}
