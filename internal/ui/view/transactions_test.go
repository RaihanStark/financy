package view

import (
	"testing"

	"github.com/raihanstark/financy/internal/core"
)

// resetTxnFilters restores the package-level filter/selection state so tests
// don't leak into each other (these globals persist across renders by design).
func resetTxnFilters(t *testing.T) {
	t.Helper()
	prevM, prevT, prevA, prevS := filterMonth, filterType, filterAccount, filterSearch
	prevSelecting, prevSelected := selecting, selected
	filterMonth, filterType, filterAccount, filterSearch = "All months", "All types", "All accounts", ""
	selecting, selected = false, map[string]bool{}
	t.Cleanup(func() {
		filterMonth, filterType, filterAccount, filterSearch = prevM, prevT, prevA, prevS
		selecting, selected = prevSelecting, prevSelected
	})
}

// The transactions screen renders the journal with filters and the summary cards.
func TestScreenTransactionsRenders(t *testing.T) {
	resetTxnFilters(t)
	s, w := newViewTest(t)
	screen := mount(w, ScreenTransactions())

	assertText(t, screen,
		"Transactions", "Add Transaction", "Import CSV", "Select",
		"SHOWING", "INCOME (filtered)", "EXPENSES (filtered)", "NET (filtered)",
		"Journal",
	)
	// The "showing N of M entries" count reflects the store.
	assertText(t, screen, "of "+itoa(len(s.Transactions()))+" entries")
}

// Filtering by type to Income hides expense-only payees and the summary updates.
func TestScreenTransactionsTypeFilter(t *testing.T) {
	resetTxnFilters(t)
	_, w := newViewTest(t)

	filterType = "Income"
	screen := mount(w, ScreenTransactions())
	// Payroll is income in the demo data; it should still show.
	assertText(t, screen, "Acme Corp Payroll")
	// Landlord is an expense; with an Income-only filter it should be gone.
	assertNoText(t, screen, "Landlord")
}

// Searching narrows the journal to matching payees.
func TestScreenTransactionsSearchFilter(t *testing.T) {
	resetTxnFilters(t)
	_, w := newViewTest(t)

	filterSearch = "Netflix"
	screen := mount(w, ScreenTransactions())
	assertText(t, screen, "Netflix")
	assertNoText(t, screen, "Landlord")
}

// A filter combination matching nothing shows the empty-journal state.
func TestScreenTransactionsNoMatchEmptyState(t *testing.T) {
	resetTxnFilters(t)
	_, w := newViewTest(t)

	filterSearch = "zzz-no-such-payee-zzz"
	screen := mount(w, ScreenTransactions())
	assertText(t, screen, "No transactions match the current filters")
}

// In select mode the bar switches to the bulk-selection controls.
func TestScreenTransactionsSelectMode(t *testing.T) {
	resetTxnFilters(t)
	_, w := newViewTest(t)

	selecting = true
	screen := mount(w, ScreenTransactions())
	assertText(t, screen, "selected", "Select all shown", "Done")
}

// deriveView classifies each kind of balanced transaction correctly.
func TestDeriveViewKinds(t *testing.T) {
	s, _ := newViewTest(t)

	mk := func(posts ...core.Posting) Transaction {
		return Transaction{Date: core.TodaySerial, Posts: posts}
	}
	// Expense: category(+) / bank(-)
	if v := deriveView(mk(core.P("groceries", 5000), core.P("checking", -5000))); v.kind != "Expense" || v.amount != 5000 {
		t.Errorf("expense derive = %+v", v)
	}
	// Income: bank(+) / income(-)
	if v := deriveView(mk(core.P("checking", 4000), core.P("salary", -4000))); v.kind != "Income" || v.amount != 4000 {
		t.Errorf("income derive = %+v", v)
	}
	// Transfer between two money accounts.
	if v := deriveView(mk(core.P("savings", 2000), core.P("checking", -2000))); v.kind != "Transfer" {
		t.Errorf("transfer derive = %+v", v)
	}
	// Opening balance touches equity.
	if v := deriveView(mk(core.P("checking", 1000), core.P("opening", -1000))); v.kind != "Opening" {
		t.Errorf("opening derive = %+v", v)
	}
	_ = s
}

// displayPayee shows the payee when present, and falls back to the transaction
// type (not the category) when the payee is blank.
func TestDisplayPayee(t *testing.T) {
	if got := displayPayee(Transaction{Payee: "Netflix"}, txnView{kind: "Expense", catName: "Subscriptions"}); got != "Netflix" {
		t.Errorf("with payee: got %q, want %q", got, "Netflix")
	}
	if got := displayPayee(Transaction{Payee: ""}, txnView{kind: "Expense", catName: "Subscriptions"}); got != "Expense" {
		t.Errorf("blank payee: got %q, want %q", got, "Expense")
	}
	if got := displayPayee(Transaction{Payee: ""}, txnView{kind: "Transfer"}); got != "Transfer" {
		t.Errorf("blank payee transfer: got %q, want %q", got, "Transfer")
	}
}

// Adding a transaction through the form posts a balanced entry to the store.
func TestTransactionFormAdd(t *testing.T) {
	resetTxnFilters(t)
	s, w := newViewTest(t)
	before := len(s.Transactions())

	TransactionForm("", "")
	d := dialogContent(w)
	if d == nil {
		t.Fatal("transaction form did not open")
	}

	selects := findSelects(d)
	entries := findEntries(d)
	// Form order: Type, Date, Amount, MoneyAccount, Category/To, Payee, Memo.
	// selects[0]=Type, selects[1]=acctA (money), selects[2]=acctB (category)
	if len(selects) < 3 || len(entries) < 4 {
		t.Fatalf("unexpected form shape: %d selects, %d entries", len(selects), len(entries))
	}
	selects[0].SetSelected("Expense")
	selects[1].SetSelected("Checking")
	selects[2].SetSelected("Groceries")

	// Find the amount entry (the one that lives between date and payee). The
	// date entry is prefilled with today; set amount + payee by content.
	for _, e := range entries {
		if e.Text == "" {
			e.SetText("123.45")
			break
		}
	}

	if !tapButton(d, "Save") {
		t.Fatal("no Save button in transaction form")
	}
	if got := len(s.Transactions()); got != before+1 {
		t.Fatalf("transactions = %d, want %d", got, before+1)
	}
}

// Duplicating a transaction adds an identical copy.
func TestDuplicateTxn(t *testing.T) {
	s, _ := newViewTest(t)
	txns := s.Transactions()
	if len(txns) == 0 {
		t.Skip("no transactions")
	}
	before := len(txns)
	duplicateTxn(txns[0])
	if got := len(s.Transactions()); got != before+1 {
		t.Fatalf("after duplicate: %d txns, want %d", got, before+1)
	}
}

// distinctMonths returns the set of months present, newest first.
func TestDistinctMonths(t *testing.T) {
	_, _ = newViewTest(t)
	ms := distinctMonths()
	if len(ms) == 0 {
		t.Fatal("expected at least one distinct month in demo data")
	}
}

// idOf/nameOf round-trip an account name through its ID.
func TestIDNameRoundTrip(t *testing.T) {
	s, _ := newViewTest(t)
	checking := s.AccountByName("Checking")
	if id := idOf("Checking"); id != checking.ID {
		t.Fatalf("idOf(Checking) = %q, want %q", id, checking.ID)
	}
	if n := nameOf(checking.ID); n != "Checking" {
		t.Fatalf("nameOf(%q) = %q, want Checking", checking.ID, n)
	}
}
