package view

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// debtTabContent is one debt's detail pane, matched to its kind: an
// installment table for schedule debts (with principal/interest columns on
// loans), a balance/utilization view for revolving credit, and a simple
// balance + history view for informal debts.
func debtTabContent(d Debt) fyne.CanvasObject {
	switch {
	case d.IsRevolving():
		return revolvingPane(d)
	case d.HasSchedule():
		return scheduleDebtPane(d)
	default:
		return informalPane(d)
	}
}

// debtPaneHeader is the shared title + Edit / Delete row.
func debtPaneHeader(d Debt, extra ...fyne.CanvasObject) fyne.CanvasObject {
	edit := secondaryButton("Edit", theme.DocumentCreateIcon(), func() { DebtForm(&d) })
	del := secondaryButton("Delete", theme.DeleteIcon(), func() {
		what := "debt “" + orDash(d.Name) + "”"
		if d.IsRevolving() {
			what += " (the account and its transactions stay)"
		}
		confirmDelete(what, func() { store.DeleteDebt(d.ID) })
	})
	sub := d.Type
	if d.Lender != "" {
		sub += " · " + d.Lender
	}
	if d.APRBps > 0 {
		sub += " · " + fmtAPRBps(d.APRBps) + " APR"
	}
	title := container.NewVBox(
		txt(d.Name, colText, 16, true),
		txt(sub, colTextDim, 11.5, false),
	)
	actions := container.NewHBox(append(extra, edit, del)...)
	return container.NewBorder(nil, nil, title, actions)
}

// scheduleDebtPane renders a BNPL or Loan debt: progress stats, the
// installment table, and (loans) the extra-payment entry point.
func scheduleDebtPane(d Debt) fyne.CanvasObject {
	st := store.DebtStatus(d.ID, todaySerial)
	isLoan := d.Type == DebtLoan

	nextLabel, nextCol := fmtSerialDate(st.NextDue), colText
	switch {
	case st.PaidCount == st.Count:
		nextLabel, nextCol = "paid off", colPositive
	case st.OverdueCount > 0:
		nextCol = colWarning
	}

	ratio := 0.0
	if st.Total > 0 {
		ratio = float64(st.Paid) / float64(st.Total)
	}

	pills := container.NewHBox(
		pillStat("Paid", itoa(st.PaidCount)+"/"+itoa(st.Count), colText), spacerW(28),
		pillStat("Remaining", fmtMoney(st.Remaining), colNegative), spacerW(28),
		pillStat("Next due", nextLabel, nextCol),
	)
	if isLoan {
		p := store.ProjectPayoff(d.ID, todaySerial, 0)
		payoff := "—"
		if p.PayoffDate > 0 {
			payoff = fmtSerialDate(p.PayoffDate)
		}
		pills = container.NewHBox(
			pillStat("Balance", fmtMoney(store.DebtBalance(d.ID)), colNegative), spacerW(28),
			pillStat("Paid", itoa(st.PaidCount)+"/"+itoa(st.Count), colText), spacerW(28),
			pillStat("Interest left", fmtMoney(p.InterestRemaining), colText), spacerW(28),
			pillStat("Payoff", payoff, nextCol),
		)
	}

	var actions []fyne.CanvasObject
	if isLoan && store.DebtBalance(d.ID) > 0 {
		actions = append(actions, secondaryButton("Extra payment", theme.MoveDownIcon(), func() {
			extraPaymentDialog(d)
		}))
	}

	header := container.NewVBox(
		debtPaneHeader(d, actions...),
		spacerH(12),
		pills,
		spacerH(12),
		progressBar(ratio, colPositive),
		spacerH(12),
		divider(),
	)

	table := container.NewVBox(installmentHeader(isLoan))
	for i, in := range store.Installments(d.ID) {
		table.Add(installmentTableRow(d, in, i, isLoan))
	}

	hint := txt("Tap an unpaid installment to edit its due date or amount.", colTextDim, 10.5, false)
	return container.New(padCell(16, 14), container.NewBorder(
		header, container.NewVBox(spacerH(8), hint), nil, nil,
		container.NewScroll(table),
	))
}

// installmentCols is the shared column layout (weights) for the installment
// header and rows; loans add principal / interest columns.
func installmentCols(loan bool) *columnsLayout {
	if loan {
		return &columnsLayout{Weights: []float32{0.5, 1.8, 1.5, 1.4, 1.4, 1.2, 1.4}, Gap: 8}
	}
	return &columnsLayout{Weights: []float32{0.5, 2.2, 2.2, 1.3, 1.4}, Gap: 10}
}

