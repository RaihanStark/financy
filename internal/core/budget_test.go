package core

import (
	"path/filepath"
	"testing"
)

func TestBudgetPersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "budget.financy")
	s, err := NewDocument(path)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	s.SetAssigned("2026-06", "groceries", 450)
	s.SetAssigned("2026-06", "housing", 1200)
	s.SetAssigned("2026-07", "groceries", 500)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = s2.Close() }()
	if got := s2.Assigned("2026-06", "groceries"); got != 450 {
		t.Fatalf("after reopen Assigned(June,groceries) = %d, want 450", got)
	}
	if got := s2.Assigned("2026-06", "housing"); got != 1200 {
		t.Fatalf("after reopen Assigned(June,housing) = %d, want 1200", got)
	}
	if got := s2.Assigned("2026-07", "groceries"); got != 500 {
		t.Fatalf("after reopen Assigned(July,groceries) = %d, want 500", got)
	}

	// Clearing persists as a deletion.
	s2.SetAssigned("2026-06", "groceries", 0)
	if err := s2.Close(); err != nil {
		t.Fatalf("Close 2: %v", err)
	}
	s3, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore 2: %v", err)
	}
	defer func() { _ = s3.Close() }()
	if got := s3.Assigned("2026-06", "groceries"); got != 0 {
		t.Fatalf("cleared assignment came back as %d, want 0", got)
	}
}

// budgetStore returns an empty in-memory store with a cash asset account and the
// seeded categories, ready for budgeting tests.
func budgetStore() *Store {
	s := NewStore()
	s.accounts = append(s.accounts, Account{ID: "cash", Name: "Cash", Type: Asset})
	return s
}

// spend records an outflow from cash into an expense category on a date.
func (s *Store) spend(date, cat string, amt int) {
	s.AddTransaction(Transaction{
		Date:  ParseDateSerial(date),
		Posts: []Posting{P("cash", -amt), P(cat, amt)},
	})
}

// earn records income into cash on a date.
func (s *Store) earn(date string, amt int) {
	s.AddTransaction(Transaction{
		Date:  ParseDateSerial(date),
		Posts: []Posting{P("cash", amt), P("salary", -amt)},
	})
}

func TestAssignedRoundTripAndClear(t *testing.T) {
	s := budgetStore()
	s.SetAssigned("2026-06", "groceries", 500)
	if got := s.Assigned("2026-06", "groceries"); got != 500 {
		t.Fatalf("Assigned = %d, want 500", got)
	}
	// Setting to zero clears it.
	s.SetAssigned("2026-06", "groceries", 0)
	if got := s.Assigned("2026-06", "groceries"); got != 0 {
		t.Fatalf("after clear Assigned = %d, want 0", got)
	}
	if _, ok := s.assignments[monthKey("2026-06", "groceries")]; ok {
		t.Fatal("zero assignment should be deleted from the map")
	}
}

func TestPreviewAutoAssign(t *testing.T) {
	s := budgetStore()
	// Last month (June): groceries and housing assigned.
	s.SetAssigned("2026-06", "groceries", 450)
	s.SetAssigned("2026-06", "housing", 1200)
	// This month (July): groceries already matches, housing differs, and a
	// third category is assigned that had nothing last month.
	s.SetAssigned("2026-07", "groceries", 450)
	s.SetAssigned("2026-07", "housing", 900)
	s.SetAssigned("2026-07", "dining", 60)

	changes := s.PreviewAutoAssign("2026-07")

	// groceries: unchanged (450 -> 450) → excluded.
	// housing: 900 -> 1200 → included as an overwrite.
	// dining: had 0 last month → excluded (auto-assign never clears).
	if len(changes) != 1 {
		t.Fatalf("PreviewAutoAssign returned %d changes, want 1: %+v", len(changes), changes)
	}
	c := changes[0]
	if c.CatID != "housing" || c.From != 900 || c.To != 1200 {
		t.Fatalf("change = %+v, want housing 900 -> 1200", c)
	}

	// Preview must not mutate anything.
	if got := s.Assigned("2026-07", "housing"); got != 900 {
		t.Fatalf("preview mutated housing to %d, want 900", got)
	}

	// Applying then previewing again yields no further changes.
	s.AssignLastMonthsAmounts("2026-07")
	if got := s.Assigned("2026-07", "housing"); got != 1200 {
		t.Fatalf("after apply housing = %d, want 1200", got)
	}
	if again := s.PreviewAutoAssign("2026-07"); len(again) != 0 {
		t.Fatalf("preview after apply returned %d changes, want 0: %+v", len(again), again)
	}
}

