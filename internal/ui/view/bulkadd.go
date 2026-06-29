package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ieRow is one line of the "Income & Expenses" table: its own date and money
// account, an amount, a category (whose account type decides income vs expense),
// and a payee.
type ieRow struct {
	date   *widget.Entry
	money  *widget.Select
	amount *widget.Entry
	cat    *widget.Select
	payee  *completionEntry
	box    fyne.CanvasObject
}

// trRow is one line of the "Transfers" table: its own date, a from/to money
// account pair, an amount and a payee.
type trRow struct {
	date   *widget.Entry
	from   *widget.Select
	to     *widget.Select
	amount *widget.Entry
	payee  *completionEntry
	box    fyne.CanvasObject
}

// QuickAddForm opens the bulk-entry modal: two appendable tables (income/expense
// rows and transfer rows), each row carrying its own date so a backlog spanning
// several days can be entered in one pass. Saving commits every valid row in one
// batch via store.AddTransactions, instead of one dialog per entry.
func QuickAddForm() {
	if win == nil {
		return
	}

	moneyNames := namesOf(store.MoneyAccounts())
	catNames := append(append([]string{}, namesOf(store.ExpenseAccounts())...), namesOf(store.IncomeAccounts())...)
	payeeNames := store.Payees()
	defaultMoney := ""
	if len(moneyNames) > 0 {
		defaultMoney = moneyNames[0]
	}

	// Column weights: give the date its own wider column so the calendar-picker
	// button reads clearly (as on the single transaction form) instead of being
	// squeezed into an equal fifth of the row.
	colWeights := []float32{1.4, 1, 1, 1, 1.1}

	// ---- Income & Expenses table ----
	var ieRows []*ieRow
	ieList := container.NewVBox()
	var rebuildIE func()
	var removeIE func(*ieRow)
	var addIERow func()
	rebuildIE = func() {
		objs := make([]fyne.CanvasObject, 0, len(ieRows))
		for _, r := range ieRows {
			objs = append(objs, r.box)
		}
		ieList.Objects = objs
		ieList.Refresh()
	}
	removeIE = func(target *ieRow) {
		kept := ieRows[:0]
		for _, r := range ieRows {
			if r != target {
				kept = append(kept, r)
			}
		}
		ieRows = kept
		rebuildIE()
	}
	addIERow = func() {
		r := &ieRow{
			date:   newDateField(todaySerial),
			money:  widget.NewSelect(moneyNames, nil),
			amount: newAmountEntry(),
			cat:    widget.NewSelect(catNames, nil),
			payee:  newCompletionEntry(payeeNames),
		}
		r.money.PlaceHolder = "Money…"
		if defaultMoney != "" {
			r.money.SetSelected(defaultMoney)
		}
		r.cat.PlaceHolder = "Category…"
		r.payee.SetPlaceHolder("Payee")
		r.box = rowWithRemove(
			container.New(&columnsLayout{Weights: colWeights, Gap: 6}, dateCell(r.date), r.money, r.amount, r.cat, r.payee),
			func() { removeIE(r) },
		)
		ieRows = append(ieRows, r)
		rebuildIE()
	}

	// ---- Transfers table ----
	var trRows []*trRow
	trList := container.NewVBox()
	var rebuildTR func()
	var removeTR func(*trRow)
	var addTRRow func()
	rebuildTR = func() {
		objs := make([]fyne.CanvasObject, 0, len(trRows))
		for _, r := range trRows {
			objs = append(objs, r.box)
		}
		trList.Objects = objs
		trList.Refresh()
	}
	removeTR = func(target *trRow) {
		kept := trRows[:0]
		for _, r := range trRows {
			if r != target {
				kept = append(kept, r)
			}
		}
		trRows = kept
		rebuildTR()
	}
	addTRRow = func() {
		r := &trRow{
			date:   newDateField(todaySerial),
			from:   widget.NewSelect(moneyNames, nil),
			to:     widget.NewSelect(moneyNames, nil),
			amount: newAmountEntry(),
			payee:  newCompletionEntry(payeeNames),
		}
		r.from.PlaceHolder = "From…"
		r.to.PlaceHolder = "To…"
		r.payee.SetPlaceHolder("Payee")
		r.box = rowWithRemove(
			container.New(&columnsLayout{Weights: colWeights, Gap: 6}, dateCell(r.date), r.from, r.to, r.amount, r.payee),
			func() { removeTR(r) },
		)
		trRows = append(trRows, r)
		rebuildTR()
	}

	addIERow()
	addTRRow()

	// commit gathers every valid row from both tables and posts them in one batch.
	commit := func() (int, error) {
		var txns []Transaction
		for _, r := range ieRows {
			serial := parseDateSerial(r.date.Text)
			amt := parseAmount(r.amount.Text)
			money := idOf(r.money.Selected)
			if serial == 0 || amt <= 0 || money == "" || r.cat.Selected == "" {
				continue
			}
			a := store.AccountByName(r.cat.Selected)
			if a == nil {
				continue
			}
			kind := "Expense"
			if a.Type == Income {
				kind = "Income"
			}
			txns = append(txns, Transaction{
				Date:  serial,
				Payee: r.payee.Text,
				Posts: postingsFor(kind, money, a.ID, amt),
			})
		}
		for _, r := range trRows {
			serial := parseDateSerial(r.date.Text)
			amt := parseAmount(r.amount.Text)
			from, to := idOf(r.from.Selected), idOf(r.to.Selected)
			if serial == 0 || amt <= 0 || from == "" || to == "" || from == to {
				continue
			}
			txns = append(txns, Transaction{
				Date:  serial,
				Payee: r.payee.Text,
				Posts: postingsFor("Transfer", from, to, amt),
			})
		}
		return store.AddTransactions(txns)
	}

	body := container.NewVBox(
		sectionTitle("Income & Expenses"),
		columnHeader(colWeights, "Date", "Money", "Amount", "Category", "Payee"),
		ieList,
		secondaryButton("＋ Add row", theme.ContentAddIcon(), func() { addIERow() }),
		spacerH(14),
		sectionTitle("Transfers"),
		columnHeader(colWeights, "Date", "From", "To", "Amount", "Payee"),
		trList,
		secondaryButton("＋ Add transfer", theme.ContentAddIcon(), func() { addTRRow() }),
	)

	var d *dialog.CustomDialog
	cancel := secondaryButton("Cancel", nil, func() { d.Hide() })
	save := primaryButton("Save all", theme.ConfirmIcon(), func() {
		n, err := commit()
		d.Hide()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		showInfo("Added", itoa(n)+" transaction"+plural(n, "", "s")+" added.")
	})

	content := container.NewBorder(nil,
		container.NewVBox(spacerH(8), container.NewHBox(layout.NewSpacer(), cancel, save)),
		nil, nil,
		container.NewScroll(container.NewPadded(body)),
	)
	d = dialog.NewCustomWithoutButtons("Quick Add", content, win)
	d.Resize(fyne.NewSize(820, 620))
	d.Show()
}

// dateCell pairs a bare YYYY-MM-DD field with an explicit calendar-picker button
// docked to its right. Unlike Entry.ActionItem (used on the single transaction
// form), the button is a real layout sibling here, so it always renders inside
// the tight Quick Add row instead of being swallowed by the entry's renderer.
func dateCell(e *widget.Entry) fyne.CanvasObject {
	return container.NewBorder(nil, nil, nil, calendarButton(e), e)
}

// rowWithRemove docks a small ✕ button at the right edge of a row's fields.
func rowWithRemove(fields fyne.CanvasObject, onRemove func()) fyne.CanvasObject {
	rm := widget.NewButton("✕", onRemove)
	rm.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, nil, rm, fields)
}

// columnHeader renders dim labels above a table's columns, using the same column
// weights as the rows and padding on the right to line up with the remove button.
func columnHeader(weights []float32, labels ...string) fyne.CanvasObject {
	cols := make([]fyne.CanvasObject, len(labels))
	for i, l := range labels {
		cols[i] = txt(l, colTextDim, 10, true)
	}
	return container.NewBorder(nil, nil, nil, spacerW(40),
		container.New(&columnsLayout{Weights: weights, Gap: 6}, cols...))
}
