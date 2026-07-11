package core

import "testing"

// revolvingStore creates a store with a credit card debt: 19.99% APR, 5,000,000
// limit, statement on the 5th, min payment 5% with a 50,000 floor, and a
// 1,000,000 starting balance.
func revolvingStore(t *testing.T) (*Store, Debt) {
	t.Helper()
	s := debtStore()
	s.AddDebt(Debt{
		Name: "Visa", Type: DebtRevolving, Lender: "BigBank", AcctMoney: "checking",
		APRBps: 1999, CreditLimit: 5_000_000, StatementDay: 5, PayDueDay: 25,
		MinPayBps: 500, MinPayFloor: 50_000, Principal: 1_000_000,
		PurchaseDate: serialOf(2026, 1, 10),
	}, nil)
	return s, s.Debts()[0]
}

func TestAddRevolvingDebt(t *testing.T) {
	s, d := revolvingStore(t)

	// The card account is created ON-budget — that's what keeps card spending
	// honest in the budget — with the starting balance booked like an opening
	// balance, not a financing origination.
	a := s.AccountByID(d.AcctLiability)
	if a == nil || a.Type != Liability {
		t.Fatalf("card account not created: %v", a)
	}
	if a.OffBudget {
		t.Error("revolving account must be on-budget")
	}
	if !d.OwnsAccount {
		t.Error("OwnsAccount not set for a created account")
	}
	if d.OriginTxnID != "" {
		t.Errorf("revolving debt must have no origination, got %q", d.OriginTxnID)
	}
	if bal := s.DebtBalance(d.ID); bal != 1_000_000 {
		t.Errorf("starting balance = %d, want 1000000", bal)
	}
	if len(s.Installments(d.ID)) != 0 {
		t.Error("revolving debt must have no schedule")
	}
	// The next statement is seeded strictly after today.
	if d.NextStatement <= TodaySerial {
		t.Errorf("NextStatement = %d, want after today %d", d.NextStatement, TodaySerial)
	}
	// No budget envelope: the card is not a budget category.
	for _, c := range s.budgetCategories() {
		if c.ID == d.AcctLiability {
			t.Error("revolving liability must not be a budget envelope")
		}
	}
}

func TestAddRevolvingDebtAttachesExistingAccount(t *testing.T) {
	s := debtStore()
	s.AddAccount(Account{ID: "visa", Name: "Visa", Type: Liability})
	s.AddTransaction(Transaction{Date: serialOf(2026, 1, 5), Payee: "Shop",
		Posts: PostingsFor(KindExpense, "visa", "shopping", 250_000)})

	s.AddDebt(Debt{
		Name: "Visa", Type: DebtRevolving, AcctMoney: "checking", AcctLiability: "visa",
		APRBps: 1999, StatementDay: 5, MinPayBps: 500, MinPayFloor: 50_000,
	}, nil)
	d := s.Debts()[0]
	if d.AcctLiability != "visa" || d.OwnsAccount {
		t.Fatalf("attach failed: %+v", d)
	}
	// The balance is whatever the account already carries.
	if bal := s.DebtBalance(d.ID); bal != 250_000 {
		t.Errorf("attached balance = %d, want 250000", bal)
	}

	// Deleting the debt detaches: the account and its history survive.
	s.DeleteDebt(d.ID)
	if s.AccountByID("visa") == nil {
		t.Error("attached account must survive debt deletion")
	}
	if len(s.Transactions()) != 1 {
		t.Error("card transactions must survive debt deletion")
	}
}

func TestRevolvingPaymentIsBudgetNeutralTransfer(t *testing.T) {
	s, d := revolvingStore(t)

	if !s.PayDebtAmount(d.ID, 300_000, serialOf(2026, 2, 1)) {
		t.Fatal("PayDebtAmount returned false")
	}
	if bal := s.DebtBalance(d.ID); bal != 700_000 {
		t.Errorf("balance after payment = %d, want 700000", bal)
	}
	if bal := s.Balance("checking"); bal != -300_000 {
		t.Errorf("checking = %d, want -300000", bal)
	}
	// A payment is a transfer — no expense category is touched, so the budget's
	// activity is untouched (the spending already hit envelopes at purchase).
	idx := s.spendIndex()
	for k, v := range idx {
		if v != 0 {
			t.Errorf("budget activity from a card payment: %s = %d", k, v)
		}
	}
}

