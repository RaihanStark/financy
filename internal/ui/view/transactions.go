package view

import (
	"errors"
	"image/color"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Filter state persists across re-renders.
var (
	filterMonth   = "All months"
	filterType    = "All types"
	filterAccount = "All accounts"
	filterSearch  = ""
)

// Bulk-selection state for recategorizing many transactions at once.
var (
	selecting bool
	selected  = map[string]bool{}
)

// txnView is a human-facing interpretation of a balanced transaction.
type txnView struct {
	kind      string // Expense / Income / Transfer / Opening / Split
	moneyID   string // the asset/liability account involved
	catID     string // the income/expense account involved
	fromID    string // transfer source
	toID      string // transfer destination
	moneyName string
	catName   string
	amount    int // positive magnitude
}

func deriveView(t Transaction) txnView {
	v := txnView{kind: "Split"}
	if len(t.Posts) != 2 {
		// Show the largest positive posting as the amount.
		for _, p := range t.Posts {
			if p.Amount > v.amount {
				v.amount = p.Amount
			}
		}
		return v
	}
	a0 := store.AccountByID(t.Posts[0].AccountID)
	a1 := store.AccountByID(t.Posts[1].AccountID)
	if a0 == nil || a1 == nil {
		return v
	}
	typeOf := func(a *Account) AcctType { return a.Type }
	has := func(tp AcctType) (*Account, int, *Account) {
		if typeOf(a0) == tp {
			return a0, t.Posts[0].Amount, a1
		}
		if typeOf(a1) == tp {
			return a1, t.Posts[1].Amount, a0
		}
		return nil, 0, nil
	}

	if exp, amt, other := has(Expense); exp != nil {
		v.kind, v.catID, v.catName, v.amount = "Expense", exp.ID, exp.Name, amt
		v.moneyID, v.moneyName = other.ID, other.Name
		return v
	}
	if inc, amt, other := has(Income); inc != nil {
		v.kind, v.catID, v.catName, v.amount = "Income", inc.ID, inc.Name, -amt
		v.moneyID, v.moneyName = other.ID, other.Name
		return v
	}
	if eq, amt, other := has(Equity); eq != nil {
		v.kind = "Opening"
		v.moneyID, v.moneyName = other.ID, other.Name
		v.amount = abs(amt)
		return v
	}
	// Transfer between money accounts.
	v.kind = "Transfer"
	if t.Posts[0].Amount < 0 {
		v.fromID, v.toID = a0.ID, a1.ID
	} else {
		v.fromID, v.toID = a1.ID, a0.ID
	}
	v.moneyName = store.AccountByID(v.fromID).Name + " → " + store.AccountByID(v.toID).Name
	v.amount = abs(t.Posts[0].Amount)
	return v
}

// displayPayee is the title shown for a transaction row. When the payee is
// blank it falls back to the transaction's type (e.g. "Expense", "Transfer")
// rather than leaving the line empty.
func displayPayee(t Transaction, v txnView) string {
	if t.Payee != "" {
		return t.Payee
	}
	return v.kind
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func ScreenTransactions() fyne.CanvasObject {
	var bar fyne.CanvasObject
	if selecting {
		recat := primaryButton("Recategorize…", theme.ContentCopyIcon(), func() {
			if len(selected) > 0 {
				bulkRecategorize()
			}
		})
		if len(selected) == 0 {
			recat.Disable()
		}
		bar = appBar("Transactions", itoa(len(selected))+" selected — tap rows to choose",
			recat,
			secondaryButton("Done", theme.CancelIcon(), func() { exitSelectMode() }),
		)
	} else {
		bar = appBar("Transactions", "Double-entry journal — every entry balances to zero",
			secondaryButton("Select", theme.ListIcon(), func() { selecting = true; render() }),
			secondaryButton("Import CSV", theme.DownloadIcon(), func() { ImportCSV() }),
			secondaryButton("Quick Add", theme.ContentAddIcon(), func() { QuickAddForm() }),
			primaryButton("Add Transaction", theme.ContentAddIcon(), func() { TransactionForm("", "") }),
		)
	}

	months := append([]string{"All months"}, distinctMonths()...)
	monthSel := widget.NewSelect(months, nil)
	monthSel.SetSelected(filterMonth)
	monthSel.OnChanged = func(v string) { filterMonth = v; render() }
	typeSel := widget.NewSelect([]string{"All types", "Income", "Expense", "Transfer", "Opening"}, nil)
	typeSel.SetSelected(filterType)
	typeSel.OnChanged = func(v string) { filterType = v; render() }
	acctSel := widget.NewSelect(append([]string{"All accounts"}, namesOf(store.MoneyAccounts())...), nil)
	acctSel.SetSelected(filterAccount)
	acctSel.OnChanged = func(v string) { filterAccount = v; render() }
	search := widget.NewEntry()
	search.SetPlaceHolder("Search payee / memo / category…  (Enter)")
	search.SetText(filterSearch)
	search.OnSubmitted = func(v string) { filterSearch = v; render() }
	clearBtn := secondaryButton("Clear", theme.ContentClearIcon(), func() {
		filterMonth, filterType, filterAccount, filterSearch = "All months", "All types", "All accounts", ""
		render()
	})

	filters := panel(container.NewVBox(
		container.NewHBox(labeledControl("Month", monthSel), labeledControl("Type", typeSel), labeledControl("Account", acctSel)),
		container.NewBorder(nil, nil, nil, clearBtn, labeledControl("Search", search)),
	))

	rows := filteredTransactions()
	var inc, exp int
	for _, t := range rows {
		v := deriveView(t)
		switch v.kind {
		case "Income":
			inc += v.amount
		case "Expense":
			exp += v.amount
		}
	}
	summary := container.NewGridWithColumns(4,
		statCard("SHOWING", itoa(len(rows)), colText, "of "+itoa(len(store.Transactions()))+" entries"),
		statCard("INCOME (filtered)", fmtMoney(inc), colPositive, ""),
		statCard("EXPENSES (filtered)", fmtMoney(exp), colNegative, ""),
		statCard("NET (filtered)", fmtMoney(inc-exp), moneyColor(inc-exp), ""),
	)

	items := []fyne.CanvasObject{filters, spacerH(4)}
	if selecting {
		items = append(items, selectionBar(rows), spacerH(4))
	} else {
		items = append(items, summary, spacerH(4))
	}
	items = append(items, panel(container.NewVBox(sectionTitle("Journal"), spacerH(4), txnLedger(rows))))
	body := container.NewVBox(items...)
	return container.NewBorder(bar, nil, nil, nil, container.NewPadded(body))
}

// txnLedger renders transactions as clean rows grouped under date headers.
func txnLedger(rows []Transaction) fyne.CanvasObject {
	if len(rows) == 0 {
		return container.New(padCell(20, 8), emptyState("No transactions match the current filters"))
	}

	// Net income−expense per day, for the date headers.
	dayNet := map[int]int{}
	for _, t := range rows {
		v := deriveView(t)
		switch v.kind {
		case "Income":
			dayNet[t.Date] += v.amount
		case "Expense":
			dayNet[t.Date] -= v.amount
		}
	}

	box := container.NewVBox()
	curDate := -1
	for i, t := range rows {
		if t.Date != curDate {
			if i > 0 {
				box.Add(spacerH(8))
			}
			box.Add(dateHeader(t.Date, dayNet[t.Date]))
			curDate = t.Date
		}
		box.Add(txnRow(t))
		box.Add(divider())
	}
	return box
}

func dateHeader(date, net int) fyne.CanvasObject {
	bg := canvas.NewRectangle(colSurfaceHi)
	bg.CornerRadius = radiusSm
	netCol := colTextDim
	netStr := "Net " + fmtMoney(net)
	if net > 0 {
		netCol = colPositive
	} else if net < 0 {
		netCol = colNegative
	}
	row := container.NewBorder(nil, nil,
		txt(relDate(date), colText, 11.5, true),
		alignRight(txt(netStr, netCol, 11, true)))
	return container.NewStack(bg, container.New(padCell(4, 8), row))
}

func relDate(serial int) string {
	switch serial {
	case todaySerial:
		return "Today · " + serialToTime(serial).Format("2 Jan 2006")
	case todaySerial - 1:
		return "Yesterday · " + serialToTime(serial).Format("2 Jan 2006")
	}
	return serialToTime(serial).Format("Monday, 2 Jan 2006")
}

// txnRow is one transaction line: type chip, payee/details, amount.
func txnRow(t Transaction) fyne.CanvasObject {
	v := deriveView(t)
	amtCol := colText
	amtStr := fmtMoney(v.amount)
	switch v.kind {
	case "Income":
		amtCol, amtStr = colPositive, "+"+fmtMoney(v.amount)
	case "Expense":
		amtCol, amtStr = colNegative, "−"+fmtMoney(v.amount)
	}

	// Secondary line describes where the money moved.
	var sub string
	switch v.kind {
	case "Transfer":
		sub = v.moneyName // already "A → B"
	case "Opening":
		sub = "Opening Balance · " + v.moneyName
	default:
		sub = v.catName + " · " + v.moneyName
	}
	center := container.NewVBox(
		txt(displayPayee(t, v), colText, 13, true),
		txt(sub, colTextDim, 10.5, false),
	)
	right := container.NewVBox(
		alignRight(mono(amtStr, amtCol, 14, true)),
		alignRight(txt(v.kind, colTextDim, 9.5, false)),
	)

	lead := container.NewHBox(typeChip(v.kind), spacerW(8))
	if selecting {
		lead = container.NewHBox(checkBox(selected[t.ID]), spacerW(6), typeChip(v.kind), spacerW(8))
	}
	inner := container.NewBorder(nil, nil, lead, right, center)

	var bg color.Color = colSurface
	if selecting && selected[t.ID] {
		bg = withAlpha(colPrimary, 0x14)
	}
	onTap := func() { TransactionForm(t.ID, "") }
	if selecting {
		onTap = func() { toggleSelect(t.ID) }
	}
	row := newTappableRow(container.New(padCell(7, 10), inner), bg, onTap)
	if !selecting {
		row.SetOnSecondary(func(pos fyne.Position) { showContextMenu(pos, txnMenuItems(t)...) })
	}
	return row
}

// checkBox is a small square selection indicator for bulk-select rows.
func checkBox(on bool) fyne.CanvasObject {
	bg := canvas.NewRectangle(colSurface)
	bg.CornerRadius = 4
	bg.StrokeColor = colBorder
	bg.StrokeWidth = 1
	glyph := spacerW(0)
	if on {
		bg.FillColor = colPrimary
		bg.StrokeColor = colPrimary
		g := txt("✓", colSurface, 13, true)
		g.Alignment = fyne.TextAlignCenter
		glyph = g
	}
	return container.NewGridWrap(fyne.NewSize(18, 18), container.NewStack(bg, container.NewCenter(glyph)))
}

func toggleSelect(id string) {
	if selected[id] {
		delete(selected, id)
	} else {
		selected[id] = true
	}
	render()
}

func exitSelectMode() {
	selecting = false
	selected = map[string]bool{}
	render()
}

// selectionBar shows the running count plus select-all / clear over the rows
// currently in view (so filters + Select all is a quick way to scope a batch).
func selectionBar(rows []Transaction) fyne.CanvasObject {
	allSelected := len(rows) > 0
	for _, t := range rows {
		if !selected[t.ID] {
			allSelected = false
			break
		}
	}
	toggleAll := secondaryButton("Select all shown", theme.ListIcon(), func() {
		for _, t := range rows {
			selected[t.ID] = true
		}
		render()
	})
	if allSelected {
		toggleAll = secondaryButton("Unselect all shown", theme.ContentClearIcon(), func() {
			for _, t := range rows {
				delete(selected, t.ID)
			}
			render()
		})
	}
	clear := secondaryButton("Clear selection", theme.CancelIcon(), func() {
		selected = map[string]bool{}
		render()
	})
	left := txt(itoa(len(selected))+" selected · "+itoa(len(rows))+" shown", colText, 12, true)
	return panel(container.NewBorder(nil, nil, left, container.NewHBox(toggleAll, clear)))
}

// bulkRecategorize asks for a target category and reassigns it to every selected
// transaction whose kind matches (the core skips transfers/openings/mismatches).
func bulkRecategorize() {
	expense := namesOf(store.ExpenseAccounts())
	income := namesOf(store.IncomeAccounts())
	cat := widget.NewSelect(append(append([]string{}, expense...), income...), nil)
	cat.PlaceHolder = "Choose a category…"

	n := len(selected)
	hint := widget.NewLabel(itoa(n) + " transaction(s) selected. Transfers, opening balances and " +
		"entries that don't match the category's type are left unchanged.")
	hint.Wrapping = fyne.TextWrapWord

	items := []*widget.FormItem{
		widget.NewFormItem("New category", cat),
		widget.NewFormItem("", hint),
	}
	showForm("Recategorize transactions", items, func() {
		if cat.Selected == "" {
			return
		}
		ids := make([]string, 0, len(selected))
		for id := range selected {
			ids = append(ids, id)
		}
		changed := store.RecategorizeTransactions(ids, idOf(cat.Selected))
		exitSelectMode()
		showInfo("Recategorized", itoa(changed)+" of "+itoa(n)+
			" selected transaction(s) moved to "+cat.Selected+".")
	})
}

// typeChip is the small colored square that flags the transaction kind.
func typeChip(kind string) fyne.CanvasObject {
	col := kindColor(kind)
	glyph := map[string]string{"Income": "+", "Expense": "−", "Transfer": "⇄", "Opening": "•", "Split": "≡"}[kind]
	if glyph == "" {
		glyph = "•"
	}
	bg := canvas.NewRectangle(withAlpha(col, 0x1f))
	bg.StrokeColor = withAlpha(col, 0x70)
	bg.StrokeWidth = 1
	bg.CornerRadius = radiusSm
	g := txt(glyph, col, 16, true)
	g.Alignment = fyne.TextAlignCenter
	return container.NewGridWrap(fyne.NewSize(32, 32), container.NewStack(bg, container.NewCenter(g)))
}

func txnMenuItems(t Transaction) []*fyne.MenuItem {
	v := deriveView(t)
	edit := fyne.NewMenuItem("Edit…", func() { TransactionForm(t.ID, "") })
	edit.Icon = theme.DocumentCreateIcon()
	dup := fyne.NewMenuItem("Duplicate", func() { duplicateTxn(t) })
	dup.Icon = theme.ContentCopyIcon()
	del := fyne.NewMenuItem("Delete", func() {
		confirmDelete("this transaction", func() { store.DeleteTransaction(t.ID) })
	})
	del.Icon = theme.DeleteIcon()

	items := []*fyne.MenuItem{edit, dup}
	if a := store.AccountByID(v.moneyID); a != nil && (a.Type == Asset || a.Type == Liability) {
		open := fyne.NewMenuItem("Open "+a.Name, func() { showRegisterDialog(*a) })
		open.Icon = theme.ListIcon()
		items = append(items, open)
	}
	items = append(items, fyne.NewMenuItemSeparator(), del)
	return items
}

func duplicateTxn(t Transaction) {
	cp := Transaction{Date: t.Date, Payee: t.Payee, Memo: t.Memo}
	cp.Posts = append([]Posting(nil), t.Posts...)
	store.AddTransaction(cp)
}

func labeledControl(label string, w fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(txt(label, colTextDim, 10, true), w)
}

func kindColor(k string) color.Color {
	switch k {
	case "Income":
		return colPositive
	case "Expense":
		return colNegative
	case "Transfer":
		return colPrimary
	default:
		return colTextDim
	}
}

func distinctMonths() []string {
	set := map[string]bool{}
	for _, t := range store.Transactions() {
		set[fmtSerialMonth(t.Date)] = true
	}
	var out []string
	for m := range set {
		out = append(out, m)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out
}

func filteredTransactions() []Transaction {
	var out []Transaction
	q := strings.ToLower(filterSearch)
	for _, t := range store.Transactions() {
		v := deriveView(t)
		if filterMonth != "All months" && fmtSerialMonth(t.Date) != filterMonth {
			continue
		}
		if filterType != "All types" && v.kind != filterType {
			continue
		}
		if filterAccount != "All accounts" && v.moneyName != filterAccount &&
			(v.kind != "Transfer" || !strings.Contains(v.moneyName, filterAccount)) {
			continue
		}
		if q != "" {
			hay := strings.ToLower(t.Payee + " " + t.Memo + " " + v.catName + " " + v.moneyName)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, t)
	}
	return out
}

// TransactionForm shows the add/edit dialog. Pass "" as id to add a new entry;
// prefillMoney (an account name) preselects the money account when adding.
func TransactionForm(id, prefillMoney string) {
	var existing *Transaction
	var v txnView
	var formDialog dialog.Dialog
	if id != "" {
		existing = store.TxnByID(id)
		if existing != nil {
			v = deriveView(*existing)
		}
	}

	// When adding (not editing) and debts exist, offer "Pay debt" — picking it
	// swaps to the focused pay-installment form.
	kinds := []string{"Expense", "Income", "Transfer"}
	if id == "" && len(store.Debts()) > 0 {
		kinds = append(kinds, "Pay debt")
	}
	kind := widget.NewSelect(kinds, nil)
	date := newDateEntry(0)
	amount := newAmountEntry()
	amount.Validator = func(s string) error {
		if parseAmount(s) <= 0 {
			return errors.New("enter an amount greater than zero")
		}
		return nil
	}
	payee := newCompletionEntry(store.Payees())
	memo := widget.NewEntry()

	moneyNames := namesOf(store.MoneyAccounts())
	assetNames := namesOf(store.AssetAccounts())
	expenseNames := namesOf(store.ExpenseAccounts())
	incomeNames := namesOf(store.IncomeAccounts())

	// Two account pickers; their options change with the chosen kind.
	// acctA = money side (paid-from / deposit-to / transfer-from)
	// acctB = category side (expense / income / transfer-to)
	acctA := widget.NewSelect(nil, nil)
	acctB := widget.NewSelect(nil, nil)

	configure := func(k string) {
		switch k {
		case "Expense":
			acctA.Options, acctB.Options = moneyNames, expenseNames
		case "Income":
			acctA.Options, acctB.Options = assetNames, incomeNames
		case "Transfer":
			acctA.Options, acctB.Options = moneyNames, moneyNames
		}
		acctA.Refresh()
		acctB.Refresh()
	}
	kind.OnChanged = func(k string) {
		if k == "Pay debt" {
			if formDialog != nil {
				formDialog.Hide()
			}
			payInstallmentForm()
			return
		}
		configure(k)
	}

	title := "Add Transaction"
	if existing != nil {
		title = "Edit Transaction"
		date.SetText(fmtSerialDate(existing.Date))
		amount.SetText(fmtMoneyInput(v.amount))
		payee.SetText(existing.Payee)
		memo.SetText(existing.Memo)
		k := v.kind
		if k != "Expense" && k != "Income" && k != "Transfer" {
			k = "Expense" // Opening/Split fall back to a simple editable form
		}
		kind.SetSelected(k)
		configure(k)
		switch v.kind {
		case "Expense":
			acctA.SetSelected(nameOf(v.moneyID))
			acctB.SetSelected(nameOf(v.catID))
		case "Income":
			acctA.SetSelected(nameOf(v.moneyID))
			acctB.SetSelected(nameOf(v.catID))
		case "Transfer":
			acctA.SetSelected(nameOf(v.fromID))
			acctB.SetSelected(nameOf(v.toID))
		}
	} else {
		date.SetText(fmtSerialDate(todaySerial))
		kind.SetSelected("Expense")
		configure("Expense")
		if prefillMoney != "" {
			acctA.SetSelected(prefillMoney)
		}
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Type", kind),
		widget.NewFormItem("Date (YYYY-MM-DD)", date),
		widget.NewFormItem("Amount", amount),
		widget.NewFormItem("Money account", acctA),
		widget.NewFormItem("Category / To", acctB),
		widget.NewFormItem("Payee", payee),
		widget.NewFormItem("Memo", memo),
	}

	// commit posts the current form as a transaction (add) or saves the edit,
	// returning whether it succeeded so "Save & Add another" knows to reset.
	commit := func() bool {
		serial := parseDateSerial(date.Text)
		amt := parseAmount(amount.Text)
		if serial == 0 || amt <= 0 || acctA.Selected == "" || acctB.Selected == "" {
			return false
		}
		if kind.Selected == "Transfer" && acctA.Selected == acctB.Selected {
			return false
		}
		posts := postingsFor(kind.Selected, idOf(acctA.Selected), idOf(acctB.Selected), amt)
		nt := Transaction{Date: serial, Payee: payee.Text(), Memo: memo.Text, Posts: posts}
		if existing != nil {
			return store.UpdateTransaction(existing.ID, nt)
		}
		return store.AddTransaction(nt)
	}

	// Editing keeps the plain Save/Cancel form. Adding uses a custom footer with a
	// "Save & Add another" action that keeps the dialog open for rapid entry.
	if existing != nil {
		formDialog = dialog.NewForm(title, "Save", "Cancel", items, func(ok bool) {
			if ok {
				commit()
			}
		}, win)
		formDialog.Resize(fyne.NewSize(460, 420))
		formDialog.Show()
		return
	}

	formGrid := container.New(layout.NewFormLayout())
	for _, it := range items {
		formGrid.Add(widget.NewLabel(it.Text))
		formGrid.Add(it.Widget)
	}

	// reset clears the per-entry fields but keeps type/date/money account so a run
	// of similar entries (same account, same day) only needs amount + category.
	reset := func() {
		amount.SetText("")
		payee.SetText("")
		memo.SetText("")
		acctB.ClearSelected()
		win.Canvas().Focus(amount)
	}

	var d *dialog.CustomDialog
	cancel := secondaryButton("Cancel", nil, func() { d.Hide() })
	saveAnother := secondaryButton("Save & Add another", theme.ContentAddIcon(), func() {
		if commit() {
			reset()
		}
	})
	save := primaryButton("Save", theme.ConfirmIcon(), func() {
		if commit() {
			d.Hide()
		}
	})
	content := container.NewVBox(formGrid, spacerH(8),
		container.NewHBox(layout.NewSpacer(), cancel, saveAnother, save))
	d = dialog.NewCustomWithoutButtons(title, content, win)
	formDialog = d
	d.Resize(fyne.NewSize(480, 460))
	d.Show()
}

// payInstallmentForm is the "Pay debt" path of the Add Transaction flow: pick a
// debt and one of its unpaid installments; saving posts that installment as a
// payment dated when you actually pay (today by default). The amount comes from
// the installment, and the money leaves the debt's configured pay-from account.
func payInstallmentForm() {
	debts := store.Debts()
	if len(debts) == 0 {
		showInfo("No debts", "Add a debt on the Debts screen first.")
		return
	}
	debtNames := make([]string, len(debts))
	for i, d := range debts {
		debtNames[i] = d.Name
	}

	debtSel := widget.NewSelect(debtNames, nil)
	instSel := widget.NewSelect(nil, nil)
	date := newDateEntry(todaySerial)
	amountLabel := widget.NewLabel("—")

	instID := map[string]string{} // installment label → ID
	instAmt := map[string]int{}   // installment label → amount
	reload := func(debtName string) {
		instID, instAmt = map[string]string{}, map[string]int{}
		var labels []string
		for _, d := range debts {
			if d.Name != debtName {
				continue
			}
			for _, in := range store.Installments(d.ID) {
				if in.Paid {
					continue
				}
				lab := "#" + itoa(in.Seq) + "    due " + fmtSerialDate(in.DueDate) + "    " + fmtMoney(in.Amount)
				labels = append(labels, lab)
				instID[lab] = in.ID
				instAmt[lab] = in.Amount
			}
		}
		instSel.Options = labels
		instSel.Refresh()
		if len(labels) > 0 {
			instSel.SetSelected(labels[0]) // fires OnChanged → fills the amount
		} else {
			instSel.SetSelected("")
			amountLabel.SetText("—")
		}
	}
	debtSel.OnChanged = reload
	instSel.OnChanged = func(lab string) {
		if a, ok := instAmt[lab]; ok {
			amountLabel.SetText(fmtMoney(a))
		}
	}
	debtSel.SetSelected(debtNames[0])

	items := []*widget.FormItem{
		widget.NewFormItem("Debt", debtSel),
		widget.NewFormItem("Installment", instSel),
		widget.NewFormItem("Amount", amountLabel),
		widget.NewFormItem("Date (YYYY-MM-DD)", date),
	}
	showForm("Pay debt", items, func() {
		id := instID[instSel.Selected]
		serial := parseDateSerial(date.Text)
		if id == "" || serial == 0 {
			return
		}
		store.PayInstallment(id, serial)
	})
}

func nameOf(id string) string {
	if a := store.AccountByID(id); a != nil {
		return a.Name
	}
	return ""
}

func idOf(name string) string {
	if a := store.AccountByName(name); a != nil {
		return a.ID
	}
	return ""
}
