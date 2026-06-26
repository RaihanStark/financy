package view

import (
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ScreenDebts lists tracked debts (BNPL etc.) with their installment progress,
// and lets you add/edit/delete them and pay installments off.
func ScreenDebts() fyne.CanvasObject {
	bar := appBar("Debts", "Buy-now-pay-later and other installment debts",
		primaryButton("Add Debt", theme.ContentAddIcon(), func() { DebtForm(nil) }),
	)

	ds := store.Debts()
	if len(ds) == 0 {
		body := panel(emptyState("No debts yet. Add a BNPL purchase and its monthly installments…"))
		return container.NewBorder(bar, nil, nil, nil, container.NewPadded(body))
	}

	rows := make([]fyne.CanvasObject, 0, len(ds))
	for _, d := range ds {
		rows = append(rows, debtCard(d))
	}
	body := panel(container.NewVBox(rows...))
	return container.NewBorder(bar, nil, nil, nil, container.NewPadded(container.NewVBox(
		debtSummary(), debtDueBanner(), body,
	)))
}

// debtSummary is a one-line total of everything still owed.
func debtSummary() fyne.CanvasObject {
	out := store.TotalDebtOutstanding()
	if out == 0 {
		return spacerH(0)
	}
	return container.NewBorder(nil, spacerH(8), nil, nil,
		container.NewHBox(
			txt("Outstanding  ", colTextDim, 12, true),
			txt(fmtMoney(out), colNegative, 15, true),
		))
}

// debtDueBanner nudges when installments are due.
func debtDueBanner() fyne.CanvasObject {
	n := store.DueDebtCount(todaySerial)
	if n == 0 {
		return spacerH(0)
	}
	msg := itoa(n) + " installment" + plural(n, "", "s") + " due"
	return container.NewBorder(nil, spacerH(8), nil, nil,
		container.NewHBox(txt("●  ", colWarning, 13, true), txt(msg, colText, 12.5, true)))
}

func debtCard(d Debt) fyne.CanvasObject {
	st := store.DebtStatus(d.ID, todaySerial)

	menuBtn := widget.NewButtonWithIcon("", theme.MoreVerticalIcon(), nil)
	menuBtn.Importance = widget.LowImportance
	menuBtn.OnTapped = func() {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn)
		pos.Y += menuBtn.Size().Height
		showContextMenu(pos, debtMenu(d)...)
	}

	sub := d.Type + " · " + orDash(d.Lender) + " · paid " + itoa(st.PaidCount) + "/" + itoa(st.Count)
	left := container.NewVBox(txt(d.Name, colText, 13.5, true), txt(sub, colTextDim, 10.5, false))

	// Progress state line: overdue (amber), all paid (positive), or next due.
	state, stateCol := "next · "+fmtSerialDate(st.NextDue), colTextDim
	switch {
	case st.PaidCount == st.Count:
		state, stateCol = "paid off", colPositive
	case st.OverdueCount > 0:
		state, stateCol = "● "+itoa(st.OverdueCount)+" due · "+fmtSerialDate(st.NextDue), colWarning
	}
	right := container.NewVBox(
		alignRight(txt(fmtMoney(st.Remaining)+" left", colNegative, 14, true)),
		alignRight(txt(state, stateCol, 10, st.OverdueCount > 0)),
	)

	ratio := 0.0
	if st.Total > 0 {
		ratio = float64(st.Paid) / float64(st.Total)
	}
	head := container.NewBorder(nil, nil, left, container.NewHBox(right, menuBtn))
	inner := container.NewVBox(head, spacerH(6), progressBar(ratio, colPositive))

	var fill color.Color = colSurface
	if st.OverdueCount > 0 {
		fill = withAlpha(colWarning, 0x14)
	}
	row := newTappableRow(container.New(padCell(10, 12), inner), fill, func() { debtDetail(d.ID) })
	row.SetBordered()
	row.SetOnSecondary(func(pos fyne.Position) { showContextMenu(pos, debtMenu(d)...) })
	return row
}

func debtMenu(d Debt) []*fyne.MenuItem {
	open := fyne.NewMenuItem("Installments…", func() { debtDetail(d.ID) })
	open.Icon = theme.ListIcon()
	edit := fyne.NewMenuItem("Edit…", func() { DebtForm(&d) })
	edit.Icon = theme.DocumentCreateIcon()
	del := fyne.NewMenuItem("Delete", func() {
		confirmDelete("debt “"+orDash(d.Name)+"”", func() { store.DeleteDebt(d.ID) })
	})
	del.Icon = theme.DeleteIcon()
	return []*fyne.MenuItem{open, edit, fyne.NewMenuItemSeparator(), del}
}

