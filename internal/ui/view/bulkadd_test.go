package view

import (
	"strings"
	"testing"
)

// findTxnByPayee returns the first stored transaction with the given payee.
func findTxnByPayee(t *testing.T, payee string) *Transaction {
	t.Helper()
	for _, tx := range store.Transactions() {
		if tx.Payee == payee {
			cp := tx
			return &cp
		}
	}
	return nil
}

// Quick Add commits a mix of expense and income rows in one batch, deriving the
// kind from each chosen category's account type.
func TestQuickAddIncomeExpenseRows(t *testing.T) {
	resetTxnFilters(t)
	s, w := newViewTest(t)
	before := len(s.Transactions())

	QuickAddForm()
	d := dialogContent(w)
	if d == nil {
		t.Fatal("quick add did not open")
	}
	// Add a second income/expense row (the first add button is the I&E table's).
	if !tapButton(d, "＋ Add row") {
		t.Fatal("no Add row button")
	}

	// Per row: selects [money, cat]; so two I&E rows then the transfer row give
	// [ieMoney1, ieCat1, ieMoney2, ieCat2, trFrom, trTo].
	selects := findSelects(d)
	if len(selects) < 6 {
		t.Fatalf("unexpected selects: %d", len(selects))
	}
	selects[0].SetSelected(acctLabel("Checking"))
	selects[1].SetSelected(acctLabel("Groceries")) // expense
	selects[2].SetSelected(acctLabel("Checking"))
	selects[3].SetSelected(acctLabel("Salary")) // income

	// Per row: entries [date, amount, payee]; two I&E rows then the transfer row
	// give [d1, a1, p1, d2, a2, p2, d3, a3, p3].
	entries := findEntries(d)
	if len(entries) < 9 {
		t.Fatalf("unexpected entries: %d", len(entries))
	}
	entries[1].SetText("40")
	entries[2].SetText("QA-exp")
	entries[4].SetText("90")
	entries[5].SetText("QA-inc")

	if !tapButton(d, "Save all") {
		t.Fatal("no Save all button")
	}

	if got := len(s.Transactions()); got != before+2 {
		t.Fatalf("transactions = %d, want %d", got, before+2)
	}
	if tx := findTxnByPayee(t, "QA-exp"); tx == nil || deriveView(*tx).kind != "Expense" {
		t.Errorf("QA-exp row not posted as Expense: %+v", tx)
	}
	if tx := findTxnByPayee(t, "QA-inc"); tx == nil || deriveView(*tx).kind != "Income" {
		t.Errorf("QA-inc row not posted as Income: %+v", tx)
	}
}

// A transfer row posts a balanced transfer; a from==to row is rejected.
func TestQuickAddTransferRow(t *testing.T) {
	resetTxnFilters(t)
	s, w := newViewTest(t)

	// Valid transfer. Default rows: one I&E row [ieMoney, ieCat] then one transfer
	// row [trFrom, trTo]; entries are [ieDate, ieAmt, iePayee, trDate, trAmt, trPayee].
	before := len(s.Transactions())
	QuickAddForm()
	d := dialogContent(w)
	selects := findSelects(d)
	if len(selects) < 4 {
		t.Fatalf("unexpected selects: %d", len(selects))
	}
	selects[2].SetSelected(acctLabel("Checking"))
	selects[3].SetSelected(acctLabel("Savings"))
	entries := findEntries(d)
	if len(entries) < 6 {
		t.Fatalf("unexpected entries: %d", len(entries))
	}
	entries[4].SetText("100")
	entries[5].SetText("QA-xfer")
	if !tapButton(d, "Save all") {
		t.Fatal("no Save all button")
	}
	if got := len(s.Transactions()); got != before+1 {
		t.Fatalf("transactions = %d, want %d", got, before+1)
	}
	if tx := findTxnByPayee(t, "QA-xfer"); tx == nil || deriveView(*tx).kind != "Transfer" {
		t.Errorf("QA-xfer not posted as Transfer: %+v", tx)
	}

	// from == to should be skipped.
	before = len(s.Transactions())
	QuickAddForm()
	d = dialogContent(w)
	selects = findSelects(d)
	selects[2].SetSelected(acctLabel("Checking"))
	selects[3].SetSelected(acctLabel("Checking"))
	entries = findEntries(d)
	entries[4].SetText("100")
	entries[5].SetText("QA-self")
	if !tapButton(d, "Save all") {
		t.Fatal("no Save all button")
	}
	if got := len(s.Transactions()); got != before {
		t.Fatalf("from==to transfer was posted: %d → %d", before, got)
	}
}

// Blank and incomplete rows are silently skipped; only valid rows commit.
func TestQuickAddSkipsBlankRows(t *testing.T) {
	resetTxnFilters(t)
	s, w := newViewTest(t)
	before := len(s.Transactions())

	QuickAddForm()
	d := dialogContent(w)
	// Add an extra (blank) income/expense row that should be ignored.
	if !tapButton(d, "＋ Add row") {
		t.Fatal("no Add row button")
	}
	selects := findSelects(d)
	selects[0].SetSelected(acctLabel("Checking"))
	selects[1].SetSelected(acctLabel("Groceries"))
	entries := findEntries(d)
	entries[1].SetText("25")
	entries[2].SetText("QA-blank")
	// Second I&E row + the transfer row are left empty.

	if !tapButton(d, "Save all") {
		t.Fatal("no Save all button")
	}
	if got := len(s.Transactions()); got != before+1 {
		t.Fatalf("transactions = %d, want %d (only one valid row)", got, before+1)
	}
}

// Adding then removing a row leaves the form back at its starting shape.
func TestQuickAddRowAddRemove(t *testing.T) {
	resetTxnFilters(t)
	_, w := newViewTest(t)

	QuickAddForm()
	d := dialogContent(w)
	start := len(findSelects(d))

	// Each I&E row has two selects (money + category), so adding one grows the
	// count by 2 and removing it shrinks it back.
	if !tapButton(d, "＋ Add row") {
		t.Fatal("no Add row button")
	}
	if got := len(findSelects(d)); got != start+2 {
		t.Fatalf("after add: %d selects, want %d", got, start+2)
	}

	if !tapButton(d, "✕") {
		t.Fatal("no remove button")
	}
	if got := len(findSelects(d)); got != start {
		t.Fatalf("after remove: %d selects, want %d", got, start)
	}
}

// Each date field shows an explicit calendar-picker button (a real layout
// sibling), matching the single transaction form, so users can pick a day
// instead of only typing.
func TestQuickAddDateHasCalendarButton(t *testing.T) {
	resetTxnFilters(t)
	_, w := newViewTest(t)

	QuickAddForm()
	d := dialogContent(w)

	cal := 0
	for _, b := range findButtons(d) {
		if b.Icon != nil && strings.Contains(strings.ToLower(b.Icon.Name()), "calendar") {
			cal++
		}
	}
	// One income/expense row + one transfer row are present by default.
	if cal != 2 {
		t.Fatalf("calendar picker buttons = %d, want 2 (one per date field)", cal)
	}
}
