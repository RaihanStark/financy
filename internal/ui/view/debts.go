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

// debtsTab is the ID of the debt whose tab is selected. It's preserved across the
// screen rebuilds that follow every pay / undo / edit, so you stay on the debt you
// were looking at instead of being thrown back to the first tab.
var debtsTab string

// ScreenDebts shows tracked debts (BNPL etc.) as a tabbed master/detail: a tab per
// debt down the side, each opening that debt's installment table with inline
// Pay / Undo, edit, and delete controls.
func ScreenDebts() fyne.CanvasObject {
	bar := appBar("Debts", "Buy-now-pay-later and other installment debts",
		primaryButton("Add Debt", theme.ContentAddIcon(), func() { DebtForm(nil) }),
	)

	ds := store.Debts()
	if len(ds) == 0 {
		body := panel(emptyState("No debts yet. Add a BNPL purchase and its monthly installments…"))
		return container.NewBorder(bar, nil, nil, nil, container.NewPadded(body))
	}

	tabs := container.NewAppTabs()
	tabs.SetTabLocation(container.TabLocationLeading)
	selected := 0
	for i, d := range ds {
		title := d.Name
		if st := store.DebtStatus(d.ID, todaySerial); st.OverdueCount > 0 {
			title = "● " + title // amber dot hint: something's overdue
		}
		tabs.Append(container.NewTabItem(title, debtTabContent(d)))
		if d.ID == debtsTab {
			selected = i
		}
	}
	tabs.SelectIndex(selected)
	debtsTab = ds[selected].ID
	tabs.OnSelected = func(*container.TabItem) {
		if i := tabs.SelectedIndex(); i >= 0 && i < len(ds) {
			debtsTab = ds[i].ID
		}
	}

	head := container.NewPadded(container.NewVBox(debtSummary(), debtDueBanner()))
	return container.NewBorder(
		container.NewVBox(bar, head), nil, nil, nil,
		container.NewPadded(tabs),
	)
}

// debtSummary is the header line: everything still owed, plus a timing breakdown
// of unpaid installments — what's already due, what's left this month, and what
// lands next month.
func debtSummary() fyne.CanvasObject {
	b := store.DebtBuckets(todaySerial)
	if b.Outstanding == 0 {
		return spacerH(0)
	}
	dueCol := colTextDim
	if b.Due > 0 {
		dueCol = colWarning // already owed — flag it
	}
	return container.NewHBox(
		debtStat("Outstanding", fmtMoney(b.Outstanding), colNegative), spacerW(32),
		debtStat("Due now", fmtMoney(b.Due), dueCol), spacerW(32),
		debtStat("This month", fmtMoney(b.ThisMonth), colText), spacerW(32),
		debtStat("Next month", fmtMoney(b.NextMonth), colTextDim),
	)
}

// debtStat is one inline "label  value" pair for the Debts header.
func debtStat(label, value string, valueCol color.Color) fyne.CanvasObject {
	return container.NewHBox(
		txt(label+"  ", colTextDim, 12, true),
		txt(value, valueCol, 15, true),
	)
}

// debtDueBanner nudges when installments are due.
func debtDueBanner() fyne.CanvasObject {
	n := store.DueDebtCount(todaySerial)
	if n == 0 {
		return spacerH(0)
	}
	msg := itoa(n) + " installment" + plural(n, "", "s") + " due"
	return container.NewHBox(spacerH(0), txt("●  ", colWarning, 13, true), txt(msg, colText, 12.5, true))
}

// debtTabContent is one debt's detail pane: a stats header with a progress bar,
// edit / delete actions, and the installment table.
func debtTabContent(d Debt) fyne.CanvasObject {
	st := store.DebtStatus(d.ID, todaySerial)

	edit := secondaryButton("Edit", theme.DocumentCreateIcon(), func() { DebtForm(&d) })
	del := secondaryButton("Delete", theme.DeleteIcon(), func() {
		confirmDelete("debt “"+orDash(d.Name)+"”", func() { store.DeleteDebt(d.ID) })
	})
	title := container.NewVBox(
		txt(d.Name, colText, 16, true),
		txt(d.Type+" · "+orDash(d.Lender), colTextDim, 11.5, false),
	)

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

	header := container.NewVBox(
		container.NewBorder(nil, nil, title, container.NewHBox(edit, del)),
		spacerH(12),
		container.NewHBox(
			pillStat("Paid", itoa(st.PaidCount)+"/"+itoa(st.Count), colText), spacerW(28),
			pillStat("Remaining", fmtMoney(st.Remaining), colNegative), spacerW(28),
			pillStat("Total", fmtMoney(st.Total), colText), spacerW(28),
			pillStat("Next due", nextLabel, nextCol),
		),
		spacerH(12),
		progressBar(ratio, colPositive),
		spacerH(12),
		divider(),
	)

	table := container.NewVBox(installmentHeader())
	for i, in := range store.Installments(d.ID) {
		table.Add(installmentTableRow(d, in, i))
	}

	hint := txt("Tap an unpaid installment to edit its due date or amount.", colTextDim, 10.5, false)
	return container.New(padCell(16, 14), container.NewBorder(
		header, container.NewVBox(spacerH(8), hint), nil, nil,
		container.NewScroll(table),
	))
}

