package view

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

const noneOpt = "— none —"

// dateFormatOptions are the friendly date layouts offered in the mapping step.
// An empty layout means "auto-detect" (try all known formats per row).
var dateFormatOptions = []struct{ label, layout string }{
	{"Auto-detect", ""},
	{"YYYY-MM-DD  (2026-06-23)", "2006-01-02"},
	{"MM/DD/YYYY  (06/23/2026)", "01/02/2006"},
	{"DD/MM/YYYY  (23/06/2026)", "02/01/2006"},
	{"DD-MM-YYYY  (23-06-2026)", "02-01-2006"},
	{"DD.MM.YYYY  (23.06.2026)", "02.01.2006"},
	{"MM/DD/YY  (06/23/26)", "01/02/06"},
	{"Mon D, YYYY  (Jun 23, 2026)", "Jan 2, 2006"},
}

// ImportCSV is the entry point: pick a CSV file, then run the guided wizard.
func ImportCSV() {
	if win == nil || store == nil {
		return
	}
	if len(store.MoneyAccounts()) == 0 {
		showInfo("No accounts yet", "Create an asset or liability account first, then import into it.")
		return
	}

	fd := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil || r == nil {
			return
		}
		defer func() { _ = r.Close() }()
		data, e := io.ReadAll(r)
		if e != nil {
			showError(e)
			return
		}
		records, e := core.ParseCSV(data)
		if e != nil || len(records) == 0 {
			showInfo("Couldn't read CSV", "The file didn't parse as CSV, or it's empty.")
			return
		}
		store.EnsureImportCategories() // so "Uncategorized" appears in the pickers
		showImportMapping(records)
	}, win)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".csv", ".txt"}))
	fd.Show()
}

// showImportMapping is the guided column-mapping step.
func showImportMapping(records [][]string) {
	guess := core.GuessSpec(records[0], sampleRows(records))
	labels := columnLabels(records)

	// Target money account.
	acctNames := groupedLabels(store.MoneyAccounts())
	account := widget.NewSelect(acctNames, nil)
	account.SetSelected(firstSelectable(acctNames))
	guardGroupHeaders(account)

	header := widget.NewCheck("First row is a header", nil)
	header.SetChecked(true)

	dateCol := colSelect(labels, guess.DateCol, false)
	payeeCol := colSelect(labels, guess.PayeeCol, false)
	memoCol := colSelect(labels, guess.MemoCol, true)

	mode := widget.NewSelect([]string{"Single amount column", "Separate debit & credit"}, nil)
	amountCol := colSelect(labels, guess.AmountCol, false)
	debitCol := colSelect(labels, guess.DebitCol, false)
	creditCol := colSelect(labels, guess.CreditCol, false)

	amountRow := labeledRow("Amount", amountCol)
	debitRow := labeledRow("Debit (out)", debitCol)
	creditRow := labeledRow("Credit (in)", creditCol)
	updateMode := func(m string) {
		single := m == "Single amount column"
		setVisible(amountRow, single)
		setVisible(debitRow, !single)
		setVisible(creditRow, !single)
	}
	mode.OnChanged = updateMode
	if guess.AmountMode == core.AmountDebitCredit {
		mode.SetSelected("Separate debit & credit")
	} else {
		mode.SetSelected("Single amount column")
	}

	dateFmt := widget.NewSelect(dateFormatLabels(), nil)
	dateFmt.SetSelected(labelForLayout(guess.DateLayout))

	flip := widget.NewCheck("My export uses + for money spent (flip the sign)", nil)

	expCats := groupedLabels(store.ExpenseAccounts())
	incCats := groupedLabels(store.IncomeAccounts())
	expCat := widget.NewSelect(expCats, nil)
	expCat.SetSelected(orFirst(expCats, "Uncategorized"))
	incCat := widget.NewSelect(incCats, nil)
	incCat.SetSelected(orFirst(incCats, "Uncategorized Income"))

	form := container.NewVBox(
		txt("Tell Financy what each column means. We've guessed from the header — adjust if needed.", colTextDim, 11.5, false),
		spacerH(8),
		labeledRow("Into account", account),
		labeledRow("", header),
		divider(),
		labeledRow("Date", dateCol),
		labeledRow("Date format", dateFmt),
		labeledRow("Payee", payeeCol),
		labeledRow("Memo", memoCol),
		divider(),
		labeledRow("Amounts", mode),
		amountRow, debitRow, creditRow,
		labeledRow("", flip),
		divider(),
		labeledRow("Default expense category", expCat),
		labeledRow("Default income category", incCat),
	)
	updateMode(mode.Selected)

	var d *modal
	preview := primaryButton("Preview…", nil, func() {
		spec := core.ImportSpec{
			AccountID:    idForName(store.MoneyAccounts(), account.Selected),
			HasHeader:    header.Checked,
			DateCol:      colIndexOf(dateCol.Selected),
			PayeeCol:     colIndexOf(payeeCol.Selected),
			MemoCol:      colIndexOf(memoCol.Selected),
			AmountCol:    colIndexOf(amountCol.Selected),
			DebitCol:     colIndexOf(debitCol.Selected),
			CreditCol:    colIndexOf(creditCol.Selected),
			NegateAmount: flip.Checked,
			DateLayout:   layoutForLabel(dateFmt.Selected),
			IncomeCatID:  idForName(store.IncomeAccounts(), incCat.Selected),
			ExpenseCatID: idForName(store.ExpenseAccounts(), expCat.Selected),
		}
		if mode.Selected == "Separate debit & credit" {
			spec.AmountMode = core.AmountDebitCredit
		}
		d.Hide()
		showImportPreview(records, spec)
	})
	cancel := secondaryButton("Cancel", nil, func() { d.Hide() })

	content := container.NewBorder(nil,
		container.NewVBox(spacerH(8), container.NewHBox(layoutSpacer(), cancel, preview)),
		nil, nil,
		container.NewScroll(form),
	)
	d = newModal("Import CSV", content)
	d.SetCardSize(fyne.NewSize(540, 620))
	d.Show()
}