func TestMinPaymentDue(t *testing.T) {
	s, d := revolvingStore(t)

	// 5% of 1,000,000 = 50,000, equal to the floor.
	if got := s.MinPaymentDue(d); got != 50_000 {
		t.Errorf("min payment = %d, want 50000", got)
	}
	// Pay it down to 400,000: 5% = 20,000 < floor → floor applies.
	s.PayDebtAmount(d.ID, 600_000, 0)
	if got := s.MinPaymentDue(*s.DebtByID(d.ID)); got != 50_000 {
		t.Errorf("min payment = %d, want floor 50000", got)
	}
	// Below the floor the balance itself is the minimum.
	s.PayDebtAmount(d.ID, 370_000, 0)
	if got := s.MinPaymentDue(*s.DebtByID(d.ID)); got != 30_000 {
		t.Errorf("min payment = %d, want the 30000 balance", got)
	}
	// Zero balance → nothing due.
	s.PayDebtAmount(d.ID, 30_000, 0)
	if got := s.MinPaymentDue(*s.DebtByID(d.ID)); got != 0 {
		t.Errorf("min payment on zero balance = %d, want 0", got)
	}
}

func TestStatementReviewFlow(t *testing.T) {
	s, d := revolvingStore(t)
	stmt := s.DebtByID(d.ID).NextStatement

	// Nothing pending before the statement date; pending on/after it — and a
	// months-long gap yields one entry per missed statement.
	if got := s.PendingStatements(stmt - 1); len(got) != 0 {
		t.Fatalf("pending before statement = %d, want 0", len(got))
	}
	pend := s.PendingStatements(stmt)
	if len(pend) != 1 {
		t.Fatalf("pending on statement = %d, want 1", len(pend))
	}
	// Proposed interest: one month at 19.99% on 1,000,000 → 16,658.
	if pend[0].Interest != PeriodInterest(1_000_000, 1999, 12) {
		t.Errorf("proposed interest = %d, want %d", pend[0].Interest, PeriodInterest(1_000_000, 1999, 12))
	}
	threeLater := NextDayOfMonth(NextDayOfMonth(NextDayOfMonth(stmt, 5), 5), 5)
	if got := s.PendingStatements(threeLater); len(got) != 4 {
		t.Errorf("pending after 3 missed months = %d, want 4", len(got))
	}

	// Posting the charge grows the card balance and expenses the interest,
	// then advances to the next month.
	if !s.PostStatement(d.ID, 16_658, stmt) {
		t.Fatal("PostStatement returned false")
	}
	if bal := s.DebtBalance(d.ID); bal != 1_016_658 {
		t.Errorf("balance after interest = %d, want 1016658", bal)
	}
	if bal := s.Balance("loanfees"); bal != 16_658 {
		t.Errorf("interest expense = %d, want 16658", bal)
	}
	if got := s.DebtByID(d.ID).NextStatement; got != NextDayOfMonth(stmt, 5) {
		t.Errorf("NextStatement = %d, want %d", got, NextDayOfMonth(stmt, 5))
	}
	// Checking stays untouched — interest is paid *from* the card.
	if bal := s.Balance("checking"); bal != 0 {
		t.Errorf("checking = %d, want 0", bal)
	}

	// Skip advances without posting.
	next := s.DebtByID(d.ID).NextStatement
	if !s.SkipStatement(d.ID) {
		t.Fatal("SkipStatement returned false")
	}
	if got := s.DebtByID(d.ID).NextStatement; got != NextDayOfMonth(next, 5) {
		t.Errorf("NextStatement after skip = %d, want %d", got, NextDayOfMonth(next, 5))
	}
	if got := len(s.Transactions()); got != 2 { // opening + interest
		t.Errorf("transactions = %d, want 2 (skip posts nothing)", got)
	}
}

func TestNextDayOfMonth(t *testing.T) {
	jan10 := serialOf(2026, 1, 10)
	if got := NextDayOfMonth(jan10, 15); got != serialOf(2026, 1, 15) {
		t.Errorf("next 15th after Jan 10 = %s, want Jan 15", FmtSerialDate(got))
	}
	if got := NextDayOfMonth(jan10, 5); got != serialOf(2026, 2, 5) {
		t.Errorf("next 5th after Jan 10 = %s, want Feb 5", FmtSerialDate(got))
	}
	// Strictly after: the same day rolls to next month.
	if got := NextDayOfMonth(serialOf(2026, 1, 5), 5); got != serialOf(2026, 2, 5) {
		t.Errorf("next 5th after Jan 5 = %s, want Feb 5", FmtSerialDate(got))
	}
	// Day clamped to 28.
	if got := NextDayOfMonth(serialOf(2026, 1, 30), 31); got != serialOf(2026, 2, 28) {
		t.Errorf("clamped day = %s, want Feb 28", FmtSerialDate(got))
	}
}

