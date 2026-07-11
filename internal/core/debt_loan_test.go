package core

import "testing"

// loanStore creates a store with a 1,000,000 @ 12% APR × 12 monthly loan.
func loanStore(t *testing.T) (*Store, Debt, []Installment) {
	t.Helper()
	s := debtStore()
	first := serialOf(2026, 1, 1)
	insts, ok := GenerateLoanSchedule(1_000_000, 1200, 88_849, first, "Monthly")
	if !ok {
		t.Fatal("schedule not generated")
	}
	s.AddDebt(Debt{
		Name: "Car", Type: DebtLoan, Lender: "Bank", AcctMoney: "checking",
		APRBps: 1200, Freq: "Monthly", PaymentAmount: 88_849,
		PurchaseDate: serialOf(2025, 12, 15),
	}, insts)
	d := s.Debts()[0]
	return s, d, s.Installments(d.ID)
}

// unpaidPrincipal sums the principal still scheduled — the invariant partner of
// the liability balance.
func unpaidPrincipal(s *Store, debtID string) int {
	sum := 0
	for _, in := range s.Installments(debtID) {
		if !in.Paid {
			sum += in.Principal
		}
	}
	return sum
}

func TestAddLoanBooksPrincipalOnly(t *testing.T) {
	s, d, insts := loanStore(t)

	// The origination books the principal — never the interest, which isn't owed
	// until it accrues. Total scheduled payments exceed principal by the interest.
	if bal := s.DebtBalance(d.ID); bal != 1_000_000 {
		t.Errorf("opening balance = %d, want 1000000 (principal only)", bal)
	}
	if d.Principal != 1_000_000 {
		t.Errorf("stored principal = %d, want 1000000", d.Principal)
	}
	if d.AcctInterest != "loanfees" {
		t.Errorf("default interest category = %q, want loanfees", d.AcctInterest)
	}
	st := s.DebtStatus(d.ID, serialOf(2026, 1, 1))
	if st.Total <= 1_000_000 {
		t.Errorf("schedule total = %d, should exceed principal by the interest", st.Total)
	}
	if st.Total != 1_000_000+ScheduleInterestTotal(insts) {
		t.Errorf("schedule total %d != principal + interest %d", st.Total, 1_000_000+ScheduleInterestTotal(insts))
	}
	if bal := s.Balance("loanfees"); bal != 0 {
		t.Errorf("interest expensed at origination: %d, want 0", bal)
	}
}

func TestPayLoanInstallmentSplits(t *testing.T) {
	s, d, insts := loanStore(t)

	if !s.PayInstallment(insts[0].ID, insts[0].DueDate) {
		t.Fatal("PayInstallment returned false")
	}
	// Money out is the full payment; the liability draws down by principal only;
	// the interest lands in the expense category.
	if bal := s.Balance("checking"); bal != -88_849 {
		t.Errorf("checking = %d, want -88849", bal)
	}
	if bal := s.DebtBalance(d.ID); bal != 1_000_000-78_849 {
		t.Errorf("loan balance = %d, want %d (drawn by principal)", bal, 1_000_000-78_849)
	}
	if bal := s.Balance("loanfees"); bal != 10_000 {
		t.Errorf("interest expense = %d, want 10000", bal)
	}
	// The payment transaction is a balanced 3-leg split.
	txn := s.TxnByID(s.Installments(d.ID)[0].TxnID)
	if txn == nil || len(txn.Posts) != 3 {
		t.Fatalf("payment txn = %+v, want 3 postings", txn)
	}
	// Invariant: liability balance always equals the principal still scheduled.
	if s.DebtBalance(d.ID) != unpaidPrincipal(s, d.ID) {
		t.Errorf("balance %d != unpaid principal %d", s.DebtBalance(d.ID), unpaidPrincipal(s, d.ID))
	}

	// Undo restores everything.
	if !s.UnpayInstallment(insts[0].ID) {
		t.Fatal("UnpayInstallment returned false")
	}
	if s.Balance("checking") != 0 || s.Balance("loanfees") != 0 || s.DebtBalance(d.ID) != 1_000_000 {
		t.Errorf("undo incomplete: checking=%d loanfees=%d balance=%d",
			s.Balance("checking"), s.Balance("loanfees"), s.DebtBalance(d.ID))
	}
}

