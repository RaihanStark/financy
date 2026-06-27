package core

import (
	"path/filepath"
	"testing"
	"time"
)

func serialOf(y int, m time.Month, d int) int {
	return TimeToSerial(time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
}

func recurStore() *Store {
	s := &Store{accounts: seedAccounts(), nextID: 1}
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	return s
}

func TestNextDateMonthEnd(t *testing.T) {
	jan31 := serialOf(2026, 1, 31)
	cases := []struct {
		freq string
		want int
	}{
		{"Weekly", serialOf(2026, 2, 7)},
		{"Biweekly", serialOf(2026, 2, 14)},
		{"Monthly", serialOf(2026, 2, 28)},   // clamps 31 → Feb 28
		{"Quarterly", serialOf(2026, 4, 30)}, // clamps 31 → Apr 30
		{"Yearly", serialOf(2027, 1, 31)},
	}
	for _, c := range cases {
		if got := nextDate(jan31, c.freq); got != c.want {
			t.Errorf("nextDate(Jan31, %s) = %s, want %s", c.freq, FmtSerialDate(got), FmtSerialDate(c.want))
		}
	}
	// Leap day yearly → non-leap year clamps to Feb 28.
	if got := nextDate(serialOf(2024, 2, 29), "Yearly"); got != serialOf(2025, 2, 28) {
		t.Errorf("leap yearly = %s, want 2025-02-28", FmtSerialDate(got))
	}
}

func TestRecurringCatchUpAndDuplicate(t *testing.T) {
	s := recurStore()
	start := TodaySerial - 65 // ~2 months ago
	s.AddRecurring(Recurring{Kind: KindExpense, AcctA: "checking", AcctB: "housing",
		Amount: 140000, Payee: "Rent", Freq: "Monthly", NextDue: start, Enabled: true})

	due := s.PendingRecurring(TodaySerial)
	if len(due) < 2 {
		t.Fatalf("expected >=2 catch-up occurrences, got %d", len(due))
	}
	for _, it := range due {
		if len(it.Candidates) != 0 {
			t.Fatal("nothing posted yet — should have no candidate matches")
		}
		if !it.Txn.balanced() {
			t.Fatal("due occurrence not balanced")
		}
	}

	// Record a matching rent payment near the first occurrence → it becomes a
	// candidate for that occurrence.
	s.AddTransaction(Transaction{Date: start + 1, Payee: "Rent",
		Posts: PostingsFor(KindExpense, "checking", "housing", 140000)})
	due = s.PendingRecurring(TodaySerial)
	matched := false
	for _, it := range due {
		if it.Date == start && len(it.Candidates) > 0 {
			matched = true
		}
	}
	if !matched {
		t.Fatal("first occurrence should have a candidate match")
	}
}

func TestPostRecurringAdvancesAndClears(t *testing.T) {
	s := recurStore()
	s.AddRecurring(Recurring{Kind: KindExpense, AcctA: "checking", AcctB: "housing",
		Amount: 140000, Payee: "Rent", Freq: "Monthly", NextDue: TodaySerial - 1, Enabled: true})

	due := s.PendingRecurring(TodaySerial)
	n, err := s.PostRecurring(TodaySerial, due)
	if err != nil || n != len(due) || n == 0 {
		t.Fatalf("PostRecurring posted %d (err %v), want %d", n, err, len(due))
	}
	if got := s.Recurrings()[0].NextDue; got <= TodaySerial {
		t.Fatalf("NextDue not advanced past today: %s", FmtSerialDate(got))
	}
	if rest := s.PendingRecurring(TodaySerial); len(rest) != 0 {
		t.Fatalf("still %d pending after posting", len(rest))
	}
	if len(s.Transactions()) != n {
		t.Fatalf("created %d transactions, want %d", len(s.Transactions()), n)
	}
}

func TestRecurringCandidatesLooseAmount(t *testing.T) {
	date := serialOf(2026, 7, 1)

	// A nearby payment that's slightly off (≈1.4%) still matches confidently.
	s := recurStore()
	s.AddTransaction(Transaction{Date: serialOf(2026, 7, 2), Payee: "Rent",
		Posts: PostingsFor(KindExpense, "checking", "housing", 138000)}) // $1,380 vs $1,400
	cands, auto := s.recurringCandidates("checking", 140000, date, 15)
	if len(cands) != 1 || !auto {
		t.Fatalf("close amount: cands=%d auto=%v, want 1/true", len(cands), auto)
	}

	// A far-off amount is still offered as a candidate, but not pre-selected.
	s2 := recurStore()
	s2.AddTransaction(Transaction{Date: serialOf(2026, 7, 2), Payee: "Something",
		Posts: PostingsFor(KindExpense, "checking", "housing", 50000)}) // $500 vs $1,400
	cands2, auto2 := s2.recurringCandidates("checking", 140000, date, 15)
	if len(cands2) != 1 || auto2 {
		t.Fatalf("far amount: cands=%d auto=%v, want 1/false", len(cands2), auto2)
	}
}

func TestAdvanceRecurring(t *testing.T) {
	s := recurStore()
	s.AddRecurring(Recurring{Kind: KindExpense, AcctA: "checking", AcctB: "housing",
		Amount: 140000, Payee: "Rent", Freq: "Monthly", NextDue: serialOf(2026, 7, 1), Enabled: true})
	id := s.Recurrings()[0].ID
	before := len(s.Transactions())
	if !s.AdvanceRecurring(id) {
		t.Fatal("AdvanceRecurring returned false")
	}
	if len(s.Transactions()) != before {
		t.Fatal("AdvanceRecurring should not create a transaction")
	}
	if got := s.Recurrings()[0].NextDue; got != serialOf(2026, 8, 1) {
		t.Fatalf("NextDue = %s, want 2026-08-01", FmtSerialDate(got))
	}
}

func TestPostRecurringNowEarly(t *testing.T) {
	s := recurStore()
	due := serialOf(2026, 7, 1)
	s.AddRecurring(Recurring{Kind: KindExpense, AcctA: "checking", AcctB: "housing",
		Amount: 140000, Payee: "Rent", Freq: "Monthly", NextDue: due, Enabled: true})

	today := serialOf(2026, 6, 25) // before the due date
	if !s.PostRecurringNow(s.Recurrings()[0].ID, today) {
		t.Fatal("PostRecurringNow returned false")
	}
	// One transaction, dated today (the early payment).
	txns := s.Transactions()
	if len(txns) != 1 || txns[0].Date != today {
		t.Fatalf("expected 1 txn dated %s, got %d txns", FmtSerialDate(today), len(txns))
	}
	// Schedule advanced one period (Jul 1 → Aug 1).
	if got := s.Recurrings()[0].NextDue; got != serialOf(2026, 8, 1) {
		t.Fatalf("NextDue = %s, want 2026-08-01", FmtSerialDate(got))
	}
}

func TestRecurringPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.financy")
	s, err := NewDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	s.AddRecurring(Recurring{Kind: KindIncome, AcctA: "checking", AcctB: "salary",
		Amount: 450000, Payee: "Payroll", Freq: "Monthly", NextDue: TodaySerial, Enabled: true})
	_ = s.Close()

	s2, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s2.Close() }()
	got := s2.Recurrings()
	if len(got) != 1 {
		t.Fatalf("recurring after reopen = %d, want 1", len(got))
	}
	if got[0].Payee != "Payroll" || got[0].Amount != 450000 || !got[0].Enabled || got[0].Freq != "Monthly" {
		t.Fatalf("recurring not persisted correctly: %+v", got[0])
	}
}
