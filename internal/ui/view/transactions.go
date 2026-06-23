package view

import (
	"errors"
	"image/color"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func ScreenTransactions() fyne.CanvasObject {
	bar := appBar("Transactions", "Double-entry journal — every entry balances to zero",
		secondaryButton("Import CSV", theme.DownloadIcon(), func() { ImportCSV() }),
		primaryButton("Add Transaction", theme.ContentAddIcon(), func() { TransactionForm("", "") }),
	)

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

	body := container.NewVBox(filters, spacerH(4), summary, spacerH(4),
		panel(container.NewVBox(sectionTitle("Journal"), spacerH(4), txnLedger(rows))))
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
	payee := t.Payee
	if payee == "" {
		payee = v.catName
	}

	center := container.NewVBox(
		txt(payee, colText, 13, true),
		txt(sub, colTextDim, 10.5, false),
	)
	right := container.NewVBox(
		alignRight(mono(amtStr, amtCol, 14, true)),
		alignRight(txt(v.kind, colTextDim, 9.5, false)),
	)
	inner := container.NewBorder(nil, nil,
		container.NewHBox(typeChip(v.kind), spacerW(8)),
		right, center)

	row := newTappableRow(container.New(padCell(7, 10), inner), colSurface,
		func() { TransactionForm(t.ID, "") })
	row.SetOnSecondary(func(pos fyne.Position) { showContextMenu(pos, txnMenuItems(t)...) })
	return row
}

// typeChip is the small colored square that flags the transaction kind.
func typeChip(kind string) fyne.CanvasObject {
	col := kindColor(kind)
	glyph := map[string]string{"Income": "+", "Expense": "−", "Transfer": "⇄", "Opening": "•", "Split": "≡"}[kind]
	if glyph == "" {
		glyph = "•"
	}
	bg := canvas.NewRectangle(withAlpha(col, 0x22))
	bg.StrokeColor = withAlpha(col, 0x99)
	bg.StrokeWidth = 1
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
			!(v.kind == "Transfer" && strings.Contains(v.moneyName, filterAccount)) {
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
	if id != "" {
		existing = store.TxnByID(id)
		if existing != nil {
			v = deriveView(*existing)
		}
	}

	kind := widget.NewSelect([]string{"Expense", "Income", "Transfer"}, nil)
	date := newDateEntry(0)
	amount := newAmountEntry()
	amount.Validator = func(s string) error {
		if parseAmount(s) <= 0 {
			return errors.New("enter an amount greater than zero")
		}
		return nil
	}
	payee := widget.NewEntry()
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
	kind.OnChanged = func(k string) { configure(k) }

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

	showForm(title, items, func() {
		serial := parseDateSerial(date.Text)
		amt := parseAmount(amount.Text)
		if serial == 0 || amt <= 0 || acctA.Selected == "" || acctB.Selected == "" {
			return
		}
		if kind.Selected == "Transfer" && acctA.Selected == acctB.Selected {
			return
		}
		posts := postingsFor(kind.Selected, idOf(acctA.Selected), idOf(acctB.Selected), amt)
		nt := Transaction{Date: serial, Payee: payee.Text, Memo: memo.Text, Posts: posts}
		if existing != nil {
			store.UpdateTransaction(existing.ID, nt)
		} else {
			store.AddTransaction(nt)
		}
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