func TestPayLoanToCompletion(t *testing.T) {
	s, d, insts := loanStore(t)
	for _, in := range insts {
		if !s.PayInstallment(in.ID, in.DueDate) {
			t.Fatalf("pay %d failed", in.Seq)
		}
	}
	// The final payment absorbs all rounding residue: the loan lands on exactly 0.
	if bal := s.DebtBalance(d.ID); bal != 0 {
		t.Errorf("balance after full payoff = %d, want exactly 0", bal)
	}
	if got, want := s.Balance("loanfees"), ScheduleInterestTotal(insts); got != want {
		t.Errorf("total interest expensed = %d, want %d", got, want)
	}
}

func TestLinkLoanInstallmentInterestFirst(t *testing.T) {
	s, d, insts := loanStore(t)

	// The user recorded the payment manually for the exact scheduled amount.
	extID := s.genID()
	s.AddTransaction(Transaction{
		ID: extID, Date: insts[0].DueDate, Payee: "Bank",
		Posts: PostingsFor(KindExpense, "checking", "misc", 88_849),
	})
	if !s.LinkInstallment(insts[0].ID, extID) {
		t.Fatal("LinkInstallment returned false")
	}
	if bal := s.DebtBalance(d.ID); bal != 1_000_000-78_849 {
		t.Errorf("balance after link = %d, want %d", bal, 1_000_000-78_849)
	}
	if bal := s.Balance("loanfees"); bal != 10_000 {
		t.Errorf("interest after link = %d, want 10000", bal)
	}
	if bal := s.Balance("misc"); bal != 0 {
		t.Errorf("misc after link = %d, want 0 (re-pointed)", bal)
	}

	// Undo restores the user's original posting.
	if !s.UnpayInstallment(insts[0].ID) {
		t.Fatal("UnpayInstallment returned false")
	}
	if bal := s.Balance("misc"); bal != 88_849 {
		t.Errorf("misc after undo = %d, want 88849", bal)
	}
}

func TestLinkLoanInstallmentShortPayment(t *testing.T) {
	s, d, insts := loanStore(t)

	// A payment smaller than the scheduled interest is all interest — the
	// principal leg is clamped to zero, never negative.
	extID := s.genID()
	s.AddTransaction(Transaction{
		ID: extID, Date: insts[0].DueDate, Payee: "Bank",
		Posts: PostingsFor(KindExpense, "checking", "misc", 4_000),
	})
	if !s.LinkInstallment(insts[0].ID, extID) {
		t.Fatal("LinkInstallment returned false")
	}
	if bal := s.DebtBalance(d.ID); bal != 1_000_000 {
		t.Errorf("balance = %d, want 1000000 (nothing hit principal)", bal)
	}
	if bal := s.Balance("loanfees"); bal != 4_000 {
		t.Errorf("interest = %d, want 4000", bal)
	}
	if txn := s.TxnByID(extID); txn == nil || !txn.balanced() {
		t.Errorf("linked txn unbalanced: %+v", txn)
	}
}

func TestApplyExtraPaymentShortensTerm(t *testing.T) {
	s, d, insts := loanStore(t)
	s.PayInstallment(insts[0].ID, insts[0].DueDate)
	balBefore := s.DebtBalance(d.ID)

	// 200k extra, keeping the regular payment → fewer remaining installments.
	if !s.ApplyExtraPayment(d.ID, 200_000, serialOf(2026, 1, 15), true) {
		t.Fatal("ApplyExtraPayment returned false")
	}
	if bal := s.DebtBalance(d.ID); bal != balBefore-200_000 {
		t.Errorf("balance = %d, want %d", bal, balBefore-200_000)
	}
	// The unpaid schedule regenerates from the new balance and still sums exactly.
	if got := unpaidPrincipal(s, d.ID); got != balBefore-200_000 {
		t.Errorf("unpaid principal = %d, want %d", got, balBefore-200_000)
	}
	var unpaid []Installment
	for _, in := range s.Installments(d.ID) {
		if !in.Paid {
			unpaid = append(unpaid, in)
		}
	}
	if len(unpaid) >= 11 {
		t.Errorf("term didn't shorten: %d unpaid installments, want < 11", len(unpaid))
	}
	// Payment unchanged; paid history intact; sequences continue after it.
	if got := s.DebtByID(d.ID).PaymentAmount; got != 88_849 {
		t.Errorf("payment changed to %d, want 88849 (shorten-term keeps it)", got)
	}
	if first := s.Installments(d.ID)[0]; !first.Paid || first.Seq != 1 {
		t.Errorf("paid history disturbed: %+v", first)
	}
	if unpaid[0].Seq != 2 {
		t.Errorf("new schedule starts at seq %d, want 2", unpaid[0].Seq)
	}
}