// debtDetail shows a debt's installment schedule with pay / undo controls. It
// reopens itself after each change so the schedule reflects the new state.
func debtDetail(debtID string) {
	if win == nil {
		return
	}
	d := store.DebtByID(debtID)
	if d == nil {
		return
	}
	st := store.DebtStatus(debtID, todaySerial)

	header := container.NewVBox(
		txt(d.Name, colText, 16, true),
		txt(d.Type+" · "+orDash(d.Lender), colTextDim, 11.5, false),
		spacerH(8),
		container.NewHBox(
			pillStat("Paid", itoa(st.PaidCount)+"/"+itoa(st.Count), colText), spacerW(24),
			pillStat("Remaining", fmtMoney(st.Remaining), colNegative), spacerW(24),
			pillStat("Total", fmtMoney(st.Total), colText),
		),
		spacerH(6), divider(),
	)

	var d2 *dialog.CustomDialog
	reload := func() { d2.Hide(); debtDetail(debtID) }
	listBox := container.NewVBox(installmentHeader())
	for _, in := range store.Installments(debtID) {
		listBox.Add(installmentRow(in, reload))
	}

	close := secondaryButton("Close", nil, func() { d2.Hide() })
	hint := txt("Edit a due date or amount inline — changes save as you type.", colTextDim, 10.5, false)
	body := container.NewBorder(header,
		container.NewVBox(spacerH(6), hint, spacerH(4), container.NewHBox(layout.NewSpacer(), close)),
		nil, nil,
		container.NewScroll(listBox),
	)
	d2 = dialog.NewCustomWithoutButtons("Installments", body, win)
	d2.Resize(fyne.NewSize(620, 580))
	d2.Show()
}

// installmentCols is the shared column layout (weights) for the installment
// header and rows, so the fields and their labels line up.
func installmentCols() *columnsLayout {
	return &columnsLayout{Weights: []float32{0.5, 2.2, 2.2, 1.3, 1.5}, Gap: 10}
}

func installmentHeader() fyne.CanvasObject {
	h := func(s string) fyne.CanvasObject { return txt(s, colTextDim, 10.5, true) }
	row := container.New(installmentCols(),
		container.NewCenter(h("#")), h("Due date"), h("Amount"), h("Status"), h(""))
	return container.NewVBox(container.New(padCell(2, 8), row), divider())
}

// installmentRow renders one installment as an inline-editable line: the due date
// and amount are live fields (saved as you type) for unpaid installments, with a
// Pay / Undo action. reload rebuilds the dialog after a pay/undo.
func installmentRow(in Installment, reload func()) fyne.CanvasObject {
	overdue := !in.Paid && in.DueDate <= todaySerial

	var due, amt *widget.Entry
	if in.Paid {
		// Posted installments are locked — Undo first to change one. Plain disabled
		// fields (no calendar button) keep the row clean.
		due = widget.NewEntry()
		due.SetText(fmtSerialDate(in.DueDate))
		due.Disable()
		amt = widget.NewEntry()
		amt.SetText(fmtMoneyInput(in.Amount))
		amt.Disable()
	} else {
		due = newDateEntry(in.DueDate)
		amt = newAmountEntry()
		amt.SetText(fmtMoneyInput(in.Amount))

		// Live-save edits, skipping no-op writes so each keystroke doesn't thrash the DB.
		lastD, lastA := in.DueDate, in.Amount
		commit := func() {
			ds := parseDateSerial(due.Text)
			a := parseAmount(amt.Text)
			if ds == 0 || a <= 0 || (ds == lastD && a == lastA) {
				return
			}
			lastD, lastA = ds, a
			store.UpdateInstallment(in.ID, ds, a)
		}
		due.OnChanged = func(string) { commit() }
		prevAmt := amt.OnChanged // keep the live thousands-grouping behaviour
		amt.OnChanged = func(s string) {
			if prevAmt != nil {
				prevAmt(s)
			}
			commit()
		}
	}

	state := badge("Due", colTextDim)
	switch {
	case in.Paid:
		state = badge("Paid", colPositive)
	case overdue:
		state = badge("Overdue", colWarning)
	}

	var action *widget.Button
	if in.Paid {
		action = secondaryButton("Undo", theme.ContentUndoIcon(), func() {
			store.UnpayInstallment(in.ID)
			reload()
		})
	} else {
		action = primaryButton("Pay", theme.ConfirmIcon(), func() {
			store.PayInstallment(in.ID)
			reload()
		})
	}

	seq := container.NewCenter(txt("#"+itoa(in.Seq), colTextDim, 12, true))
	row := container.New(installmentCols(), seq, due, amt, container.NewCenter(state), action)
	return container.New(padCell(5, 8), row)
}

