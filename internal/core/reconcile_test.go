package core

import "testing"

func reconcileStore(t *testing.T) *Store {
	t.Helper()
	SetCurrencySymbol("$")
	s := &Store{accounts: seedAccounts(), nextID: 1}
	return s
}

func TestReconcileAssetUpAndDown(t *testing.T) {
	s := reconcileStore(t)
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Opening",
		Posts: []Posting{P("checking", 400000), P("opening", -400000)}}) // $4,000

	// Bank says $5,000 → +$1,000 adjustment.
	delta, created := s.Reconcile("checking", 500000, TodaySerial)
	if !created || delta != 100000 {
		t.Fatalf("up: delta=%d created=%v, want 100000/true", delta, created)
	}
	if got := s.DisplayBalance(*s.AccountByID("checking")); got != 500000 {
		t.Fatalf("balance after up = %d, want 500000", got)
	}

	// Bank says $4,250 → −$750 adjustment.
	delta, created = s.Reconcile("checking", 425000, TodaySerial)
	if !created || delta != -75000 {
		t.Fatalf("down: delta=%d created=%v, want -75000/true", delta, created)
	}
	if got := s.DisplayBalance(*s.AccountByID("checking")); got != 425000 {
		t.Fatalf("balance after down = %d, want 425000", got)
	}
}

func TestReconcileLiability(t *testing.T) {
	s := reconcileStore(t)
	s.AddAccount(Account{ID: "visa", Name: "Visa", Type: Liability})
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Opening",
		Posts: []Posting{P("visa", -120000), P("opening", 120000)}}) // owes $1,200

	// Statement says you owe $1,500 → owed goes up by $300.
	delta, created := s.Reconcile("visa", 150000, TodaySerial)
	if !created || delta != 30000 {
		t.Fatalf("liability: delta=%d created=%v, want 30000/true", delta, created)
	}
	if got := s.DisplayBalance(*s.AccountByID("visa")); got != 150000 {
		t.Fatalf("owed after = %d, want 150000", got)
	}
	// Net worth dropped by the extra debt; books stay balanced.
	if got := s.TotalLiabilities(); got != 150000 {
		t.Fatalf("total liabilities = %d, want 150000", got)
	}
}

func TestReconcileNoChange(t *testing.T) {
	s := reconcileStore(t)
	s.AddAccount(Account{ID: "cash", Name: "Cash", Type: Asset})
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Opening",
		Posts: []Posting{P("cash", 5000), P("opening", -5000)}})
	before := len(s.Transactions())
	if delta, created := s.Reconcile("cash", 5000, TodaySerial); created || delta != 0 {
		t.Fatalf("no-change: delta=%d created=%v, want 0/false", delta, created)
	}
	if len(s.Transactions()) != before {
		t.Fatal("a balanced reconcile should not create a transaction")
	}
	// The Reconciliation equity account isn't created until actually needed.
	if s.AccountByID(reconcileAccountID) != nil {
		t.Fatal("reconcile account created despite no adjustment")
	}
}

func TestReconcileBooksStayBalanced(t *testing.T) {
	s := reconcileStore(t)
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	s.Reconcile("checking", 333333, TodaySerial)
	total := 0
	for _, a := range s.accounts {
		total += s.Balance(a.ID)
	}
	if total != 0 {
		t.Fatalf("all postings sum to %d after reconcile, want 0", total)
	}
}
