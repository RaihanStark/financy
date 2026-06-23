package core

import "testing"

func reportStore() *Store {
	SetCurrencySymbol("$")
	s := &Store{accounts: seedAccounts(), nextID: 1}
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	s.AddAccount(Account{ID: "visa", Name: "Credit Card", Type: Liability})
	// Opening balances + a month of activity.
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

func TestIncomeStatement(t *testing.T) {
	s := reportStore()
	is := s.IncomeStatementFor(serialOf(2026, 6, 1), serialOf(2026, 6, 30))
	if is.TotalIncome != 450000 {
		t.Fatalf("income = %d, want 450000", is.TotalIncome)
	}
	if is.TotalExpenses != 149000 { // 140000 rent + 9000 groceries
		t.Fatalf("expenses = %d, want 149000", is.TotalExpenses)
	}
	if is.NetIncome != 301000 {
		t.Fatalf("net = %d, want 301000", is.NetIncome)
	}
}

func TestBalanceSheetBalances(t *testing.T) {
	s := reportStore()
	bs := s.BalanceSheetAsOf(serialOf(2026, 6, 30))
	// The accounting identity must hold: Assets = Liabilities + Equity.
	if bs.TotalAssets != bs.TotalLiabilities+bs.TotalEquity {
		t.Fatalf("Assets %d != Liabilities %d + Equity %d",
			bs.TotalAssets, bs.TotalLiabilities, bs.TotalEquity)
	}
	// And equity equals net worth.
	if bs.TotalEquity != s.NetWorth() {
		t.Fatalf("equity %d != net worth %d", bs.TotalEquity, s.NetWorth())
	}
	// Checking = 3000 + 4500 - 1400 = 6100; card owed 90.
	if bs.TotalAssets != 610000 || bs.TotalLiabilities != 9000 {
		t.Fatalf("assets=%d liab=%d", bs.TotalAssets, bs.TotalLiabilities)
	}
}

func TestCashFlow(t *testing.T) {
	s := reportStore()
	cf := s.CashFlowFor(serialOf(2026, 6, 1), serialOf(2026, 6, 30))
	// Cash (Checking) went 0 → 6100 over the period; the card expense isn't cash.
	if cf.ClosingCash != 610000 || cf.OpeningCash != 0 {
		t.Fatalf("cash open=%d close=%d", cf.OpeningCash, cf.ClosingCash)
	}
	if cf.NetChange != 610000 {
		t.Fatalf("net change = %d, want 610000", cf.NetChange)
	}
}