// DebtForm is the add/edit dialog. On add it captures the full plan (total,
// number of installments, first due date, frequency) and generates the schedule.
// On edit only the metadata is changed — the schedule is left as-is.
func DebtForm(existing *Debt) {
	name := widget.NewEntry()
	name.SetPlaceHolder("e.g. iPhone 15")
	lender := widget.NewEntry()
	lender.SetPlaceHolder("e.g. Shopee PayLater")
	kind := widget.NewSelect(debtTypes(), nil)
	note := widget.NewEntry()

	moneyNames := namesOf(store.MoneyAccounts())
	expenseNames := namesOf(store.ExpenseAccounts())
	acctMoney := widget.NewSelect(moneyNames, nil)
	acctExpense := widget.NewSelect(expenseNames, nil)

	if existing != nil {
		// Edit: metadata only.
		kind.SetSelected(existing.Type)
		name.SetText(existing.Name)
		lender.SetText(existing.Lender)
		note.SetText(existing.Note)
		acctMoney.SetSelected(nameOf(existing.AcctMoney))
		acctExpense.SetSelected(nameOf(existing.AcctExpense))

		items := []*widget.FormItem{
			widget.NewFormItem("Name", name),
			widget.NewFormItem("Lender", lender),
			widget.NewFormItem("Pay from", acctMoney),
			widget.NewFormItem("Category", acctExpense),
			widget.NewFormItem("Note", note),
		}
		showForm("Edit Debt", items, func() {
			if name.Text == "" || acctMoney.Selected == "" || acctExpense.Selected == "" {
				return
			}
			store.UpdateDebt(existing.ID, Debt{
				Name: name.Text, Type: kind.Selected, Lender: lender.Text,
				AcctMoney: idOf(acctMoney.Selected), AcctExpense: idOf(acctExpense.Selected),
				Note: note.Text,
			})
		})
		return
	}

	// Add: capture the plan and generate the schedule.
	kind.SetSelected(debtTypes()[0])
	total := newAmountEntry()
	total.Validator = amountValidator
	count := widget.NewEntry()
	count.SetText("12")
	first := newDateEntry(todaySerial)
	freq := widget.NewSelect(frequencies(), nil)
	freq.SetSelected("Monthly")

	perLabel := widget.NewLabel("")
	updatePer := func() {
		amt := parseAmount(total.Text)
		n, _ := strconv.Atoi(count.Text)
		if amt > 0 && n > 0 {
			perLabel.SetText("≈ " + fmtMoney(amt/n) + " × " + itoa(n))
		} else {
			perLabel.SetText("—")
		}
	}
	total.OnChanged = func(string) { updatePer() }
	count.OnChanged = func(string) { updatePer() }
	updatePer()

	items := []*widget.FormItem{
		widget.NewFormItem("Type", kind),
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Lender", lender),
		widget.NewFormItem("Pay from", acctMoney),
		widget.NewFormItem("Category", acctExpense),
		widget.NewFormItem("Total amount", total),
		widget.NewFormItem("Installments", count),
		widget.NewFormItem("Per installment", perLabel),
		widget.NewFormItem("First due", first),
		widget.NewFormItem("Frequency", freq),
		widget.NewFormItem("Note", note),
	}
	showForm("Add Debt", items, func() {
		amt := parseAmount(total.Text)
		n, _ := strconv.Atoi(count.Text)
		due := parseDateSerial(first.Text)
		if name.Text == "" || acctMoney.Selected == "" || acctExpense.Selected == "" ||
			amt <= 0 || n <= 0 || due == 0 {
			return
		}
		d := Debt{
			Name: name.Text, Type: kind.Selected, Lender: lender.Text,
			AcctMoney: idOf(acctMoney.Selected), AcctExpense: idOf(acctExpense.Selected),
			Note: note.Text,
		}
		store.AddDebt(d, generateInstallments(amt, n, due, freq.Selected))
	})
}
