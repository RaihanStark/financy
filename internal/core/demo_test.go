package core

import (
	"path/filepath"
	"testing"
)

func TestSeedDemo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.financy")
	s, err := NewDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SeedDemo(s, "$")

	if s.Currency() != "$" {
		t.Fatalf("currency = %q, want $", s.Currency())
	}
	// 4 hand-made money accounts (3 assets + the credit card) plus one Liability
	// account per demo BNPL debt.
	if got := len(s.MoneyAccounts()); got != 7 {
		t.Fatalf("money accounts = %d, want 7", got)
	}
	if got := len(s.Debts()); got != 3 {
		t.Fatalf("demo debts = %d, want 3", got)
	}
	for _, d := range s.Debts() {
		if s.AccountByID(d.AcctLiability) == nil {
			t.Fatalf("demo debt %q has no liability account", d.Name)
		}
	}
	if got := len(s.Recurrings()); got != 4 {
		t.Fatalf("recurring templates = %d, want 4", got)
	}
	// Demo recurring should be upcoming (not due), so loading demo doesn't nag.
	if len(s.PendingRecurring(TodaySerial)) != 0 {
		t.Fatal("demo recurring templates should not be due on load")
	}

	txns := s.Transactions()
	// Roughly six months of recurring + occasional activity — should be dozens
	// of entries, not the old handful.
	if len(txns) < 60 {
		t.Fatalf("demo has %d transactions, want a richer set (>= 60)", len(txns))
	}

	// The data should span ~6 months: the earliest entry is well before today.
	earliest := txns[0].Date
	for _, tx := range txns {
		if tx.Date < earliest {
			earliest = tx.Date
		}
	}
	if span := TodaySerial - earliest; span < 150 {
		t.Fatalf("demo spans %d days, want at least ~6 months (>= 150)", span)
	}

	// Double-entry must hold globally: every posting nets to zero, so the sum of
	// all account balances across the whole chart is exactly zero.
	total := 0
	for _, a := range append(append(s.AssetAccounts(), s.LiabilityAccounts()...),
		append(s.IncomeAccounts(), s.ExpenseAccounts()...)...) {
		total += s.Balance(a.ID)
	}
	total += s.Balance("opening") // equity
	if total != 0 {
		t.Fatalf("books are out of balance: all postings sum to %d, want 0", total)
	}

	// A healthy demo: positive net worth, with the card showing some balance owed.
	if s.NetWorth() <= 0 {
		t.Fatalf("net worth = %d, want positive", s.NetWorth())
	}
	if s.TotalLiabilities() <= 0 {
		t.Fatalf("liabilities = %d, want a non-zero card balance", s.TotalLiabilities())
	}

	// SeedDemo is idempotent (no-op when money accounts exist).
	before := len(s.Transactions())
	SeedDemo(s, "$")
	if len(s.Transactions()) != before {
		t.Fatal("SeedDemo duplicated data")
	}
}