func installmentHeader(loan bool) fyne.CanvasObject {
	h := func(s string) fyne.CanvasObject { return txt(s, colTextDim, 10.5, true) }
	rh := func(s string) fyne.CanvasObject { return alignRight(txt(s, colTextDim, 10.5, true)) }
	var row fyne.CanvasObject
	if loan {
		row = container.New(installmentCols(true),
			container.NewCenter(h("#")), h("Due date"), rh("Amount"), rh("Principal"), rh("Interest"),
			container.NewCenter(h("Status")), h(""))
	} else {
		row = container.New(installmentCols(false),
			container.NewCenter(h("#")), h("Due date"), rh("Amount"),
			container.NewCenter(h("Status")), h(""))
	}
	return container.NewVBox(container.New(padCell(2, 8), row), divider())
}

// installmentTableRow renders one installment as a table line: number, due date,
// amount (split into principal / interest on loans), a status badge, and a
// Pay / Undo action. Unpaid rows are tappable to edit the due date / amount
// (schedules vary — a balloon payment, a skipped month, a fee); paid rows are
// locked until undone.
func installmentTableRow(d Debt, in Installment, idx int, loan bool) fyne.CanvasObject {
	overdue := !in.Paid && in.DueDate <= todaySerial

	state := badge("Due", colTextDim)
	switch {
	case in.Paid:
		state = badge("Paid", colPositive)
	case overdue:
		state = badge("Overdue", colWarning)
	}

	var action *widget.Button
	if in.Paid {
		action = secondaryButton("Undo", theme.ContentUndoIcon(), func() { store.UnpayInstallment(in.ID) })
	} else {
		action = primaryButton("Pay", theme.ConfirmIcon(), func() { payInstallmentDialog(d, in) })
	}

	seq := container.NewCenter(txt("#"+itoa(in.Seq), colTextDim, 12, true))
	due := txt(fmtSerialDate(in.DueDate), colText, 12.5, false)
	amt := alignRight(mono(fmtMoney(in.Amount), colText, 12.5, false))
	var cells *fyne.Container
	if loan {
		principal := alignRight(mono(fmtMoney(in.Principal), colText, 12, false))
		interest := alignRight(mono(fmtMoney(in.Interest), colTextDim, 12, false))
		cells = container.New(installmentCols(true), seq, due, amt, principal, interest,
			container.NewCenter(state), action)
	} else {
		cells = container.New(installmentCols(false), seq, due, amt, container.NewCenter(state), action)
	}

	var fill color.Color = colSurface
	if idx%2 == 1 {
		fill = colAltRow
	}
	if overdue {
		fill = withAlpha(colWarning, 0x14)
	}

	var onTap func()
	if !in.Paid {
		onTap = func() { installmentEditDialog(d, in) }
	}
	return newTappableRow(container.New(padCell(8, 8), cells), fill, onTap)
}

// revolvingPane renders a credit card / line of credit: balance against the
// limit, minimum payment, statement dates, and recent account activity.
func revolvingPane(d Debt) fyne.CanvasObject {
	bal := store.DebtBalance(d.ID)
	min := store.MinPaymentDue(d)

	utilization := 0.0
	utilLabel := ""
	if d.CreditLimit > 0 {
		utilization = float64(bal) / float64(d.CreditLimit)
		utilLabel = fmtMoney(bal) + " of " + fmtMoney(d.CreditLimit) + " limit"
	}
	utilCol := colPositive
	if utilization > 0.3 {
		utilCol = colWarning
	}
	if utilization > 0.7 {
		utilCol = colNegative
	}

	nextStmt := "—"
	if d.NextStatement > 0 {
		nextStmt = fmtSerialDate(d.NextStatement)
	}
	pills := container.NewHBox(
		pillStat("Balance", fmtMoney(bal), colNegative), spacerW(28),
		pillStat("Min payment", fmtMoney(min), colText), spacerW(28),
		pillStat("Next statement", nextStmt, colText),
	)

	var actions []fyne.CanvasObject
	if bal > 0 {
		actions = append(actions, primaryButton("Record payment", theme.ConfirmIcon(), func() {
			recordPaymentDialog(d)
		}))
	}

	header := container.NewVBox(
		debtPaneHeader(d, actions...),
		spacerH(12),
		pills,
		spacerH(12),
	)
	if utilLabel != "" {
		header.Add(progressBar(utilization, utilCol))
		header.Add(txt(utilLabel, colTextDim, 10.5, false))
		header.Add(spacerH(12))
	}
	if d.NextStatement > 0 && d.NextStatement <= todaySerial {
		review := secondaryButton("Review", theme.SearchIcon(), func() {
			for _, sd := range store.PendingStatements(todaySerial) {
				if sd.DebtID == d.ID {
					statementReviewDialog(sd)
					return
				}
			}
		})
		header.Add(alertBanner(theme.InfoIcon(), "Statement to review — confirm this month's interest", colWarning, review))
		header.Add(spacerH(12))
	}
	header.Add(divider())

	return container.New(padCell(16, 14), container.NewBorder(
		header, nil, nil, nil,
		container.NewScroll(debtActivityList(d)),
	))
}

