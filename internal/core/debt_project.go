package core

import (
	"sort"
	"time"
)

// Payoff projections and strategies. Everything here is pure simulation over
// the debts' current state — nothing is posted. All simulations share the
// PeriodInterest kernel with the schedule generator, so a projection can never
// disagree with the schedule it's projecting, and every loop is capped at
// maxLoanPeriods with Feasible=false instead of running away when a payment
// can't beat the interest.

// PayoffProjection is one debt's road to zero under its current payment plan.
type PayoffProjection struct {
	Months            int // periods until paid off (0 = already clear)
	PayoffDate        int // serial of the final payment; 0 when infeasible
	InterestRemaining int // interest still to be paid
	Feasible          bool
}

// ProjectPayoff projects a single debt's payoff. extraPerPeriod is an optional
// additional principal payment applied every period on top of the plan:
//
//   - BNPL / Loan follow their unpaid schedule; with an extra, the loan is
//     re-simulated at payment+extra (interest-free BNPL just finishes sooner).
//   - Revolving assumes you keep paying today's minimum payment (plus the
//     extra) every month — the classic minimum-payment projection.
//   - Informal has no plan of its own: its due date (you'll pay by then) or an
//     extra per period makes it projectable; otherwise it isn't, unless it's
//     already settled.
func (s *Store) ProjectPayoff(debtID string, asOf, extraPerPeriod int) PayoffProjection {
	d := s.DebtByID(debtID)
	if d == nil {
		return PayoffProjection{}
	}
	bal := s.DebtBalance(debtID)
	if bal <= 0 {
		return PayoffProjection{Feasible: true}
	}
	if d.Type == DebtInformal && extraPerPeriod <= 0 {
		if d.DueDate > 0 {
			return PayoffProjection{Months: 1, PayoffDate: d.DueDate, Feasible: true}
		}
		return PayoffProjection{}
	}

	if d.HasSchedule() && extraPerPeriod <= 0 {
		// The schedule already is the projection.
		var p PayoffProjection
		for _, in := range s.Installments(d.ID) {
			if in.Paid {
				continue
			}
			p.Months++
			p.InterestRemaining += in.Interest
			if in.DueDate > p.PayoffDate {
				p.PayoffDate = in.DueDate
			}
		}
		p.Feasible = true
		return p
	}

	freq := d.Freq
	payment := 0
	switch {
	case d.IsRevolving():
		freq = "Monthly"
		payment = s.MinPaymentDue(*d)
	case d.HasSchedule():
		payment = d.PaymentAmount
		if payment == 0 { // BNPL stores no payment; its installments are near-equal
			for _, in := range s.Installments(d.ID) {
				if !in.Paid {
					payment = in.Amount
					break
				}
			}
		}
	}
	return simulatePayoff(bal, d.APRBps, payment+extraPerPeriod, asOf, freq)
}

// simulatePayoff walks a balance down at a constant per-period payment.
func simulatePayoff(bal, aprBps, payment, asOf int, freq string) PayoffProjection {
	if payment <= 0 {
		return PayoffProjection{}
	}
	ppy := PeriodsPerYear(freq)
	var p PayoffProjection
	date := asOf
	for bal > 0 {
		if p.Months >= maxLoanPeriods {
			return PayoffProjection{InterestRemaining: p.InterestRemaining}
		}
		interest := PeriodInterest(bal, aprBps, ppy)
		if payment <= interest {
			return PayoffProjection{}
		}
		p.Months++
		p.InterestRemaining += interest
		bal -= payment - interest
		date = nextDate(date, freq)
	}
	p.PayoffDate = date
	p.Feasible = true
	return p
}

// ---- payoff strategies (snowball vs avalanche) ----

// DebtPayoff is one debt's outcome under a strategy: when it's cleared and
// what interest it cost along the way.
type DebtPayoff struct {
	DebtID       string
	Name         string
	PayoffDate   int
	InterestPaid int
}

// StrategyResult is a full payoff plan under one strategy.
type StrategyResult struct {
	Strategy      string // "Snowball" | "Avalanche"
	DebtFreeDate  int    // serial when the last debt clears; 0 when infeasible
	TotalInterest int
	Order         []DebtPayoff // in payoff order
	Feasible      bool
}

