package core

import "testing"

func TestProjectPayoffScheduleDebts(t *testing.T) {
	s, d, insts := loanStore(t)
	asOf := serialOf(2026, 1, 1)

	// With no extra, the loan's projection IS its schedule.
	p := s.ProjectPayoff(d.ID, asOf, 0)
	if !p.Feasible || p.Months != 12 {
		t.Errorf("projection = %+v, want feasible over 12 months", p)
	}
	if p.PayoffDate != insts[11].DueDate {
		t.Errorf("payoff = %s, want last due %s", FmtSerialDate(p.PayoffDate), FmtSerialDate(insts[11].DueDate))
	}
	if p.InterestRemaining != ScheduleInterestTotal(insts) {
		t.Errorf("interest remaining = %d, want %d", p.InterestRemaining, ScheduleInterestTotal(insts))
	}

	// Paying two installments shrinks the projection accordingly.
	s.PayInstallment(insts[0].ID, insts[0].DueDate)
	s.PayInstallment(insts[1].ID, insts[1].DueDate)
	p = s.ProjectPayoff(d.ID, asOf, 0)
	if p.Months != 10 || p.InterestRemaining >= ScheduleInterestTotal(insts) {
		t.Errorf("projection after 2 payments = %+v, want 10 months and less interest", p)
	}

	// An extra per period pays it off sooner and cheaper.
	pe := s.ProjectPayoff(d.ID, asOf, 100_000)
	if !pe.Feasible || pe.Months >= p.Months || pe.InterestRemaining >= p.InterestRemaining {
		t.Errorf("with extra = %+v, want faster and cheaper than %+v", pe, p)
	}

	// A settled debt projects as already clear.
	s2 := debtStore()
	s2.AddDebt(Debt{Name: "IOU", Type: DebtInformal, AcctMoney: "checking", Principal: 1000}, nil)
	iou := s2.Debts()[0].ID
	s2.PayDebtAmount(iou, 1000, 0)
	if p := s2.ProjectPayoff(iou, asOf, 0); !p.Feasible || p.Months != 0 {
		t.Errorf("settled projection = %+v, want feasible/0 months", p)
	}
}

func TestProjectPayoffRevolvingMinimumTrap(t *testing.T) {
	s, d := revolvingStore(t)
	asOf := serialOf(2026, 1, 10)

	// Minimum payments only: feasible (the floor beats the interest) but long
	// and expensive.
	p := s.ProjectPayoff(d.ID, asOf, 0)
	if !p.Feasible {
		t.Fatalf("minimum-payment projection infeasible: %+v", p)
	}
	if p.Months <= 12 {
		t.Errorf("minimum-payment payoff in %d months — too fast to be right", p.Months)
	}
	// An extra 200k/month shortens it dramatically.
	pe := s.ProjectPayoff(d.ID, asOf, 200_000)
	if !pe.Feasible || pe.Months >= p.Months || pe.InterestRemaining >= p.InterestRemaining {
		t.Errorf("with extra = %+v, want far better than %+v", pe, p)
	}

	// A minimum below the monthly interest can never pay off.
	s2 := debtStore()
	s2.AddDebt(Debt{
		Name: "Trap", Type: DebtRevolving, AcctMoney: "checking",
		APRBps: 3600, StatementDay: 5, MinPayBps: 100, MinPayFloor: 0, Principal: 1_000_000,
	}, nil) // 3% monthly interest vs 1% minimum payment
	trap := s2.Debts()[0]
	if p := s2.ProjectPayoff(trap.ID, asOf, 0); p.Feasible {
		t.Errorf("trap projection = %+v, want infeasible", p)
	}
}