// informalPane renders an IOU: what's still owed, when it's due, and the
// payments made so far.
func informalPane(d Debt) fyne.CanvasObject {
	bal := store.DebtBalance(d.ID)

	dueLabel, dueCol := "—", colText
	if d.DueDate > 0 {
		dueLabel = fmtSerialDate(d.DueDate)
		if d.DueDate <= todaySerial && bal > 0 {
			dueCol = colWarning
		}
	}
	settled := ""
	if bal == 0 {
		settled = "settled"
	}
	pills := container.NewHBox(
		pillStat("Owed", fmtMoney(bal), colNegative), spacerW(28),
		pillStat("Borrowed", fmtMoney(d.Principal), colText), spacerW(28),
		pillStat("Due", dueLabel, dueCol),
	)

	ratio := 0.0
	if d.Principal > 0 {
		ratio = float64(d.Principal-bal) / float64(d.Principal)
	}

	var actions []fyne.CanvasObject
	if bal > 0 {
		actions = append(actions, primaryButton("Record payment", theme.ConfirmIcon(), func() {
			recordPaymentDialog(d)
		}))
	}

	header := container.NewVBox(
		debtPaneHeader(d, actions...),
		spacerH(12),
		pills,
		spacerH(12),
		progressBar(ratio, colPositive),
	)
	if settled != "" {
		header.Add(container.NewHBox(badge("Settled", colPositive)))
	}
	header.Add(spacerH(12))
	header.Add(divider())

	return container.New(padCell(16, 14), container.NewBorder(
		header, nil, nil, nil,
		container.NewScroll(debtActivityList(d)),
	))
}

// debtActivityList shows the debt account's recent transactions, newest first —
// purchases, payments, and interest — straight from the register.
func debtActivityList(d Debt) fyne.CanvasObject {
	rows := store.Register(d.AcctLiability)
	box := container.NewVBox(txt("Activity", colTextDim, 10.5, true), spacerH(4))
	if len(rows) == 0 {
		box.Add(txt("No activity yet.", colTextDim, 11.5, false))
		return box
	}
	shown := 0
	for i := len(rows) - 1; i >= 0 && shown < 20; i-- {
		r := rows[i]
		label := orDash(r.Txn.Payee)
		if r.Txn.Memo != "" {
			label = r.Txn.Memo
		}
		// In display sense a payment shrinks the balance (negative), a purchase
		// or interest charge grows it.
		amt := -r.Amount
		col := colText
		if amt < 0 {
			col = colPositive // paying it down
		}
		line := container.NewBorder(nil, nil,
			container.NewHBox(
				mono(fmtSerialDate(r.Txn.Date), colTextDim, 11.5, false), spacerW(10),
				txt(label, colText, 12, false),
			),
			mono(fmtMoney(amt), col, 12, false),
		)
		var fill color.Color = colSurface
		if shown%2 == 1 {
			fill = colAltRow
		}
		box.Add(newTappableRow(container.New(padCell(6, 8), line), fill, nil))
		shown++
	}
	return box
}

// ---- dialogs ----

