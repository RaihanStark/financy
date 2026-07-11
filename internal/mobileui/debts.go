package mobileui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/raihanstark/financy/internal/core"
)

// debtsScreen lists each debt with its balance and progress, under a summary
// header (total owed, weighted APR, debt-free date). Statements awaiting
// review surface as a banner card; a Strategies row opens the snowball vs
// avalanche comparison. Tapping a debt opens its detail page.
func (m *mobileApp) debtsScreen() fyne.CanvasObject {
	s := m.store
	debts := s.Debts()

	o := s.DebtOverview(core.TodaySerial)
	sub := "Outstanding " + core.FmtMoney(o.TotalOwed)
	if o.WeightedAPRBps > 0 {
		sub += " · " + core.FmtAPRBps(o.WeightedAPRBps) + " avg APR"
	}
	if o.DebtFreeDate > 0 {
		sub += " · debt-free " + core.FmtSerialDate(o.DebtFreeDate)
	}
	header := insets(screenHeader("Debts", sub), 12, 8, 16, 16)

	if len(debts) == 0 {
		return container.NewBorder(header, nil, nil, nil,
			container.NewVScroll(insets(mutedCard("No debts tracked — tap Add to track one."), 4, 0, 16, 16)))
	}

	col := container.NewVBox(gap(4))
	if pending := s.PendingStatements(core.TodaySerial); len(pending) > 0 {
		first := pending[0]
		msg := strconv.Itoa(len(pending)) + " card statement"
		if len(pending) > 1 {
			msg += "s"
		}
		msg += " to review"
		banner := container.NewStack(rounded(withAlpha(colNeg, 0x14), 16),
			insets(newText("●  "+msg, colNeg, 13, true), 14, 14, 16, 16))
		col.Add(newTappableCard(banner, func() { m.statementReviewPage(first, m.render) }))
		col.Add(gap(10))
	}
	for _, d := range debts {
		d := d
		col.Add(newTappableCard(m.debtCard(d), func() { m.openDebt(d) }))
		col.Add(gap(10))
	}
	if o.TotalOwed > 0 {
		strat := container.NewStack(rounded(colCard, 16),
			insets(container.NewVBox(
				newText("Payoff strategies", colInk, 14, true),
				newText("Snowball vs avalanche — where should extra money go?", colInkFaint, 11, false),
			), 14, 14, 16, 16))
		col.Add(newTappableCard(strat, func() { m.strategyPage() }))
		col.Add(gap(10))
	}
	col.Add(gap(110))
	return container.NewBorder(header, nil, nil, nil, container.NewVScroll(insets(col, 0, 0, 16, 16)))
}

func (m *mobileApp) debtCard(d core.Debt) fyne.CanvasObject {
	switch {
	case d.IsRevolving():
		return m.revolvingCard(d)
	case d.HasSchedule():
		return m.scheduleCard(d)
	default:
		return m.informalCard(d)
	}
}

// debtCardShell is the shared card frame: name + subtitle on the left, the
// balance on the right, with a progress bar and status line below.
func debtCardShell(name0, sub0 string, amount int, frac float64, status string, alarmed bool) fyne.CanvasObject {
	name := newText(name0, colInk, 15, true)
	sub := newText(sub0, colInkFaint, 11, false)
	amt := newText(core.FmtMoney(amount), colInk, 16, true)
	amt.Alignment = fyne.TextAlignTrailing

	statusText := newText(status, colInkDim, 11, false)
	if alarmed {
		statusText.Color = colNeg
	}

	top := container.NewBorder(nil, nil, container.NewVBox(name, sub), container.NewCenter(amt))
	inner := container.NewVBox(top, gap(10), miniBar(frac), gap(6), statusText)
	return container.NewStack(rounded(colCard, 16), insets(inner, 16, 16, 16, 16))
}

func (m *mobileApp) scheduleCard(d core.Debt) fyne.CanvasObject {
	st := m.store.DebtStatus(d.ID, core.TodaySerial)

	sub := strings.TrimSpace(d.Lender)
	if sub == "" {
		sub = d.Type
	}
	if d.APRBps > 0 {
		sub += " · " + core.FmtAPRBps(d.APRBps)
	}
	status := strconv.Itoa(st.PaidCount) + " of " + strconv.Itoa(st.Count) + " paid"
	if st.OverdueCount > 0 {
		status += "  ·  " + strconv.Itoa(st.OverdueCount) + " overdue"
	}
	amount := st.Remaining
	if d.Type == core.DebtLoan {
		amount = m.store.DebtBalance(d.ID)
	}
	return debtCardShell(d.Name, sub, amount,
		float64(st.Paid)/float64(max1(st.Total)), status, st.OverdueCount > 0)
}

func (m *mobileApp) revolvingCard(d core.Debt) fyne.CanvasObject {
	bal := m.store.DebtBalance(d.ID)

	sub := strings.TrimSpace(d.Lender)
	if sub == "" {
		sub = d.Type
	}
	if d.APRBps > 0 {
		sub += " · " + core.FmtAPRBps(d.APRBps)
	}
	frac := 0.0
	status := "min due " + core.FmtMoney(m.store.MinPaymentDue(d))
	if d.CreditLimit > 0 {
		frac = float64(bal) / float64(d.CreditLimit)
		status += "  ·  " + core.FmtMoney(d.CreditLimit) + " limit"
	}
	review := d.NextStatement > 0 && d.NextStatement <= core.TodaySerial
	if review {
		status = "statement to review  ·  " + status
	}
	return debtCardShell(d.Name, sub, bal, frac, status, review)
}

func (m *mobileApp) informalCard(d core.Debt) fyne.CanvasObject {
	bal := m.store.DebtBalance(d.ID)

	sub := strings.TrimSpace(d.Lender)
	if sub == "" {
		sub = d.Type
	}
	frac := 0.0
	if d.Principal > 0 {
		frac = float64(d.Principal-bal) / float64(d.Principal)
	}
	status := "no due date"
	overdue := false
	switch {
	case bal == 0:
		status = "settled"
	case d.DueDate > 0:
		status = "due " + core.FmtSerialDate(d.DueDate)
		overdue = d.DueDate <= core.TodaySerial
	}
	return debtCardShell(d.Name, sub, bal, frac, status, overdue)
}
