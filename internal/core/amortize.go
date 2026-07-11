package core

import "math"

// Amortization: the interest math behind Loan debts, kept deliberately small
// and deterministic. All money stays in integer minor units; rates are annual
// basis points (1999 = 19.99%). The only float64 is the transient annuity
// payment solve, rounded half-up exactly once — every schedule walk and
// projection uses the same integer kernel (PeriodInterest), so schedules,
// projections and strategy simulations can never drift from each other.

// maxLoanPeriods caps every schedule walk and payoff simulation. No real loan
// runs 100 years; hitting the cap means the payment doesn't amortize the debt.
const maxLoanPeriods = 1200

// PeriodsPerYear maps a payment frequency to periods per year. Unknown or empty
// frequencies fall back to Monthly, matching nextDate.
func PeriodsPerYear(freq string) int {
	switch freq {
	case "Weekly":
		return 52
	case "Biweekly":
		return 26
	case "Quarterly":
		return 4
	case "Yearly":
		return 1
	default: // Monthly
		return 12
	}
}

// PeriodInterest is the shared interest kernel: one period's interest on a
// balance at aprBps, rounded half-up in pure integer math.
func PeriodInterest(balance, aprBps, periodsPerYear int) int {
	if balance <= 0 || aprBps <= 0 || periodsPerYear <= 0 {
		return 0
	}
	d := int64(10000) * int64(periodsPerYear)
	return int((int64(balance)*int64(aprBps) + d/2) / d)
}

// AmortizedPayment solves the regular payment that amortizes principal over n
// periods at aprBps (the standard annuity formula), rounded UP to the next
// minor unit — rounding down would leave a residue that spills into an n+1th
// installment; rounding up keeps the term at n with a slightly smaller final
// payment, which is what lenders do. A zero rate degenerates to a near-equal
// split (ceiling division, same reasoning).
func AmortizedPayment(principal, aprBps, n int, freq string) int {
	if principal <= 0 || n <= 0 {
		return 0
	}
	if aprBps <= 0 {
		return (principal + n - 1) / n
	}
	r := float64(aprBps) / (10000 * float64(PeriodsPerYear(freq)))
	pay := float64(principal) * r / (1 - math.Pow(1+r, -float64(n)))
	return int(math.Ceil(pay))
}

// LoanTermFor solves how many payments of the given size it takes to pay the
// loan off. ok is false when the payment can never amortize the debt (it
// doesn't cover even the first period's interest, or the term would exceed
// maxLoanPeriods).
func LoanTermFor(principal, aprBps, payment int, freq string) (n int, ok bool) {
	insts, ok := GenerateLoanSchedule(principal, aprBps, payment, TodaySerial, freq)
	if !ok {
		return 0, false
	}
	return len(insts), true
}

// GenerateLoanSchedule walks the balance period by period: each installment's
// interest is PeriodInterest on the running balance, the rest of the payment
// is principal. The final installment pays exactly the remaining balance plus
// its interest, absorbing all rounding residue — so the schedule's principal
// column always sums back to exactly `principal`. ok is false when the payment
// never amortizes the debt.
func GenerateLoanSchedule(principal, aprBps, payment, firstDue int, freq string) ([]Installment, bool) {
	if principal <= 0 || payment <= 0 {
		return nil, false
	}
	ppy := PeriodsPerYear(freq)
	var out []Installment
	bal := principal
	due := firstDue
	for seq := 1; bal > 0; seq++ {
		if seq > maxLoanPeriods {
			return nil, false
		}
		interest := PeriodInterest(bal, aprBps, ppy)
		princ := payment - interest
		if princ <= 0 {
			return nil, false
		}
		if princ > bal {
			princ = bal
		}
		out = append(out, Installment{
			Seq:       seq,
			DueDate:   due,
			Amount:    princ + interest,
			Principal: princ,
			Interest:  interest,
		})
		bal -= princ
		due = nextDate(due, freq)
	}
	return out, true
}

// ScheduleInterestTotal sums the interest column of a schedule.
func ScheduleInterestTotal(insts []Installment) int {
	sum := 0
	for _, in := range insts {
		sum += in.Interest
	}
	return sum
}