// Strategy names.
const (
	StrategySnowball  = "Snowball"
	StrategyAvalanche = "Avalanche"
)

// CompareStrategies simulates paying every open debt down under both classic
// strategies, given an extra amount available each month on top of all
// required payments. The total monthly outlay is held constant: each debt's
// required payment (loan payment, BNPL installment, revolving minimum as of
// today) keeps being paid, and once a debt clears, its payment rolls into the
// pool — aimed at the smallest balance (snowball: quick wins) or the highest
// APR (avalanche: least interest).
func (s *Store) CompareStrategies(asOf, extraMonthly int) (snowball, avalanche StrategyResult) {
	return s.runStrategy(StrategySnowball, asOf, extraMonthly),
		s.runStrategy(StrategyAvalanche, asOf, extraMonthly)
}

// simDebt is a debt normalized to a monthly cadence for strategy simulation.
type simDebt struct {
	id, name     string
	bal, aprBps  int
	required     int // required monthly payment, fixed as of the start
	interestPaid int
	payoffMonth  int
}

func (s *Store) runStrategy(strategy string, asOf, extraMonthly int) StrategyResult {
	res := StrategyResult{Strategy: strategy}
	var debts []*simDebt
	for _, d := range s.debts {
		bal := s.DebtBalance(d.ID)
		if bal <= 0 {
			continue
		}
		debts = append(debts, &simDebt{
			id: d.ID, name: d.Name, bal: bal, aprBps: d.APRBps,
			required: s.monthlyRequired(d),
		})
	}
	if len(debts) == 0 {
		res.Feasible = true
		res.DebtFreeDate = asOf
		return res
	}

	month := 0
	for {
		open := 0
		for _, d := range debts {
			if d.bal > 0 {
				open++
			}
		}
		if open == 0 {
			break
		}
		month++
		if month > maxLoanPeriods {
			return StrategyResult{Strategy: strategy, TotalInterest: res.TotalInterest}
		}

		// Interest accrues on every open balance.
		for _, d := range debts {
			if d.bal <= 0 {
				continue
			}
			i := PeriodInterest(d.bal, d.aprBps, 12)
			d.bal += i
			d.interestPaid += i
			res.TotalInterest += i
		}
		// Every debt gets its required payment; what a closed (or overpaid) debt
		// doesn't need joins the extra pool.
		pool := extraMonthly
		for _, d := range debts {
			pay := d.required
			if pay > d.bal {
				pay = d.bal
			}
			d.bal -= pay
			pool += d.required - pay
			if d.bal == 0 && d.payoffMonth == 0 {
				d.payoffMonth = month
			}
		}
		// The pool attacks the target debt; rollover cascades to the next target.
		for pool > 0 {
			t := pickTarget(debts, strategy)
			if t == nil {
				break
			}
			pay := pool
			if pay > t.bal {
				pay = t.bal
			}
			t.bal -= pay
			pool -= pay
			if t.bal == 0 {
				t.payoffMonth = month
			}
		}
	}

	sort.SliceStable(debts, func(i, j int) bool { return debts[i].payoffMonth < debts[j].payoffMonth })
	for _, d := range debts {
		date := serialPlusMonths(asOf, d.payoffMonth)
		res.Order = append(res.Order, DebtPayoff{
			DebtID: d.id, Name: d.name, PayoffDate: date, InterestPaid: d.interestPaid,
		})
		if date > res.DebtFreeDate {
			res.DebtFreeDate = date
		}
	}
	res.Feasible = true
	return res
}

// pickTarget selects the strategy's next victim among open debts.
func pickTarget(debts []*simDebt, strategy string) *simDebt {
	var best *simDebt
	for _, d := range debts {
		if d.bal <= 0 {
			continue
		}
		if best == nil {
			best = d
			continue
		}
		if strategy == StrategyAvalanche {
			if d.aprBps > best.aprBps || (d.aprBps == best.aprBps && d.bal < best.bal) {
				best = d
			}
		} else if d.bal < best.bal {
			best = d
		}
	}
	return best
}

