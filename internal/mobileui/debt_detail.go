package mobileui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// openDebt drills into a debt's detail page from the Debts tab.
func (m *mobileApp) openDebt(d core.Debt) {
	m.pushView(m.debtContent(d))
}

// showDebt re-renders the detail page in place after a mutation.
func (m *mobileApp) showDebt(d core.Debt) {
	m.replaceView(m.debtContent(d))
}

// rebuildDebt re-renders the detail page with fresh data (after pay/undo/edit).
func (m *mobileApp) rebuildDebt(id string) {
	if nd := m.store.DebtByID(id); nd != nil {
		m.showDebt(*nd)
	}
}

// debtContent is the detail page, matched to the debt's kind: schedule debts
// list their installments; revolving and informal debts show the balance and
// the account's recent activity.
func (m *mobileApp) debtContent(d core.Debt) fyne.CanvasObject {
	head := m.drilldownBar(d.Name)

	var col *fyne.Container
	switch {
	case d.IsRevolving():
		col = container.NewVBox(
			gap(12),
			m.revolvingHero(d),
			gap(12),
			m.debtActions(d),
			gap(14),
			sectionHeader("Activity", "", nil),
			gap(4),
			m.debtActivityCard(d),
			gap(20),
		)
	case d.HasSchedule():
		insts := m.store.Installments(d.ID)
		rows := make([]fyne.CanvasObject, 0, len(insts)*2)
		for i, in := range insts {
			if i > 0 {
				rows = append(rows, rowDivider())
			}
			rows = append(rows, m.installmentRow(d, in))
		}
		col = container.NewVBox(
			gap(12),
			m.debtHero(d),
			gap(12),
			m.debtActions(d),
			gap(14),
			sectionHeader("Schedule", "", nil),
			gap(4),
			listCard(rows...),
			gap(20),
		)
	default:
		col = container.NewVBox(
			gap(12),
			m.informalHero(d),
			gap(12),
			m.debtActions(d),
			gap(14),
			sectionHeader("Activity", "", nil),
			gap(4),
			m.debtActivityCard(d),
			gap(20),
		)
	}
	body := container.NewVScroll(insets(col, 0, 0, 14, 14))
	return container.NewBorder(head, nil, nil, nil, container.NewStack(canvas.NewRectangle(colBg), body))
}

// debtHero is the schedule-debt (BNPL / Loan) headline card.
func (m *mobileApp) debtHero(d core.Debt) fyne.CanvasObject {
	st := m.store.DebtStatus(d.ID, core.TodaySerial)

	status := strconv.Itoa(st.PaidCount) + " of " + strconv.Itoa(st.Count) + " paid"
	if st.NextDue > 0 {
		status += "  ·  next " + core.FmtSerialDate(st.NextDue)
	}
	statusT := newText(status, colInkDim, 12, false)
	if st.OverdueCount > 0 {
		statusT = newText(strconv.Itoa(st.OverdueCount)+" overdue  ·  "+status, colNeg, 12, false)
	}

	lender := d.Lender
	if lender == "" {
		lender = d.Type
	}
	amount := st.Remaining
	if d.Type == core.DebtLoan {
		amount = m.store.DebtBalance(d.ID)
		p := m.store.ProjectPayoff(d.ID, core.TodaySerial, 0)
		extra := core.FmtAPRBps(d.APRBps) + " APR  ·  " + core.FmtMoney(p.InterestRemaining) + " interest left"
		if p.PayoffDate > 0 {
			extra += "  ·  payoff " + core.FmtSerialDate(p.PayoffDate)
		}
		lender = ellipsize(lender+"  ·  "+extra, 60)
	}
	inner := container.NewVBox(
		newText("OUTSTANDING", colInkDim, 10, true),
		gap(2),
		newText(core.FmtMoney(amount), colNeg, 30, true),
		newText(ellipsize(lender, 60), colInkDim, 12, false),
		gap(12),
		miniBar(float64(st.Paid)/float64(max1(st.Total))),
		gap(8),
		statusT,
	)
	return container.NewStack(rounded(withAlpha(colNeg, 0x12), 20), insets(inner, 18, 18, 18, 18))
}