func TestAvailableRollsForward(t *testing.T) {
	s := budgetStore()
	// May: assign 100, spend 30.
	s.SetAssigned("2026-05", "groceries", 100)
	s.spend("2026-05-10", "groceries", 30)
	// June: assign 50, spend 40.
	s.SetAssigned("2026-06", "groceries", 50)
	s.spend("2026-06-05", "groceries", 40)

	may := findCat(t, s.BudgetFor("2026-05"), "groceries")
	if may.Assigned != 100 || may.Activity != -30 || may.Available != 70 {
		t.Fatalf("May = %+v, want assigned=100 activity=-30 available=70", may)
	}

	jun := findCat(t, s.BudgetFor("2026-06"), "groceries")
	if jun.Assigned != 50 || jun.Activity != -40 {
		t.Fatalf("June flows = %+v, want assigned=50 activity=-40", jun)
	}
	// Available rolls forward: (100+50) assigned − (30+40) spent = 80.
	if jun.Available != 80 {
		t.Fatalf("June Available = %d, want 80 (rollover)", jun.Available)
	}
}

func TestOverspendingGoesNegative(t *testing.T) {
	s := budgetStore()
	s.SetAssigned("2026-06", "dining", 20)
	s.spend("2026-06-12", "dining", 75)
	c := findCat(t, s.BudgetFor("2026-06"), "dining")
	if c.Available != -55 {
		t.Fatalf("Available = %d, want -55", c.Available)
	}
	if !c.Overspent() {
		t.Fatal("expected Overspent() to be true")
	}
}

func TestOverspendDoesNotRollNegative(t *testing.T) {
	s := budgetStore()
	s.earn("2026-06-01", 1000)
	// June: assign 20 to dining, spend 75 → overspent by 55 this month.
	s.SetAssigned("2026-06", "dining", 20)
	s.spend("2026-06-12", "dining", 75)

	jun := findCat(t, s.BudgetFor("2026-06"), "dining")
	if jun.Available != -55 {
		t.Fatalf("June dining Available = %d, want -55 (red this month)", jun.Available)
	}

	// July: the cash overspend is absorbed — the envelope resets, it does NOT
	// carry the −55 forward.
	jul := findCat(t, s.BudgetFor("2026-07"), "dining")
	if jul.Available != 0 {
		t.Fatalf("July dining Available = %d, want 0 (overspend absorbed, not rolled)", jul.Available)
	}

	// The absorbed overspend reduces Ready to Assign: 1000 funds − 75 spent − 0
	// in envelopes = 925 left to assign in July.
	if rta := s.ReadyToAssign("2026-07"); rta != 925 {
		t.Fatalf("July ReadyToAssign = %d, want 925 (overspend covered from funds)", rta)
	}
}

func TestReadyToAssign(t *testing.T) {
	s := budgetStore()
	s.earn("2026-06-01", 1000)
	s.SetAssigned("2026-06", "groceries", 300)
	s.SetAssigned("2026-06", "housing", 400)

	// 1000 received − 700 assigned = 300 still to assign.
	if rta := s.ReadyToAssign("2026-06"); rta != 300 {
		t.Fatalf("ReadyToAssign = %d, want 300", rta)
	}
	// It also surfaces on the month view.
	if got := s.BudgetFor("2026-06").ReadyToAssign; got != 300 {
		t.Fatalf("BudgetFor RTA = %d, want 300", got)
	}
}

func TestReadyToAssignIsGlobalAcrossMonths(t *testing.T) {
	s := budgetStore()
	s.earn("2026-05-01", 1000)
	s.SetAssigned("2026-05", "groceries", 200)

	// Ready to Assign is one global pool: 1000 funds − 200 committed = 800, the
	// same figure whatever month you view.
	for _, m := range []string{"2026-05", "2026-06", "2026-09"} {
		if rta := s.BudgetFor(m).ReadyToAssign; rta != 800 {
			t.Fatalf("ReadyToAssign viewing %s = %d, want 800", m, rta)
		}
	}
}

func TestFutureAssignmentReducesReadyToAssign(t *testing.T) {
	s := budgetStore()
	s.earn("2026-06-01", 1000) // 1000 received in June

	if rta := s.BudgetFor("2026-06").ReadyToAssign; rta != 1000 {
		t.Fatalf("June ReadyToAssign before assigning = %d, want 1000", rta)
	}

	// Assign all of it in JULY (a future month). Because the pool is global, June's
	// Ready to Assign must drop to 0 — that money already has a job.
	s.SetAssigned("2026-07", "groceries", 1000)
	if rta := s.BudgetFor("2026-06").ReadyToAssign; rta != 0 {
		t.Fatalf("June ReadyToAssign after assigning in July = %d, want 0", rta)
	}
	if rta := s.BudgetFor("2026-07").ReadyToAssign; rta != 0 {
		t.Fatalf("July ReadyToAssign = %d, want 0", rta)
	}
}

