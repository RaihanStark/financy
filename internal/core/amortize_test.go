package core

import "testing"

func TestAmortizedPaymentGolden(t *testing.T) {
	// The classic annuity check: 1,000,000 minor units at 12.00% APR over 12
	// monthly payments → 1% per period → payment 88,849 (half-up from 88,848.79).
	if pay := AmortizedPayment(1_000_000, 1200, 12, "Monthly"); pay != 88_849 {
		t.Errorf("payment = %d, want 88849", pay)
	}
	// Zero APR degenerates to a near-equal split via ceiling division.
	if pay := AmortizedPayment(10_000, 0, 3, "Monthly"); pay != 3334 {
		t.Errorf("zero-APR payment = %d, want 3334", pay)
	}
	// Degenerate inputs.
	if AmortizedPayment(0, 1200, 12, "Monthly") != 0 || AmortizedPayment(1000, 1200, 0, "Monthly") != 0 {
		t.Error("degenerate inputs should yield 0")
	}
}

func TestPeriodInterestRounding(t *testing.T) {
	// 100,000 at 19.99% APR monthly: 100000*1999/120000 = 1665.83… → 1666.
	if got := PeriodInterest(100_000, 1999, 12); got != 1666 {
		t.Errorf("interest = %d, want 1666 (half-up)", got)
	}
	// Exact half rounds up: 30,000 at 12% monthly = 300 exactly; make a .5 case:
	// 25 at 12% monthly = 0.25 → 0; 50 at 12% monthly = 0.5 → 1.
	if got := PeriodInterest(50, 1200, 12); got != 1 {
		t.Errorf("half case = %d, want 1 (round half up)", got)
	}
	if got := PeriodInterest(25, 1200, 12); got != 0 {
		t.Errorf("quarter case = %d, want 0", got)
	}
	if PeriodInterest(0, 1200, 12) != 0 || PeriodInterest(1000, 0, 12) != 0 {
		t.Error("zero balance/rate must yield 0")
	}
	// Weekly halves the per-period rate vs biweekly on the same balance.
	if w, b := PeriodInterest(1_000_000, 1200, 52), PeriodInterest(1_000_000, 1200, 26); w >= b {
		t.Errorf("weekly interest %d should be less than biweekly %d", w, b)
	}
}

func TestGenerateLoanScheduleGolden(t *testing.T) {
	first := serialOf(2026, 1, 1)
	insts, ok := GenerateLoanSchedule(1_000_000, 1200, 88_849, first, "Monthly")
	if !ok {
		t.Fatal("schedule not generated")
	}
	if len(insts) != 12 {
		t.Fatalf("got %d installments, want 12", len(insts))
	}
	// First row: interest = 1% of 1,000,000 = 10,000; principal = 78,849.
	if insts[0].Interest != 10_000 || insts[0].Principal != 78_849 {
		t.Errorf("row 1 split = P%d/I%d, want P78849/I10000", insts[0].Principal, insts[0].Interest)
	}
	sumP, sumI, sumA := 0, 0, 0
	for i, in := range insts {
		if in.Seq != i+1 {
			t.Errorf("row %d seq = %d", i, in.Seq)
		}
		if in.Amount != in.Principal+in.Interest {
			t.Errorf("row %d amount %d != principal %d + interest %d", i, in.Amount, in.Principal, in.Interest)
		}
		sumP += in.Principal
		sumI += in.Interest
		sumA += in.Amount
	}
	// Principal column sums back exactly; the final payment absorbs the residual.
	if sumP != 1_000_000 {
		t.Errorf("Σprincipal = %d, want exactly 1000000", sumP)
	}
	if sumI != ScheduleInterestTotal(insts) || sumA != sumP+sumI {
		t.Error("schedule totals inconsistent")
	}
	if last := insts[11]; last.Amount > 88_849 {
		t.Errorf("final payment %d exceeds the regular payment", last.Amount)
	}
	// Interest declines as the balance draws down.
	for i := 1; i < len(insts); i++ {
		if insts[i].Interest > insts[i-1].Interest {
			t.Errorf("interest rose at row %d: %d > %d", i, insts[i].Interest, insts[i-1].Interest)
		}
	}
	// Dates advance monthly.
	if insts[1].DueDate != serialOf(2026, 2, 1) || insts[11].DueDate != serialOf(2026, 12, 1) {
		t.Error("due dates didn't advance monthly")
	}
}

func TestGenerateLoanScheduleZeroAPR(t *testing.T) {
	first := serialOf(2026, 1, 1)
	insts, ok := GenerateLoanSchedule(10_000, 0, 3334, first, "Monthly")
	if !ok {
		t.Fatal("zero-APR schedule not generated")
	}
	if len(insts) != 3 {
		t.Fatalf("got %d installments, want 3", len(insts))
	}
	// 3334 + 3334 + 3332: all principal, last absorbs the residual.
	if insts[2].Amount != 3332 || insts[2].Interest != 0 {
		t.Errorf("final row = %+v, want amount 3332 interest 0", insts[2])
	}
}

func TestGenerateLoanScheduleNonAmortizing(t *testing.T) {
	// Payment below the first period's interest (10,000) can never amortize.
	if _, ok := GenerateLoanSchedule(1_000_000, 1200, 9_000, TodaySerial, "Monthly"); ok {
		t.Error("payment below interest must not generate a schedule")
	}
	if _, ok := GenerateLoanSchedule(1_000_000, 1200, 10_000, TodaySerial, "Monthly"); ok {
		t.Error("payment equal to interest must not generate a schedule")
	}
	if _, ok := GenerateLoanSchedule(0, 1200, 1000, TodaySerial, "Monthly"); ok {
		t.Error("zero principal must not generate a schedule")
	}
}

func TestLoanTermForInverse(t *testing.T) {
	// Solving payment from term, then term from that payment, round-trips.
	pay := AmortizedPayment(1_000_000, 1200, 12, "Monthly")
	n, ok := LoanTermFor(1_000_000, 1200, pay, "Monthly")
	if !ok || n != 12 {
		t.Errorf("term for solved payment = %d ok=%v, want 12", n, ok)
	}
	if _, ok := LoanTermFor(1_000_000, 1200, 10_000, "Monthly"); ok {
		t.Error("non-amortizing payment must report ok=false")
	}
	// A generous payment shortens the term.
	if n, ok := LoanTermFor(1_000_000, 1200, 200_000, "Monthly"); !ok || n != 6 {
		t.Errorf("term at 200k/mo = %d ok=%v, want 6", n, ok)
	}
}

func TestParseFmtAPRBps(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"19.99", 1999}, {"19,99%", 1999}, {"12", 1200}, {"12%", 1200},
		{"0.5", 50}, {"7.125", 712}, {"", 0}, {"abc", 0},
	}
	for _, c := range cases {
		if got := ParseAPRBps(c.in); got != c.want {
			t.Errorf("ParseAPRBps(%q) = %d, want %d", c.in, got, c.want)
		}
	}
	fmtCases := []struct {
		in   int
		want string
	}{
		{1999, "19.99%"}, {1200, "12%"}, {1250, "12.5%"}, {50, "0.5%"}, {712, "7.12%"}, {0, "0%"},
	}
	for _, c := range fmtCases {
		if got := FmtAPRBps(c.in); got != c.want {
			t.Errorf("FmtAPRBps(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