// monthlyRequired normalizes a debt's required payment to a monthly figure.
func (s *Store) monthlyRequired(d Debt) int {
	switch {
	case d.IsRevolving():
		return s.MinPaymentDue(d)
	case d.HasSchedule():
		pay := d.PaymentAmount
		if pay == 0 { // BNPL: the near-equal installment
			for _, in := range s.Installments(d.ID) {
				if !in.Paid {
					pay = in.Amount
					break
				}
			}
		}
		ppy := PeriodsPerYear(d.Freq)
		if ppy == 12 {
			return pay
		}
		return int((int64(pay)*int64(ppy) + 6) / 12)
	default: // Informal: no required payment — only the extra pool clears it
		return 0
	}
}

func serialPlusMonths(serial, months int) int {
	return TimeToSerial(addMonths(SerialToTime(serial), months))
}

// ---- overview dashboard ----

// DebtOverviewRow is one debt's line on the overview dashboard.
type DebtOverviewRow struct {
	DebtID            string
	Name              string
	Type              string
	Balance           int
	APRBps            int
	PayoffDate        int
	InterestRemaining int
	Feasible          bool
	// Progress is the repaid fraction of what was borrowed — or, for revolving,
	// the credit utilization (balance / limit).
	Progress float64
}

// MonthBalance is one point of the total-owed history.
type MonthBalance struct {
	Month   string // "YYYY-MM"
	Balance int    // total owed at month end
}

// DebtOverview is the aggregate picture across every debt.
type DebtOverview struct {
	TotalOwed      int
	WeightedAPRBps int // balance-weighted average APR
	DebtFreeDate   int // latest projected payoff; 0 when any debt is unprojectable
	Rows           []DebtOverviewRow
	History        []MonthBalance // last 12 months of total owed, oldest first
}

// DebtOverview rolls every debt into the dashboard aggregate.
func (s *Store) DebtOverview(asOf int) DebtOverview {
	var o DebtOverview
	var weighted int64
	allFeasible := true
	for _, d := range s.debts {
		bal := s.DebtBalance(d.ID)
		p := s.ProjectPayoff(d.ID, asOf, 0)
		row := DebtOverviewRow{
			DebtID: d.ID, Name: d.Name, Type: d.Type, Balance: bal, APRBps: d.APRBps,
			PayoffDate: p.PayoffDate, InterestRemaining: p.InterestRemaining, Feasible: p.Feasible,
		}
		switch {
		case d.IsRevolving():
			if d.CreditLimit > 0 {
				row.Progress = float64(bal) / float64(d.CreditLimit)
			}
		default:
			borrowed := d.Principal
			if d.Type == DebtBNPL {
				borrowed = 0
				for _, in := range s.Installments(d.ID) {
					borrowed += in.Principal
				}
			}
			if borrowed > 0 {
				row.Progress = float64(borrowed-bal) / float64(borrowed)
			}
		}
		o.Rows = append(o.Rows, row)
		if bal <= 0 {
			continue
		}
		o.TotalOwed += bal
		weighted += int64(bal) * int64(d.APRBps)
		if !p.Feasible {
			allFeasible = false
		}
		if p.PayoffDate > o.DebtFreeDate {
			o.DebtFreeDate = p.PayoffDate
		}
	}
	if o.TotalOwed > 0 {
		o.WeightedAPRBps = int((weighted + int64(o.TotalOwed)/2) / int64(o.TotalOwed))
	}
	if !allFeasible {
		o.DebtFreeDate = 0
	}
	o.History = s.debtHistory(asOf, 12)
	return o
}

// debtHistory is the total owed across all debts at each of the last n month
// ends (this month is measured as of asOf).
func (s *Store) debtHistory(asOf, n int) []MonthBalance {
	liabs := map[string]bool{}
	for _, d := range s.debts {
		if d.AcctLiability != "" {
			liabs[d.AcctLiability] = true
		}
	}
	out := make([]MonthBalance, 0, n)
	for i := n - 1; i >= 0; i-- {
		end := asOf
		if i > 0 {
			t := SerialToTime(asOf)
			firstOfMonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			end = TimeToSerial(addMonths(firstOfMonth, -i+1)) - 1 // last day of that month
		}
		sum := 0
		for _, t := range s.txns {
			if t.Date > end {
				continue
			}
			for _, p := range t.Posts {
				if liabs[p.AccountID] {
					sum -= p.Amount
				}
			}
		}
		if sum < 0 {
			sum = 0
		}
		out = append(out, MonthBalance{Month: FmtSerialMonth(end), Balance: sum})
	}
	return out
}
