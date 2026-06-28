package core

import "testing"

// recatStore seeds opening balances plus a month of income/expense activity used
// by the recategorization test.
func recatStore() *Store {
	SetCurrencySymbol("$")
	s := &Store{accounts: seedAccounts(), nextID: 1}
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	s.AddAccount(Account{ID: "visa", Name: "Credit Card", Type: Liability})
	s.AddTransaction(Transaction{Date: serialOf(2026, 6, 1), Payee: "Opening",
		Posts: []Posting{P("checking", 300000), P("opening", -300000)}})
	s.AddTransaction(Transaction{Date: serialOf(2026, 6, 2), Payee: "Payroll",
		Posts: PostingsFor(KindIncome, "checking", "salary", 450000)})
	s.AddTransaction(Transaction{Date: serialOf(2026, 6, 3), Payee: "Rent",
		Posts: PostingsFor(KindExpense, "checking", "housing", 140000)})
	s.AddTransaction(Transaction{Date: serialOf(2026, 6, 4), Payee: "Groceries on card",
		Posts: PostingsFor(KindExpense, "visa", "groceries", 9000)})
	return s
}

func TestRecategorize(t *testing.T) {
	s := recatStore()
	// The two expense txns: Rent→housing, Groceries→groceries.
	var rent, groc, payroll, opening string
	for _, txn := range s.Transactions() {
		switch txn.Payee {
		case "Rent":
			rent = txn.ID
		case "Groceries on card":
			groc = txn.ID
		case "Payroll":
			payroll = txn.ID
		case "Opening":
			opening = txn.ID
		}
	}

	// Move both expenses to "shopping", but also pass an income and an opening
	// txn — those must be skipped because their category posting isn't an Expense.
	n := s.RecategorizeTransactions([]string{rent, groc, payroll, opening}, "shopping")
	if n != 2 {
		t.Fatalf("changed %d, want 2", n)
	}

	catOf := func(id string) string {
		for _, txn := range s.Transactions() {
			if txn.ID != id {
				continue
			}
			if i := s.categoryPostingIndex(txn, Expense); i >= 0 {
				return txn.Posts[i].AccountID
			}
		}
		return ""
	}
	if got := catOf(rent); got != "shopping" {
		t.Fatalf("rent category = %q, want shopping", got)
	}
	if got := catOf(groc); got != "shopping" {
		t.Fatalf("groceries category = %q, want shopping", got)
	}

	// Both expenses now lump under Shopping; the total is unchanged.
	total := 0
	for _, c := range s.CategoryBreakdown(serialOf(2026, 6, 1), serialOf(2026, 6, 30), Expense) {
		total += c.Total
	}
	if total != 149000 {
		t.Fatalf("expenses total changed to %d, want 149000", total)
	}

	// Re-running is a no-op (already there) and an unknown category does nothing.
	if again := s.RecategorizeTransactions([]string{rent, groc}, "shopping"); again != 0 {
		t.Fatalf("re-run changed %d, want 0", again)
	}
	if bad := s.RecategorizeTransactions([]string{rent}, "no-such-account"); bad != 0 {
		t.Fatalf("unknown category changed %d, want 0", bad)
	}
	// Recategorizing to an income account must be refused for an expense txn.
	if cross := s.RecategorizeTransactions([]string{rent}, "salary"); cross != 0 {
		t.Fatalf("cross-type changed %d, want 0", cross)
	}
}