// installmentEditDialog edits one unpaid installment's due date and amount.
func installmentEditDialog(d Debt, in Installment) {
	if win == nil {
		return
	}
	due := newDateEntry(in.DueDate)
	amt := newAmountEntry()
	amt.SetText(fmtMoneyInput(in.Amount))

	form := container.NewVBox(
		txt(d.Name+" · installment "+itoa(in.Seq), colTextDim, 11, true),
		spacerH(8),
		txt("Due date", colTextDim, 10.5, true), due,
		spacerH(8),
		txt("Amount", colTextDim, 10.5, true), amt,
	)
	if in.Interest > 0 {
		form.Add(spacerH(6))
		form.Add(txt("Includes "+fmtMoney(in.Interest)+" scheduled interest — the amount must exceed it.",
			colTextDim, 10.5, false))
	}
	dialogWithSave("Edit installment", form, func() {
		ds := parseDateSerial(due.Text)
		a := parseAmount(amt.Text)
		if ds == 0 || a <= 0 {
			return
		}
		store.UpdateInstallment(in.ID, ds, a)
	}).Show()
}

// payInstallmentDialog reviews how to record an installment payment — the same
// treatment as posting a recurring transaction. You either post a new payment now,
// or link the installment to a transaction already in your books (paid manually,
// imported, or a slightly different amount). Linking re-points that transaction
// onto the debt's liability so the outstanding balance stays correct rather than
// double-posting. Either way no payment is posted twice.
func payInstallmentDialog(d Debt, in Installment) {
	if win == nil {
		store.PayInstallment(in.ID, todaySerial)
		return
	}
	cands, auto := store.RecurringCandidates(d.AcctMoney, in.Amount, todaySerial)
	// Default to creating a new payment; pre-select a confident match so an
	// already-recorded installment links in one click.
	sel := candidateSelect(postNewTodayOption, cands, d.AcctMoney, auto)

	sub := "Post a new payment now, or link one already in your books."
	if in.Interest > 0 {
		sub += " " + fmtMoney(in.Interest) + " of it is interest."
	}
	body := container.NewVBox(
		txt(orDash(d.Name)+" · installment "+itoa(in.Seq)+"   ·   "+fmtMoney(in.Amount), colText, 14, true),
		txt(sub, colTextDim, 11.5, false),
		spacerH(10),
		txt("Record this payment as:", colTextDim, 11.5, false),
		sel.widget,
	)

	var dlg *dialog.CustomDialog
	cancel := secondaryButton("Cancel", nil, func() { dlg.Hide() })
	confirm := primaryButton("Confirm", theme.ConfirmIcon(), func() {
		postNew, picked := sel.isPostNew(), sel.picked()
		dlg.Hide()
		label := orDash(d.Name) + " · installment " + itoa(in.Seq)
		if postNew || picked == nil {
			if store.PayInstallment(in.ID, todaySerial) {
				showInfo("Paid", label+" posted.")
			}
			return
		}
		if store.LinkInstallment(in.ID, picked.ID) {
			showInfo("Linked", label+" linked to an existing transaction.")
		}
	})

	content := container.NewVBox(body, spacerH(16),
		container.NewHBox(layout.NewSpacer(), cancel, confirm))
	dlg = dialog.NewCustomWithoutButtons("Pay installment", content, win)
	dlg.Resize(fyne.NewSize(520, 300))
	dlg.Show()
}

// recordPaymentDialog records an arbitrary-amount payment against a revolving
// or informal debt — defaulted to the minimum due (cards) or the full balance
// (IOUs).
func recordPaymentDialog(d Debt) {
	bal := store.DebtBalance(d.ID)
	def := bal
	hint := "Owed: " + fmtMoney(bal)
	if d.IsRevolving() {
		def = store.MinPaymentDue(d)
		hint = "Balance: " + fmtMoney(bal) + " · minimum: " + fmtMoney(def)
	}

	amt := newAmountEntry()
	amt.Validator = amountValidator
	amt.SetText(fmtMoneyInput(def))
	date := newDateEntry(todaySerial)

	form := container.NewVBox(
		txt(d.Name, colTextDim, 11, true),
		txt(hint, colTextDim, 10.5, false),
		spacerH(8),
		txt("Amount", colTextDim, 10.5, true), amt,
		spacerH(8),
		txt("Date", colTextDim, 10.5, true), date,
	)
	dialogWithSave("Record payment", form, func() {
		a := parseAmount(amt.Text)
		ds := parseDateSerial(date.Text)
		if a <= 0 || ds == 0 {
			return
		}
		store.PayDebtAmount(d.ID, a, ds)
	}).Show()
}