func TestProjectPayoffInformal(t *testing.T) {
	s := debtStore()
	s.AddDebt(Debt{Name: "IOU", Type: DebtInformal, AcctMoney: "checking", Principal: 300_000}, nil)
	d := s.Debts()[0]
	asOf := serialOf(2026, 1, 1)

	// No payment plan, no due date, no extra → unprojectable.
	if p := s.ProjectPayoff(d.ID, asOf, 0); p.Feasible {
		t.Errorf("informal with no plan = %+v, want infeasible", p)
	}
	// A due date IS the plan: you'll have paid it by then.
	due := serialOf(2026, 4, 1)
	s2 := debtStore()
	s2.AddDebt(Debt{Name: "IOU2", Type: DebtInformal, AcctMoney: "checking",
		Principal: 300_000, DueDate: due}, nil)
	if p := s2.ProjectPayoff(s2.Debts()[0].ID, asOf, 0); !p.Feasible || p.PayoffDate != due {
		t.Errorf("informal with due date = %+v, want feasible payoff at %d", p, due)
	}
	// With 100k/month it clears in 3.
	p := s.ProjectPayoff(d.ID, asOf, 100_000)
	if !p.Feasible || p.Months != 3 || p.InterestRemaining != 0 {
		t.Errorf("informal with extra = %+v, want 3 interest-free months", p)
	}
}

func TestCompareStrategiesOrderingAndRollover(t *testing.T) {
	s := debtStore()
	asOf := serialOf(2026, 1, 1)
	// Two contrasting debts:
	//  - "Small" — tiny balance, low APR (snowball's first target)
	//  - "Costly" — big balance, high APR (avalanche's first target)
	small, ok := GenerateLoanSchedule(300_000, 600, 26_000, serialOf(2026, 2, 1), "Monthly")
	if !ok {
		t.Fatal("small schedule")
	}
	s.AddDebt(Debt{Name: "Small", Type: DebtLoan, AcctMoney: "checking",
		APRBps: 600, Freq: "Monthly", PaymentAmount: 26_000}, small)
	costly, ok := GenerateLoanSchedule(2_000_000, 2400, 95_000, serialOf(2026, 2, 1), "Monthly")
	if !ok {
		t.Fatal("costly schedule")
	}
	s.AddDebt(Debt{Name: "Costly", Type: DebtLoan, AcctMoney: "checking",
		APRBps: 2400, Freq: "Monthly", PaymentAmount: 95_000}, costly)

	snow, aval := s.CompareStrategies(asOf, 50_000)
	if !snow.Feasible || !aval.Feasible {
		t.Fatalf("strategies infeasible: %+v / %+v", snow, aval)
	}
	// Each strategy accelerates its own target: snowball clears Small sooner
	// than avalanche does, avalanche clears Costly sooner than snowball does.
	payoff := func(r StrategyResult, name string) int {
		for _, p := range r.Order {
			if p.Name == name {
				return p.PayoffDate
			}
		}
		t.Fatalf("%s missing from %s order", name, r.Strategy)
		return 0
	}
	if payoff(snow, "Small") > payoff(aval, "Small") {
		t.Errorf("snowball pays Small at %d, later than avalanche %d", payoff(snow, "Small"), payoff(aval, "Small"))
	}
	if payoff(aval, "Costly") > payoff(snow, "Costly") {
		t.Errorf("avalanche pays Costly at %d, later than snowball %d", payoff(aval, "Costly"), payoff(snow, "Costly"))
	}
	// Avalanche never pays more interest than snowball.
	if aval.TotalInterest > snow.TotalInterest {
		t.Errorf("avalanche interest %d > snowball %d", aval.TotalInterest, snow.TotalInterest)
	}
	// The freed payment rolls over: with the extra pool both finish within the
	// simulation and every debt has a payoff date.
	for _, r := range append(snow.Order, aval.Order...) {
		if r.PayoffDate == 0 {
			t.Errorf("missing payoff date for %s", r.Name)
		}
	}
	if snow.DebtFreeDate == 0 || aval.DebtFreeDate == 0 {
		t.Error("missing debt-free date")
	}

	// More extra money means an earlier (or equal) debt-free date, less interest.
	snow2, _ := s.CompareStrategies(asOf, 500_000)
	if snow2.DebtFreeDate > snow.DebtFreeDate || snow2.TotalInterest > snow.TotalInterest {
		t.Errorf("bigger extra didn't help: %+v vs %+v", snow2, snow)
	}
}