// revolvingHero is the card-debt headline: balance against the limit, minimum
// due, and the statement cycle.
func (m *mobileApp) revolvingHero(d core.Debt) fyne.CanvasObject {
	bal := m.store.DebtBalance(d.ID)
	min := m.store.MinPaymentDue(d)

	frac := 0.0
	sub := core.FmtAPRBps(d.APRBps) + " APR"
	if d.CreditLimit > 0 {
		frac = float64(bal) / float64(d.CreditLimit)
		sub += "  ·  " + core.FmtMoney(d.CreditLimit) + " limit"
	}
	status := "min due " + core.FmtMoney(min)
	if d.NextStatement > 0 {
		status += "  ·  statement " + core.FmtSerialDate(d.NextStatement)
	}
	statusT := newText(status, colInkDim, 12, false)

	inner := container.NewVBox(
		newText("BALANCE", colInkDim, 10, true),
		gap(2),
		newText(core.FmtMoney(bal), colNeg, 30, true),
		newText(ellipsize(sub, 60), colInkDim, 12, false),
		gap(12),
		miniBar(frac),
		gap(8),
		statusT,
	)
	hero := container.NewVBox(insets(inner, 0, 0, 0, 0))
	if d.NextStatement > 0 && d.NextStatement <= core.TodaySerial {
		review := widget.NewButton("Review statement", func() {
			for _, sd := range m.store.PendingStatements(core.TodaySerial) {
				if sd.DebtID == d.ID {
					m.statementReviewPage(sd, func() { m.rebuildDebt(d.ID) })
					return
				}
			}
		})
		review.Importance = widget.WarningImportance
		hero.Add(gap(10))
		hero.Add(review)
	}
	return container.NewStack(rounded(withAlpha(colNeg, 0x12), 20), insets(hero, 18, 18, 18, 18))
}

// informalHero is the IOU headline: what's owed, to whom, and until when.
func (m *mobileApp) informalHero(d core.Debt) fyne.CanvasObject {
	bal := m.store.DebtBalance(d.ID)

	status := "no due date"
	statusCol := colInkDim
	switch {
	case bal == 0:
		status = "settled"
		statusCol = colPos
	case d.DueDate > 0:
		status = "due " + core.FmtSerialDate(d.DueDate)
		if d.DueDate <= core.TodaySerial {
			statusCol = colNeg
		}
	}
	frac := 0.0
	if d.Principal > 0 {
		frac = float64(d.Principal-bal) / float64(d.Principal)
	}
	inner := container.NewVBox(
		newText("OWED", colInkDim, 10, true),
		gap(2),
		newText(core.FmtMoney(bal), colNeg, 30, true),
		newText(ellipsize(orDashText(d.Lender), 30), colInkDim, 12, false),
		gap(12),
		miniBar(frac),
		gap(8),
		newText(status, statusCol, 12, false),
	)
	return container.NewStack(rounded(withAlpha(colNeg, 0x12), 20), insets(inner, 18, 18, 18, 18))
}

// debtActions are the per-kind action buttons under the hero.
func (m *mobileApp) debtActions(d core.Debt) fyne.CanvasObject {
	edit := widget.NewButton("Edit", func() { m.editDebt(d) })
	del := widget.NewButton("Delete", func() { m.confirmDeleteDebt(d) })
	del.Importance = widget.DangerImportance

	bal := m.store.DebtBalance(d.ID)
	switch {
	case !d.HasSchedule() && bal > 0:
		pay := widget.NewButton("Record payment", func() { m.payDebtAmountPage(d) })
		pay.Importance = widget.HighImportance
		return container.NewVBox(pay, gap(8), container.NewGridWithColumns(2, edit, del))
	case d.Type == core.DebtLoan && bal > 0:
		extra := widget.NewButton("Extra payment", func() { m.extraPaymentPage(d) })
		extra.Importance = widget.HighImportance
		return container.NewVBox(extra, gap(8), container.NewGridWithColumns(2, edit, del))
	default:
		edit.Importance = widget.HighImportance
		return container.NewGridWithColumns(2, edit, del)
	}
}