// extraPaymentDialog posts an extra principal payment on a loan, with a live
// preview of what it buys: pick whether the regular payment stays (term
// shortens — the default, and the most interest saved) or the term stays (each
// remaining payment drops).
func extraPaymentDialog(d Debt) {
	bal := store.DebtBalance(d.ID)
	cur := store.ProjectPayoff(d.ID, todaySerial, 0)

	amt := newAmountEntry()
	amt.Validator = amountValidator
	date := newDateEntry(todaySerial)
	const shorten = "Shorten the term (keep the payment)"
	const lower = "Lower the payment (keep the term)"
	mode := widget.NewRadioGroup([]string{shorten, lower}, nil)
	mode.SetSelected(shorten)

	preview := widget.NewLabel("")
	preview.Wrapping = fyne.TextWrapWord
	update := func() {
		a := parseAmount(amt.Text)
		if a <= 0 || a >= bal {
			if a >= bal && bal > 0 {
				preview.SetText("Pays the loan off entirely.")
			} else {
				preview.SetText("")
			}
			return
		}
		// Dry-run the new schedule to show what the extra payment buys.
		newBal := bal - a
		unpaid := 0
		firstDue := todaySerial
		for _, in := range store.Installments(d.ID) {
			if !in.Paid {
				if unpaid == 0 {
					firstDue = in.DueDate
				}
				unpaid++
			}
		}
		payment := d.PaymentAmount
		if mode.Selected == lower && unpaid > 0 {
			payment = amortizedPayment(newBal, d.APRBps, unpaid, d.Freq)
		}
		insts, ok := generateLoanSchedule(newBal, d.APRBps, payment, firstDue, d.Freq)
		if !ok {
			preview.SetText("")
			return
		}
		last := insts[len(insts)-1].DueDate
		saved := cur.InterestRemaining - scheduleInterest(insts)
		msg := "Payoff " + fmtSerialDate(last) + " · saves " + fmtMoney(saved) + " interest"
		if mode.Selected == lower {
			msg = "New payment " + fmtMoney(payment) + " · " + msg
		}
		preview.SetText(msg)
	}
	amt.OnChanged = func(string) { update() }
	mode.OnChanged = func(string) { update() }

	form := container.NewVBox(
		txt(d.Name+" · balance "+fmtMoney(bal), colTextDim, 11, true),
		spacerH(8),
		txt("Extra amount", colTextDim, 10.5, true), amt,
		spacerH(8),
		txt("Date", colTextDim, 10.5, true), date,
		spacerH(8),
		mode,
		preview,
	)
	dialogWithSave("Extra payment", form, func() {
		a := parseAmount(amt.Text)
		ds := parseDateSerial(date.Text)
		if a <= 0 || ds == 0 {
			return
		}
		store.ApplyExtraPayment(d.ID, a, ds, mode.Selected == shorten)
	}).Show()
}

// statementReviewDialog confirms one revolving statement: the proposed interest
// estimate is editable (the real statement is ground truth), posting charges it
// to the card, and "No interest" advances without posting.
func statementReviewDialog(sd StatementDue) {
	if win == nil {
		return
	}
	amt := newAmountEntry()
	amt.SetText(fmtMoneyInput(sd.Interest))

	body := container.NewVBox(
		txt(sd.Name+" · statement "+fmtSerialDate(sd.Date), colText, 14, true),
		txt("Balance "+fmtMoney(sd.Balance)+" · minimum payment "+fmtMoney(sd.MinPayment), colTextDim, 11.5, false),
		spacerH(10),
		txt("Interest charged this month (from your statement):", colTextDim, 11.5, false),
		amt,
	)

	var dlg *dialog.CustomDialog
	cancel := secondaryButton("Cancel", nil, func() { dlg.Hide() })
	skip := secondaryButton("No interest", theme.ContentClearIcon(), func() {
		dlg.Hide()
		if store.SkipStatement(sd.DebtID) {
			showInfo("Reviewed", sd.Name+" statement closed with no interest.")
		}
	})
	post := primaryButton("Post interest", theme.ConfirmIcon(), func() {
		a := parseAmount(amt.Text)
		if a < 0 {
			return
		}
		dlg.Hide()
		if store.PostStatement(sd.DebtID, a, sd.Date) {
			showInfo("Posted", sd.Name+" interest recorded.")
		}
	})

	content := container.NewVBox(body, spacerH(16),
		container.NewHBox(layout.NewSpacer(), cancel, skip, post))
	dlg = dialog.NewCustomWithoutButtons("Review statement", content, win)
	dlg.Resize(fyne.NewSize(520, 280))
	dlg.Show()
}
