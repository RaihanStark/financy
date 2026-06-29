package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ieRow is one line of the "Income & Expenses" table: an amount, a category
// (whose account type decides income vs expense), and a payee.
type ieRow struct {
	amount *widget.Entry
	cat    *widget.Select
	payee  *widget.Entry
	box    fyne.CanvasObject
}

// trRow is one line of the "Transfers" table: a from/to money account pair plus
// an amount and payee.
type trRow struct {
	from   *widget.Select
	to     *widget.Select
	amount *widget.Entry
	payee  *widget.Entry
	box    fyne.CanvasObject
}

// QuickAddForm opens the bulk-entry modal: two appendable tables (income/expense
// rows that share a money account, and transfer rows) sharing a single date.
// Saving commits every valid row in one batch via store.AddTransactions, so a
// stack of receipts becomes one quick pass instead of one dialog per entry.
func QuickAddForm() {
	if win == nil {
		return
	}

	moneyNames := namesOf(store.MoneyAccounts())
	catNames := append(append([]string{}, namesOf(store.ExpenseAccounts())...), namesOf(store.IncomeAccounts())...)

	date := newDateEntry(todaySerial)
	moneyAcct := widget.NewSelect(moneyNames, nil)
	moneyAcct.PlaceHolder = "Money account…"
	if len(moneyNames) > 0 {
		moneyAcct.SetSelected(moneyNames[0])
	}

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
		r := &ieRow{amount: newAmountEntry(), cat: widget.NewSelect(catNames, nil), payee: widget.NewEntry()}
		r.cat.PlaceHolder = "Category…"
		r.payee.SetPlaceHolder("Payee")
		r.box = rowWithRemove(
			container.NewGridWithColumns(3, r.amount, r.cat, r.payee),
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
			from:   widget.NewSelect(moneyNames, nil),
			to:     widget.NewSelect(moneyNames, nil),
			amount: newAmountEntry(),
			payee:  widget.NewEntry(),
		}
		r.from.PlaceHolder = "From…"
		r.to.PlaceHolder = "To…"
		r.payee.SetPlaceHolder("Payee")
		r.box = rowWithRemove(
			container.NewGridWithColumns(4, r.from, r.to, r.amount, r.payee),
			func() { removeTR(r) },
		)
		trRows = append(trRows, r)
		rebuildTR()
	}

	addIERow()
	addTRRow()

	// commit gathers every valid row from both tables and posts them in one batch.
	commit := func() (int, error) {
		serial := parseDateSerial(date.Text)
		money := idOf(moneyAcct.Selected)
		var txns []Transaction
		if serial != 0 {
			for _, r := range ieRows {
				amt := parseAmount(r.amount.Text)
				if amt <= 0 || money == "" || r.cat.Selected == "" {
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
				amt := parseAmount(r.amount.Text)
				from, to := idOf(r.from.Selected), idOf(r.to.Selected)
				if amt <= 0 || from == "" || to == "" || from == to {
					continue
				}
				txns = append(txns, Transaction{
					Date:  serial,
					Payee: r.payee.Text,
					Posts: postingsFor("Transfer", from, to, amt),
				})
			}
		}
		return store.AddTransactions(txns)
	}

	body := container.NewVBox(
		container.NewGridWithColumns(2,
			labeledControl("Date (YYYY-MM-DD)", date),
			labeledControl("Money account", moneyAcct),
		),
		spacerH(10),
		sectionTitle("Income & Expenses"),
		columnHeader("Amount", "Category", "Payee"),
		ieList,
		secondaryButton("＋ Add row", theme.ContentAddIcon(), func() { addIERow() }),
		spacerH(14),
		sectionTitle("Transfers"),
		columnHeader("From", "To", "Amount", "Payee"),
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
	d.Resize(fyne.NewSize(640, 600))
	d.Show()
}

// rowWithRemove docks a small ✕ button at the right edge of a row's fields.
func rowWithRemove(fields fyne.CanvasObject, onRemove func()) fyne.CanvasObject {
	rm := widget.NewButton("✕", onRemove)
	rm.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, nil, rm, fields)
}

// columnHeader renders dim labels above a table's columns, padded on the right to
// line up with rows that carry a remove button.
func columnHeader(labels ...string) fyne.CanvasObject {
	cols := make([]fyne.CanvasObject, len(labels))
	for i, l := range labels {
		cols[i] = txt(l, colTextDim, 10, true)
	}
	return container.NewBorder(nil, nil, nil, spacerW(40),
		container.NewGridWithColumns(len(labels), cols...))
}