// installmentRow shows one scheduled payment with a Pay / Undo action; loans
// carry their principal / interest split in the subtitle.
func (m *mobileApp) installmentRow(d core.Debt, in core.Installment) fyne.CanvasObject {
	overdue := !in.Paid && in.DueDate <= core.TodaySerial
	dotCol := colInkDim
	switch {
	case in.Paid:
		dotCol = colPos
	case overdue:
		dotCol = colNeg
	}
	dot := initialCircle("#"+strconv.Itoa(in.Seq), dotCol, 34)

	due := newText(core.FmtSerialDate(in.DueDate), colInk, 13, true)
	amtLabel := core.FmtMoney(in.Amount)
	if in.Interest > 0 {
		amtLabel += "  ·  " + core.FmtMoney(in.Interest) + " interest"
	}
	amt := newText(amtLabel, colInkDim, 11, false)
	left := flexBox(container.NewVBox(due, amt))

	var action *widget.Button
	if in.Paid {
		action = widget.NewButton("Undo", func() {
			m.store.UnpayInstallment(in.ID)
			m.rebuildDebt(d.ID)
		})
	} else {
		action = widget.NewButton("Pay", func() { m.payInstallment(d, in) })
		action.Importance = widget.HighImportance
	}

	row := container.NewBorder(nil, nil, dot, action, insets(left, 0, 0, 10, 10))
	return insets(row, 7, 7, 8, 8)
}

// debtActivityCard lists the debt account's recent transactions, newest first.
func (m *mobileApp) debtActivityCard(d core.Debt) fyne.CanvasObject {
	rows := m.store.Register(d.AcctLiability)
	if len(rows) == 0 {
		return mutedCard("No activity yet.")
	}
	var items []fyne.CanvasObject
	shown := 0
	for i := len(rows) - 1; i >= 0 && shown < 20; i-- {
		r := rows[i]
		label := orDashText(r.Txn.Payee)
		if r.Txn.Memo != "" {
			label = r.Txn.Memo
		}
		amt := -r.Amount // display sense: payments negative, charges positive
		col := colInk
		if amt < 0 {
			col = colPos
		}
		date := newText(core.FmtSerialDate(r.Txn.Date), colInk, 13, true)
		what := newText(ellipsize(label, 28), colInkDim, 11, false)
		val := newText(core.FmtMoney(amt), col, 13, true)
		val.Alignment = fyne.TextAlignTrailing
		row := container.NewBorder(nil, nil, nil, container.NewCenter(val),
			flexBox(container.NewVBox(date, what)))
		if shown > 0 {
			items = append(items, rowDivider())
		}
		items = append(items, insets(row, 7, 7, 12, 12))
		shown++
	}
	return listCard(items...)
}

func (m *mobileApp) confirmDeleteDebt(d core.Debt) {
	msg := "Delete “" + d.Name + "”? Its schedule and liability account are removed. This can't be undone."
	if d.IsRevolving() {
		msg = "Delete “" + d.Name + "”? The card account and its transactions stay — only the debt profile is removed."
	}
	m.pushPage(func(close func()) fyne.CanvasObject {
		delBtn := widget.NewButton("Delete", func() {
			close() // pop the confirm page
			m.store.DeleteDebt(d.ID)
			m.back() // pop the debt detail → Debts tab
		})
		delBtn.Importance = widget.DangerImportance
		cancel := widget.NewButton("Cancel", close)
		body := container.NewVBox(
			wrapText(msg),
			gap(18),
			delBtn,
			gap(6),
			cancel,
		)
		return m.formPage("Delete debt?", "", body, nil, close)
	})
}