// showImportPreview builds the rows from the spec and lets the user confirm.
func showImportPreview(records [][]string, spec core.ImportSpec) {
	rows := store.BuildImport(records, spec)

	var okN, dupN, errN int
	for _, r := range rows {
		switch r.Status {
		case core.RowOK:
			okN++
		case core.RowDuplicate:
			dupN++
		default:
			errN++
		}
	}

	incCats, incAccts := groupedLabels(store.IncomeAccounts()), store.IncomeAccounts()
	expCats, expAccts := groupedLabels(store.ExpenseAccounts()), store.ExpenseAccounts()

	tbl := tableSpec{
		Headers: []string{"#", "Date", "Payee", "Amount", "Category", "Status"},
		Weights: []float32{0.4, 1, 2, 1.1, 1.8, 1.2},
		Aligns:  []fyne.TextAlign{fyne.TextAlignLeading, fyne.TextAlignLeading, fyne.TextAlignLeading, fyne.TextAlignTrailing, fyne.TextAlignLeading, fyne.TextAlignLeading},
	}
	for i := range rows {
		r := rows[i]
		statusCell, amtCol := tc("Ready"), colPositive
		if r.Amount < 0 {
			amtCol = colNegative
		}
		switch r.Status {
		case core.RowDuplicate:
			statusCell = tcc("Duplicate", colWarning)
		case core.RowError:
			statusCell, amtCol = tcc("Skip · "+r.Reason, colNegative), colTextDim
		}

		date, amount, catCell := "—", "—", tc("—")
		if r.Status != core.RowError {
			date, amount = fmtSerialDate(r.Date), fmtMoney(r.Amount)
			opts, accts := incCats, incAccts
			if r.Amount < 0 {
				opts, accts = expCats, expAccts
			}
			sel := widget.NewSelect(opts, nil)
			sel.SetSelected(labelOf(r.CategoryID))
			idx := i
			sel.OnChanged = func(name string) {
				id := idForName(accts, name)
				rows[idx].CategoryID = id
				rows[idx].Txn = core.BuildImportTxn(spec.AccountID, id,
					rows[idx].Date, rows[idx].Payee, rows[idx].Memo, rows[idx].Amount)
			}
			catCell = tco(sel)
		}
		tbl.Rows = append(tbl.Rows, []cell{
			tc(itoa(r.Line)), tc(date), tc(orDash(r.Payee)),
			tcc(amount, amtCol), catCell, statusCell,
		})
	}

	includeDups := widget.NewCheck(fmt.Sprintf("Also import %d duplicate(s)", dupN), nil)
	if dupN == 0 {
		includeDups.Disable()
	}

	summary := txt(fmt.Sprintf("%d ready · %d duplicate · %d skipped", okN, dupN, errN), colTextDim, 12, false)

	body := container.NewBorder(
		container.NewVBox(
			txt("Into "+nameOf(spec.AccountID)+" — duplicates and unparseable rows are excluded by default.", colTextDim, 11.5, false),
			spacerH(4), summary, spacerH(6), divider(),
		),
		nil, nil, nil,
		container.NewScroll(tbl.Build()),
	)

	var d *modal
	back := secondaryButton("Back", nil, func() { d.Hide(); showImportMapping(records) })
	cancel := secondaryButton("Cancel", nil, func() { d.Hide() })
	doImport := primaryButton(fmt.Sprintf("Import %d", okN), nil, func() {
		var commit []core.Transaction
		for _, r := range rows {
			if r.Status == core.RowOK || (r.Status == core.RowDuplicate && includeDups.Checked) {
				commit = append(commit, r.Txn)
			}
		}
		d.Hide()
		n, err := store.AddTransactions(commit)
		if err != nil {
			showError(err)
			return
		}
		showInfo("Import complete", fmt.Sprintf("Imported %d transaction(s) into %s.\nSkipped %d duplicate(s) and %d unparseable row(s).",
			n, nameOf(spec.AccountID), dupN-importedDups(rows, includeDups.Checked), errN))
	})

	footer := container.NewVBox(spacerH(6), includeDups,
		container.NewHBox(layoutSpacer(), cancel, back, doImport))
	content := container.NewBorder(nil, footer, nil, nil, body)

	d = newModal("Review import", content)
	d.SetCardSize(fyne.NewSize(760, 600))
	d.Show()
}

