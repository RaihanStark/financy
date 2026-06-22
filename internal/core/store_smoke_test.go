package core

import "testing"

func TestDoubleEntryInvariants(t *testing.T) {
	s := NewStore()

	// A fresh store has no personal data and zero net worth.
	if s.NetWorth() != 0 || s.TotalAssets() != 0 || s.TotalLiabilities() != 0 {
		t.Fatalf("fresh store not empty: assets=%d liab=%d nw=%d",
			s.TotalAssets(), s.TotalLiabilities(), s.NetWorth())
	}
	if len(s.Transactions()) != 0 {
		t.Fatalf("fresh store has %d transactions, want 0", len(s.Transactions()))
	}

	// Add money accounts.
	s.AddAccount(Account{ID: "bank", Name: "Bank", Type: Asset})
	s.AddAccount(Account{ID: "loan", Name: "Loan", Type: Liability})

	// Opening balance: asset debit, equity credit.
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Opening",
		Posts: []Posting{P("bank", 1_000_000), P("opening", -1_000_000)}})
	if s.TotalAssets() != 1_000_000 || s.NetWorth() != 1_000_000 {
		t.Fatalf("after opening: assets=%d nw=%d", s.TotalAssets(), s.NetWorth())
	}

	// Income raises the asset balance and the income total.
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Pay",
		Posts: []Posting{P("bank", 500_000), P("salary", -500_000)}})
	if s.IncomeTotal() != 500_000 {
		t.Fatalf("income total = %d, want 500000", s.IncomeTotal())
	}

	// Expense lowers assets and raises the expense total.
	exp0 := s.ExpenseTotal()
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Food",
		Posts: []Posting{P("groceries", 200_000), P("bank", -200_000)}})
	if s.ExpenseTotal() != exp0+200_000 {
		t.Fatalf("expense total not updated")
	}
	if s.TotalAssets() != 1_300_000 {
		t.Fatalf("assets = %d, want 1300000", s.TotalAssets())
	}

	// A transfer never changes net worth.
	nw := s.NetWorth()
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Pay loan",
		Posts: []Posting{P("loan", 300_000), P("bank", -300_000)}})
	if s.NetWorth() != nw {
		t.Fatalf("transfer changed net worth: %d vs %d", s.NetWorth(), nw)
	}

	// Every transaction balances to zero.
	for _, tx := range s.txns {
		if !tx.balanced() {
			t.Fatalf("unbalanced txn %s (%s)", tx.ID, tx.Payee)
		}
	}

	// Unbalanced transactions are rejected.
	if s.AddTransaction(Transaction{Date: TodaySerial,
		Posts: []Posting{P("bank", 100), P("salary", -50)}}) {
		t.Fatal("unbalanced transaction was accepted")
	}
}