func TestOpeningBalanceFundsBudget(t *testing.T) {
	s := budgetStore()
	// Opening balance of an asset: cash +500, opening −500 (as AccountForm does).
	s.AddTransaction(Transaction{
		Date:  ParseDateSerial("2026-06-01"),
		Payee: "Opening Balance",
		Posts: []Posting{P("cash", 500), P("opening", -500)},
	})
	if rta := s.ReadyToAssign("2026-06"); rta != 500 {
		t.Fatalf("ReadyToAssign from opening balance = %d, want 500", rta)
	}
}

func TestOffBudgetFlagPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ob.financy")
	s, err := NewDocument(path)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	s.AddAccount(Account{Name: "Brokerage", Type: Asset, OffBudget: true})
	s.AddAccount(Account{Name: "Checking", Type: Asset})
	_ = s.Close()

	s2, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = s2.Close() }()
	if a := s2.AccountByName("Brokerage"); a == nil || !a.OffBudget {
		t.Fatal("Brokerage should round-trip as off-budget")
	}
	if a := s2.AccountByName("Checking"); a == nil || a.OffBudget {
		t.Fatal("Checking should round-trip as on-budget")
	}
}

func TestCreditCardSpendingIsBudgetNeutral(t *testing.T) {
	s := budgetStore()
	s.accounts = append(s.accounts, Account{ID: "visa", Name: "Visa", Type: Liability})
	s.AddTransaction(Transaction{Date: ParseDateSerial("2026-06-01"), Payee: "open",
		Posts: []Posting{P("cash", 1000), P("opening", -1000)}})
	s.SetAssigned("2026-06", "groceries", 100)

	before := s.ReadyToAssign("2026-06")
	if before != 900 {
		t.Fatalf("RTA before credit purchase = %d, want 900", before)
	}

	// Buy 100 of groceries on the credit card: category +100, visa −100.
	s.AddTransaction(Transaction{Date: ParseDateSerial("2026-06-10"), Payee: "store",
		Posts: []Posting{P("groceries", 100), P("visa", -100)}})

	// The envelope is spent, but Ready to Assign must NOT change — the assigned
	// money didn't come back to us, it became a card balance we still owe.
	g := s.BudgetFor("2026-06")
	if av := findCat(t, g, "groceries").Available; av != 0 {
		t.Fatalf("groceries Available after spend = %d, want 0", av)
	}
	if g.ReadyToAssign != 900 {
		t.Fatalf("RTA after credit purchase = %d, want 900 (must stay put)", g.ReadyToAssign)
	}

	// Excluding the card (tracking account) brings the phantom money back —
	// confirms the flag is what drives the behaviour.
	s.accounts[len(s.accounts)-1].OffBudget = true
	if rta := s.ReadyToAssign("2026-06"); rta != 1000 {
		t.Fatalf("RTA with card off-budget = %d, want 1000", rta)
	}
}

func TestDebtIsBudgetEnvelope(t *testing.T) {
	s := budgetStore()
	// Fund the budget with 1000 of cash (opening balance).
	s.AddTransaction(Transaction{Date: ParseDateSerial("2026-06-01"), Payee: "open",
		Posts: []Posting{P("cash", 1000), P("opening", -1000)}})
	if rta := s.ReadyToAssign("2026-06"); rta != 1000 {
		t.Fatalf("RTA after funding = %d, want 1000", rta)
	}

	// Take on a debt: 120 over 12 monthly installments, first due in June.
	first := ParseDateSerial("2026-06-15")
	s.AddDebt(Debt{Name: "Phone", Type: DebtBNPL, Lender: "PayLater", AcctMoney: "cash"},
		GenerateInstallments(120, 12, first, "Monthly"))
	d := s.Debts()[0]

	// Origination is balance-sheet only: Ready to Assign must NOT move — you took
	// on a tracked liability, you didn't spend budget money.
	if rta := s.ReadyToAssign("2026-06"); rta != 1000 {
		t.Fatalf("RTA after origination = %d, want 1000 (financing isn't budget spending)", rta)
	}

	// The debt shows up as its own budget envelope, keyed to its liability account.
	if env := findCat(t, s.BudgetFor("2026-06"), d.AcctLiability); !env.IsDebt {
		t.Fatalf("debt category should be flagged IsDebt")
	}

	// Assign 10 to the debt envelope (RTA drops by the assignment), then pay the
	// first installment of 10.
	s.SetAssigned("2026-06", d.AcctLiability, 10)
	if rta := s.ReadyToAssign("2026-06"); rta != 990 {
		t.Fatalf("RTA after assigning 10 to the debt = %d, want 990", rta)
	}
	s.PayInstallment(s.Installments(d.ID)[0].ID, first)

	// Paying is budgeted spending: the envelope nets to zero and Ready to Assign
	// stays put (the payment drew from the envelope, not from unassigned funds).
	env := findCat(t, s.BudgetFor("2026-06"), d.AcctLiability)
	if env.Activity != -10 {
		t.Fatalf("debt envelope Activity after payment = %d, want -10", env.Activity)
	}
	if env.Available != 0 {
		t.Fatalf("debt envelope Available = %d, want 0 (assigned 10, paid 10)", env.Available)
	}
	if rta := s.ReadyToAssign("2026-06"); rta != 990 {
		t.Fatalf("RTA after paying installment = %d, want 990 (payment was budgeted)", rta)
	}
}

