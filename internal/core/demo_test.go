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
	defer func() { _ = s.Close() }()
	SeedDemo(s, "$")

	if s.Currency() != "$" {
		t.Fatalf("currency = %q, want $", s.Currency())
	}
	// 5 hand-made money accounts (4 assets incl. the off-budget brokerage + the
	// credit card) plus one Liability account per debt that owns one: 3 BNPL,
	// the car loan, and the IOU. The revolving debt attaches the existing card.
	if got := len(s.MoneyAccounts()); got != 10 {
		t.Fatalf("money accounts = %d, want 10", got)
	}
	if got := len(s.Debts()); got != 6 {
		t.Fatalf("demo debts = %d, want 6 (3 BNPL + loan + card + IOU)", got)
	}
	types := map[string]int{}
	for _, d := range s.Debts() {
		types[d.Type]++
		if s.AccountByID(d.AcctLiability) == nil {
			t.Fatalf("demo debt %q has no liability account", d.Name)
		}
	}
	if types[DebtBNPL] != 3 || types[DebtLoan] != 1 || types[DebtRevolving] != 1 || types[DebtInformal] != 1 {
		t.Fatalf("demo debt mix = %v, want 3 BNPL / 1 Loan / 1 Revolving / 1 Informal", types)
	}
	// The revolving debt attaches the pre-existing card account, not a new one.
	for _, d := range s.Debts() {
		if d.IsRevolving() && (d.AcctLiability != "visa" || d.OwnsAccount) {
			t.Fatalf("revolving demo debt should attach the visa account: %+v", d)
		}
	}
	// The demo card's statement shouldn't nag the moment the demo loads.
	if got := len(s.PendingStatements(TodaySerial)); got != 0 {
		t.Fatalf("pending statements on load = %d, want 0", got)
	}
	if got := len(s.Recurrings()); got != 4 {
		t.Fatalf("recurring templates = %d, want 4", got)
	}

	// The brokerage is an off-budget tracking account: in Net Worth, out of the
	// budget's funds pool.
	if a := s.AccountByName("Investments"); a == nil || !a.OffBudget {
		t.Fatal("Investments should exist and be off-budget")
	}

	// The Budget module is wired up: this month has real assignments, money has
	// been given jobs (Ready to Assign < total funds), and the sinking funds have
	// rolled an Available balance forward.
	bm := s.BudgetFor(CurrentMonthKey())
	if bm.TotalAssigned <= 0 {
		t.Fatalf("current month TotalAssigned = %d, want > 0", bm.TotalAssigned)
	}
	// Zero-based: the demo assigns every dollar, so Ready to Assign lands on 0.
	if bm.ReadyToAssign != 0 {
		t.Fatalf("ReadyToAssign = %d, want 0 (demo is fully budgeted)", bm.ReadyToAssign)
	}
	if bm.TotalAvailable <= 0 {
		t.Fatalf("TotalAvailable = %d, want > 0 (money held in envelopes)", bm.TotalAvailable)
	}
	// Off-budget brokerage must NOT inflate the assignable pool: funds should
	// track on-budget accounts only, not total assets.
	if bm.Funds >= s.TotalAssets() {
		t.Fatalf("budget Funds (%d) should exclude the off-budget brokerage and be < TotalAssets (%d)",
			bm.Funds, s.TotalAssets())
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
	// all account balances across the whole chart (every type, including equity
	// such as opening balances and the debt-financing contra) is exactly zero.
	total := 0
	for _, a := range s.accounts {
		total += s.Balance(a.ID)
	}
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