func TestInformalDebtLifecycle(t *testing.T) {
	s := debtStore()
	due := serialOf(2026, 3, 1)
	s.AddDebt(Debt{
		Name: "Loan from Andi", Type: DebtInformal, Lender: "Andi",
		AcctMoney: "checking", Principal: 500_000, DueDate: due,
		PurchaseDate: serialOf(2026, 1, 1),
	}, nil)
	d := s.Debts()[0]

	// Owed outright, no schedule, envelope-backed like BNPL.
	if bal := s.DebtBalance(d.ID); bal != 500_000 {
		t.Errorf("balance = %d, want 500000", bal)
	}
	if len(s.Installments(d.ID)) != 0 {
		t.Error("informal debt must have no schedule")
	}
	if a := s.AccountByID(d.AcctLiability); a == nil || !a.OffBudget {
		t.Error("informal liability must be off-budget")
	}
	found := false
	for _, c := range s.budgetCategories() {
		if c.ID == d.AcctLiability {
			found = true
		}
	}
	if !found {
		t.Error("informal debt must have a budget envelope")
	}

	// Pay arbitrary amounts; overpayment clamps to what's owed.
	if !s.PayDebtAmount(d.ID, 200_000, serialOf(2026, 1, 15)) {
		t.Fatal("PayDebtAmount returned false")
	}
	if bal := s.DebtBalance(d.ID); bal != 300_000 {
		t.Errorf("balance after payment = %d, want 300000", bal)
	}
	if !s.PayDebtAmount(d.ID, 999_999, serialOf(2026, 1, 20)) {
		t.Fatal("overpayment should clamp, not fail")
	}
	if bal := s.DebtBalance(d.ID); bal != 0 {
		t.Errorf("balance after clamped payoff = %d, want 0", bal)
	}
	if bal := s.Balance("checking"); bal != -500_000 {
		t.Errorf("checking = %d, want -500000", bal)
	}
	// Paid off → nothing more to pay.
	if s.PayDebtAmount(d.ID, 1000, 0) {
		t.Error("payment on a settled informal debt must fail")
	}
}

func TestInformalDebtDepositedTo(t *testing.T) {
	s := debtStore()
	// Andi wired 500,000 into checking: the origination must raise BOTH the
	// asset and the liability, so net worth is unchanged by borrowing.
	s.AddDebt(Debt{
		Name: "Cash from Andi", Type: DebtInformal, Lender: "Andi",
		AcctMoney: "checking", Principal: 500_000, AcctOrigin: "checking",
	}, nil)
	d := s.Debts()[0]
	if bal := s.Balance("checking"); bal != 500_000 {
		t.Errorf("checking after borrowing = %d, want 500000", bal)
	}
	if bal := s.DebtBalance(d.ID); bal != 500_000 {
		t.Errorf("owed = %d, want 500000", bal)
	}
	if nw := s.NetWorth(); nw != 0 {
		t.Errorf("net worth after borrowing = %d, want 0", nw)
	}
}

func TestDueRollupsAcrossTypes(t *testing.T) {
	s, d := revolvingStore(t)
	due := serialOf(2026, 3, 1)
	s.AddDebt(Debt{Name: "IOU", Type: DebtInformal, AcctMoney: "checking",
		Principal: 200_000, DueDate: due}, nil)
	s.AddDebt(Debt{Name: "Phone", Type: DebtBNPL, AcctMoney: "checking"},
		GenerateInstallments(300_000, 3, serialOf(2026, 2, 1), "Monthly"))

	// Total owed spans all kinds: card 1,000,000 + IOU 200,000 + BNPL 300,000.
	if got := s.TotalDebtOutstanding(); got != 1_500_000 {
		t.Errorf("TotalDebtOutstanding = %d, want 1500000", got)
	}
	// As of the IOU due date: 2 BNPL installments due + informal due + the
	// card's pending statement(s) all count.
	stmt := s.DebtByID(d.ID).NextStatement
	asOf := due
	if stmt > asOf {
		asOf = stmt
	}
	if got := s.DueDebtCount(asOf); got < 3 {
		t.Errorf("DueDebtCount = %d, want at least 3", got)
	}
	b := s.DebtBuckets(due - 10)
	if b.Outstanding != 1_500_000 {
		t.Errorf("buckets outstanding = %d, want 1500000", b.Outstanding)
	}
}

func TestDeleteDebtRestoresLinkedTransaction(t *testing.T) {
	s := debtStore()
	first := serialOf(2026, 1, 1)
	s.AddDebt(Debt{Name: "Phone", Type: DebtBNPL, AcctMoney: "checking"},
		GenerateInstallments(12000, 12, first, "Monthly"))
	id := s.Debts()[0].ID

	// A user-authored transaction linked to an installment…
	extID := s.genID()
	s.AddTransaction(Transaction{
		ID: extID, Date: first, Payee: "PayLater",
		Posts: PostingsFor(KindExpense, "checking", "shopping", 1000),
	})
	s.LinkInstallment(s.Installments(id)[0].ID, extID)

	// …must survive deleting the whole debt, restored to its original postings.
	s.DeleteDebt(id)
	txn := s.TxnByID(extID)
	if txn == nil {
		t.Fatal("linked transaction was destroyed by DeleteDebt")
	}
	if bal := s.Balance("shopping"); bal != 1000 {
		t.Errorf("shopping = %d, want 1000 (original posting restored)", bal)
	}
	if got := len(s.Transactions()); got != 1 {
		t.Errorf("transactions after delete = %d, want 1 (just the user's)", got)
	}
}