func TestCompareStrategiesEdgeCases(t *testing.T) {
	asOf := serialOf(2026, 1, 1)

	// No debts → trivially feasible, debt-free today.
	s := debtStore()
	snow, aval := s.CompareStrategies(asOf, 0)
	if !snow.Feasible || !aval.Feasible || snow.DebtFreeDate != asOf {
		t.Errorf("empty comparison = %+v / %+v", snow, aval)
	}

	// An informal debt with no required payment and no extra never terminates →
	// reported infeasible, not an endless loop.
	s.AddDebt(Debt{Name: "IOU", Type: DebtInformal, AcctMoney: "checking", Principal: 100_000}, nil)
	snow, _ = s.CompareStrategies(asOf, 0)
	if snow.Feasible {
		t.Errorf("unpayable plan = %+v, want infeasible", snow)
	}
	// With an extra pool it resolves.
	snow, _ = s.CompareStrategies(asOf, 50_000)
	if !snow.Feasible || len(snow.Order) != 1 {
		t.Errorf("with extra = %+v, want feasible single payoff", snow)
	}
}

func TestDebtOverview(t *testing.T) {
	s, card := revolvingStore(t) // 1,000,000 @ 19.99%
	asOf := serialOf(2026, 1, 10)
	loan, ok := GenerateLoanSchedule(1_000_000, 1200, 88_849, serialOf(2026, 2, 1), "Monthly")
	if !ok {
		t.Fatal("loan schedule")
	}
	s.AddDebt(Debt{Name: "Car", Type: DebtLoan, AcctMoney: "checking",
		APRBps: 1200, Freq: "Monthly", PaymentAmount: 88_849,
		PurchaseDate: serialOf(2026, 1, 5)}, loan)

	o := s.DebtOverview(asOf)
	if o.TotalOwed != 2_000_000 {
		t.Errorf("TotalOwed = %d, want 2000000", o.TotalOwed)
	}
	// Balance-weighted APR of equal balances at 19.99% and 12%: 1599.5 → 1600.
	if o.WeightedAPRBps != 1600 {
		t.Errorf("WeightedAPRBps = %d, want 1600", o.WeightedAPRBps)
	}
	if len(o.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(o.Rows))
	}
	for _, r := range o.Rows {
		if !r.Feasible || r.PayoffDate == 0 {
			t.Errorf("row %s unprojectable: %+v", r.Name, r)
		}
	}
	// Revolving progress is utilization: 1M / 5M = 0.2.
	for _, r := range o.Rows {
		if r.DebtID == card.ID && (r.Progress < 0.19 || r.Progress > 0.21) {
			t.Errorf("card utilization = %f, want 0.2", r.Progress)
		}
	}
	if o.DebtFreeDate == 0 {
		t.Error("missing DebtFreeDate")
	}
	// History spans 12 months ending now, and the current month carries the debt.
	if len(o.History) != 12 {
		t.Fatalf("history = %d points, want 12", len(o.History))
	}
	last := o.History[11]
	if last.Balance != 2_000_000 {
		t.Errorf("current history point = %+v, want balance 2000000", last)
	}
	if first := o.History[0]; first.Balance != 0 {
		t.Errorf("a year ago = %+v, want 0 (debts created this month)", first)
	}

	// Loan progress grows once payments land.
	insts := s.Installments(s.Debts()[1].ID)
	s.PayInstallment(insts[0].ID, insts[0].DueDate)
	o = s.DebtOverview(asOf)
	for _, r := range o.Rows {
		if r.Type == DebtLoan && (r.Progress <= 0 || r.Progress >= 1) {
			t.Errorf("loan progress = %f, want within (0,1)", r.Progress)
		}
	}
}