// installmentCols is the shared column layout (weights) for the installment
// header and rows, so the fields and their labels line up.
func installmentCols() *columnsLayout {
	return &columnsLayout{Weights: []float32{0.5, 2.2, 2.2, 1.3, 1.4}, Gap: 10}
}

func installmentHeader() fyne.CanvasObject {
	h := func(s string) fyne.CanvasObject { return txt(s, colTextDim, 10.5, true) }
	row := container.New(installmentCols(),
		container.NewCenter(h("#")), h("Due date"), alignRight(txt("Amount", colTextDim, 10.5, true)),
		container.NewCenter(h("Status")), h(""))
	return container.NewVBox(container.New(padCell(2, 8), row), divider())
}

// installmentTableRow renders one installment as a table line: number, due date,
// amount, a status badge, and a Pay / Undo action. Unpaid rows are tappable to
// edit the due date / amount (schedules vary — a balloon payment, a skipped month,
// a fee); paid rows are locked until undone.
func installmentTableRow(d Debt, in Installment, idx int) fyne.CanvasObject {
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
	cells := container.New(installmentCols(), seq, due, amt, container.NewCenter(state), action)

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

	body := container.NewVBox(
		txt(orDash(d.Name)+" · installment "+itoa(in.Seq)+"   ·   "+fmtMoney(in.Amount), colText, 14, true),
		txt("Post a new payment now, or link one already in your books.", colTextDim, 11.5, false),
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

	moneyNames := groupedLabels(store.MoneyAccounts())
	acctMoney := widget.NewSelect(moneyNames, nil)
	guardGroupHeaders(acctMoney)

	if existing != nil {
		// Edit: metadata only.
		kind.SetSelected(existing.Type)
		name.SetText(existing.Name)
		lender.SetText(existing.Lender)
		note.SetText(existing.Note)
		acctMoney.SetSelected(labelOf(existing.AcctMoney))

		items := []*widget.FormItem{
			widget.NewFormItem("Name", name),
			widget.NewFormItem("Lender", lender),
			widget.NewFormItem("Pay from", acctMoney),
			widget.NewFormItem("Note", note),
		}
		showForm("Edit Debt", items, func() {
			if name.Text == "" || acctMoney.Selected == "" {
				return
			}
			store.UpdateDebt(existing.ID, Debt{
				Name: name.Text, Type: kind.Selected, Lender: lender.Text,
				AcctMoney: idOf(acctMoney.Selected),
				Note:      note.Text,
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
	purchase := newDateEntry(todaySerial)
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
		widget.NewFormItem("Total amount", total),
		widget.NewFormItem("Installments", count),
		widget.NewFormItem("Per installment", perLabel),
		widget.NewFormItem("Purchase date", purchase),
		widget.NewFormItem("First due", first),
		widget.NewFormItem("Frequency", freq),
		widget.NewFormItem("Note", note),
	}
	showForm("Add Debt", items, func() {
		amt := parseAmount(total.Text)
		n, _ := strconv.Atoi(count.Text)
		due := parseDateSerial(first.Text)
		bought := parseDateSerial(purchase.Text)
		if name.Text == "" || acctMoney.Selected == "" ||
			amt <= 0 || n <= 0 || due == 0 || bought == 0 {
			return
		}
		d := Debt{
			Name: name.Text, Type: kind.Selected, Lender: lender.Text,
			AcctMoney:    idOf(acctMoney.Selected),
			PurchaseDate: bought,
			Note:         note.Text,
		}
		store.AddDebt(d, generateInstallments(amt, n, due, freq.Selected))
	})
}