func TestApplyExtraPaymentReamortizes(t *testing.T) {
	s, d, insts := loanStore(t)
	s.PayInstallment(insts[0].ID, insts[0].DueDate)

	// keepPayment=false keeps the remaining 11 payments and lowers each.
	if !s.ApplyExtraPayment(d.ID, 200_000, serialOf(2026, 1, 15), false) {
		t.Fatal("ApplyExtraPayment returned false")
	}
	var unpaid []Installment
	for _, in := range s.Installments(d.ID) {
		if !in.Paid {
			unpaid = append(unpaid, in)
		}
	}
	if len(unpaid) != 11 {
		t.Errorf("got %d unpaid installments, want 11 (re-amortize keeps the term)", len(unpaid))
	}
	pay := s.DebtByID(d.ID).PaymentAmount
	if pay >= 88_849 {
		t.Errorf("payment = %d, want lower than 88849", pay)
	}
	if got, want := unpaidPrincipal(s, d.ID), s.DebtBalance(d.ID); got != want {
		t.Errorf("unpaid principal %d != balance %d", got, want)
	}
}

func TestApplyExtraPaymentClampsAndPaysOff(t *testing.T) {
	s, d, _ := loanStore(t)

	// Paying more than the balance clamps to it and clears the loan.
	if !s.ApplyExtraPayment(d.ID, 2_000_000, 0, true) {
		t.Fatal("ApplyExtraPayment returned false")
	}
	if bal := s.DebtBalance(d.ID); bal != 0 {
		t.Errorf("balance = %d, want 0", bal)
	}
	if got := unpaidPrincipal(s, d.ID); got != 0 {
		t.Errorf("unpaid principal = %d, want 0 (schedule cleared)", got)
	}
	if bal := s.Balance("checking"); bal != -1_000_000 {
		t.Errorf("checking = %d, want -1000000 (clamped to balance)", bal)
	}
	// Guards: not a loan / nothing owed / bad amount.
	if s.ApplyExtraPayment(d.ID, 1000, 0, true) {
		t.Error("extra payment on a paid-off loan must fail")
	}
	if s.ApplyExtraPayment(d.ID, 0, 0, true) || s.ApplyExtraPayment("nope", 1000, 0, true) {
		t.Error("invalid extra payments must fail")
	}
}

func TestUpdateLoanInstallmentGuardsInterest(t *testing.T) {
	s, d, insts := loanStore(t)

	// An amount at or below the scheduled interest can't carry any principal.
	if s.UpdateInstallment(insts[0].ID, insts[0].DueDate, insts[0].Interest) {
		t.Error("amount == interest must be rejected")
	}
	// A valid edit moves the principal, keeping the interest as scheduled.
	if !s.UpdateInstallment(insts[0].ID, insts[0].DueDate, 100_000) {
		t.Fatal("UpdateInstallment returned false")
	}
	got := s.Installments(d.ID)[0]
	if got.Principal != 90_000 || got.Interest != 10_000 {
		t.Errorf("split after edit = P%d/I%d, want P90000/I10000", got.Principal, got.Interest)
	}
	// The loan's origination stays at the borrowed principal — editing one
	// installment doesn't rewrite history.
	if bal := s.DebtBalance(d.ID); bal != 1_000_000 {
		t.Errorf("balance after edit = %d, want 1000000", bal)
	}
}