func TestReadyToAssignTracksTotalAssets(t *testing.T) {
	s := budgetStore()
	s.accounts = append(s.accounts, Account{ID: "savings", Name: "Savings", Type: Asset})
	s.earn("2026-06-01", 1000) // cash +1000

	// Moving money between your own accounts must not change what's assignable.
	s.AddTransaction(Transaction{
		Date:  ParseDateSerial("2026-06-02"),
		Payee: "Transfer to savings",
		Posts: []Posting{P("cash", -200), P("savings", 200)},
	})
	s.SetAssigned("2026-06", "groceries", 300)

	// Total assets = 1000 (split across cash + savings); 300 assigned → 700 ready.
	if rta := s.ReadyToAssign("2026-06"); rta != 700 {
		t.Fatalf("ReadyToAssign = %d, want 700 (assets 1000 − assigned 300)", rta)
	}

	// The identity holds: assets = Ready to Assign + everything in envelopes.
	bm := s.BudgetFor("2026-06")
	if bm.ReadyToAssign+bm.TotalAvailable != s.TotalAssets() {
		t.Fatalf("RTA(%d) + Available(%d) = %d, want TotalAssets %d",
			bm.ReadyToAssign, bm.TotalAvailable, bm.ReadyToAssign+bm.TotalAvailable, s.TotalAssets())
	}
}

func TestAutoAssignLastMonth(t *testing.T) {
	s := budgetStore()
	s.SetAssigned("2026-05", "groceries", 250)
	s.SetAssigned("2026-05", "housing", 800)

	n := s.AssignLastMonthsAmounts("2026-06")
	if n != 2 {
		t.Fatalf("AssignLastMonthsAmounts copied %d, want 2", n)
	}
	if s.Assigned("2026-06", "groceries") != 250 || s.Assigned("2026-06", "housing") != 800 {
		t.Fatal("June assignments should mirror May")
	}
}

func TestQuickAssignSuggestions(t *testing.T) {
	s := budgetStore()
	s.SetAssigned("2026-05", "groceries", 250)
	s.spend("2026-05-15", "groceries", 220)
	s.spend("2026-04-15", "groceries", 200)
	s.spend("2026-03-15", "groceries", 180)

	if got := s.AssignedLastMonth("2026-06", "groceries"); got != 250 {
		t.Fatalf("AssignedLastMonth = %d, want 250", got)
	}
	if got := s.SpentLastMonth("2026-06", "groceries"); got != 220 {
		t.Fatalf("SpentLastMonth = %d, want 220", got)
	}
	// Average over Mar/Apr/May = (180+200+220)/3 = 200.
	if got := s.AverageSpent("2026-06", "groceries", 3); got != 200 {
		t.Fatalf("AverageSpent(3) = %d, want 200", got)
	}
}

func TestMonthEditableLocksPastMonths(t *testing.T) {
	now := CurrentMonthKey()
	if !MonthEditable(now) {
		t.Fatal("current month should be editable")
	}
	if !MonthEditable(ShiftMonth(now, 1)) {
		t.Fatal("future month should be editable")
	}
	if MonthEditable(ShiftMonth(now, -1)) {
		t.Fatal("last month should be locked")
	}
	if MonthEditable(ShiftMonth(now, -12)) {
		t.Fatal("a year ago should be locked")
	}
}

func TestMonthHelpers(t *testing.T) {
	if got := ShiftMonth("2026-06", -1); got != "2026-05" {
		t.Fatalf("ShiftMonth(-1) = %q, want 2026-05", got)
	}
	if got := ShiftMonth("2026-12", 1); got != "2027-01" {
		t.Fatalf("ShiftMonth(+1 across year) = %q, want 2027-01", got)
	}
	if got := FmtMonthLong("2026-06"); got != "June 2026" {
		t.Fatalf("FmtMonthLong = %q, want June 2026", got)
	}
}

func findCat(t *testing.T, bm BudgetMonth, id string) BudgetCategory {
	t.Helper()
	for _, c := range bm.Categories {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("category %q not found in budget month", id)
	return BudgetCategory{}
}