// ---- helpers ----

func importedDups(rows []core.ImportRow, include bool) int {
	if !include {
		return 0
	}
	n := 0
	for _, r := range rows {
		if r.Status == core.RowDuplicate {
			n++
		}
	}
	return n
}

func sampleRows(records [][]string) [][]string {
	end := min(len(records), 6)
	if len(records) > 1 {
		return records[1:end]
	}
	return nil
}

func columnLabels(records [][]string) []string {
	n := 0
	for _, r := range records {
		if len(r) > n {
			n = len(r)
		}
	}
	labels := make([]string, n)
	for i := range labels {
		name := fmt.Sprintf("Column %d", i+1)
		if len(records) > 0 && i < len(records[0]) {
			if h := strings.TrimSpace(records[0][i]); h != "" {
				name = h
			}
		}
		labels[i] = fmt.Sprintf("%d. %s", i+1, name)
	}
	return labels
}

func colSelect(labels []string, idx int, includeNone bool) *widget.Select {
	opts := labels
	if includeNone {
		opts = append([]string{noneOpt}, labels...)
	}
	sel := widget.NewSelect(opts, nil)
	switch {
	case includeNone && idx < 0:
		sel.SetSelected(noneOpt)
	case idx >= 0 && idx < len(labels):
		sel.SetSelected(labels[idx])
	case len(opts) > 0:
		sel.SetSelected(opts[0])
	}
	return sel
}

func colIndexOf(sel string) int {
	if sel == "" || sel == noneOpt {
		return -1
	}
	if i := strings.IndexByte(sel, '.'); i > 0 {
		if n, err := strconv.Atoi(strings.TrimSpace(sel[:i])); err == nil {
			return n - 1
		}
	}
	return -1
}

func dateFormatLabels() []string {
	out := make([]string, len(dateFormatOptions))
	for i, o := range dateFormatOptions {
		out[i] = o.label
	}
	return out
}

func labelForLayout(layout string) string {
	for _, o := range dateFormatOptions {
		if o.layout == layout {
			return o.label
		}
	}
	return dateFormatOptions[0].label // Auto-detect
}

func layoutForLabel(label string) string {
	for _, o := range dateFormatOptions {
		if o.label == label {
			return o.layout
		}
	}
	return ""
}

func labeledRow(label string, ctl fyne.CanvasObject) fyne.CanvasObject {
	if label == "" {
		return ctl
	}
	return container.NewBorder(nil, nil,
		container.NewGridWrap(fyne.NewSize(180, 34), txt(label, colTextDim, 12, false)),
		nil, ctl)
}

func setVisible(o fyne.CanvasObject, v bool) {
	if v {
		o.Show()
	} else {
		o.Hide()
	}
}

func layoutSpacer() fyne.CanvasObject { return layout.NewSpacer() }

func orFirst(opts []string, want string) string {
	for _, o := range opts {
		if o == want {
			return o
		}
	}
	if len(opts) > 0 {
		return opts[0]
	}
	return ""
}

// idForName resolves a selector's displayed string (a disambiguated label, or a
// bare account name) back to its account ID within the given slice.
func idForName(accts []core.Account, name string) string {
	for _, a := range accts {
		if core.AccountLabel(a) == name || a.Name == name {
			return a.ID
		}
	}
	return ""
}
