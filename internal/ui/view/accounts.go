package view

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func ScreenAccounts() fyne.CanvasObject {
	bar := appBar("Accounts", "Click a row to open its register · right-click for actions",
		primaryButton("Add Account", theme.ContentAddIcon(), func() { AccountForm(nil) }),
	)

	body := container.NewVBox(
		netWorthHero(),
		spacerH(6),
		upcomingBills(),
		accountGroup("Assets", store.AssetAccounts(), store.TotalAssets(), false),
		spacerH(6),
		accountGroup("Liabilities", store.LiabilityAccounts(), store.TotalLiabilities(), true),
	)
	return container.NewBorder(bar, nil, nil, nil, container.NewPadded(body))
}

// upcomingBills surfaces the recurring schedule on the overview: a due alert
// when something needs review, then the next two weeks of occurrences. Hidden
// until the user has recurring templates at all.
func upcomingBills() fyne.CanvasObject {
	if len(store.Recurrings()) == 0 {
		return spacerH(0)
	}
	sum := store.RecurringSummaryFor(todaySerial)

	box := container.NewVBox()
	if sum.DueCount > 0 {
		msg := itoa(sum.DueCount) + " recurring entr" + plural(sum.DueCount, "y", "ies") + " due · " + fmtMoney(sum.DueTotal)
		box.Add(alertBanner(theme.WarningIcon(), msg, colWarning,
			secondaryButton("Review & Post", theme.MediaPlayIcon(), postDueNow)))
		box.Add(spacerH(6))
	}

	netSub := "net " + fmtMoney(sum.MonthlyNet) + "/mo"
	head := container.NewBorder(nil, nil,
		sectionTitle("Upcoming"),
		txt("next 14 days · "+netSub, colTextDim, 11, false))
	rows := []fyne.CanvasObject{head, spacerH(6)}

	occ := store.UpcomingRecurring(todaySerial, todaySerial+14)
	if len(occ) == 0 {
		rows = append(rows, txt("Nothing due in the next 14 days.", colTextDim, 12, false))
	}
	for i, o := range occ {
		if i == 5 {
			rows = append(rows, container.New(padCell(4, 12),
				txt("+ "+itoa(len(occ)-5)+" more on the Recurring screen", colTextDim, 11, false)))
			break
		}
		rows = append(rows, upcomingBillRow(o))
	}

	more := newTappableRow(container.New(padCell(5, 12),
		txt("View all recurring →", colPrimary, 11.5, true)), colSurface,
		func() { nav("recurring") })
	rows = append(rows, spacerH(2), more)

	box.Add(panel(container.New(padCell(10, 12), container.NewVBox(rows...))))
	box.Add(spacerH(6))
	return box
}

// upcomingBillRow is a compact occurrence line on the overview; tapping goes to
// the Recurring screen.
func upcomingBillRow(o Occurrence) fyne.CanvasObject {
	amtCol := colNegative
	switch o.Kind {
	case "Income":
		amtCol = colPositive
	case "Transfer":
		amtCol = colText
	}
	title := o.Payee
	if title == "" {
		title = o.Kind
	}
	state, stateCol := dueLabel(o.Date)
	line := container.NewBorder(nil, nil,
		container.NewHBox(
			mono(fmtSerialDate(o.Date), colTextDim, 11, false),
			spacerW(8),
			txt(title, colText, 12.5, false),
		),
		container.NewHBox(
			txt(state, stateCol, 10.5, o.Due),
			spacerW(10),
			txt(amountLabel(o.Kind, o.Amount), amtCol, 12.5, true),
		))

	var fill color.Color = colSurface
	if o.Due {
		fill = withAlpha(colWarning, 0x14)
	}
	row := newTappableRow(container.New(padCell(5, 12), line), fill, func() { nav("recurring") })
	row.SetBordered()
	return row
}

// netWorthHero is the prominent summary banner at the top.
func netWorthHero() fyne.CanvasObject {
	nw := store.NetWorth()
	netBlock := container.NewVBox(
		txt("NET WORTH", colTextDim, 11, true),
		spacerH(4),
		txt(fmtMoney(nw), moneyColor(nw), 30, true),
		spacerH(3),
		txt("Assets − Liabilities", colTextDim, 11, false),
	)
	assetsBlock := container.NewVBox(
		txt("TOTAL ASSETS", colTextDim, 10.5, true),
		spacerH(4),
		txt(fmtMoney(store.TotalAssets()), colPositive, 18, true),
		spacerH(3),
		txt(itoa(len(store.AssetAccounts()))+" accounts", colTextDim, 10, false),
	)
	liabBlock := container.NewVBox(
		txt("TOTAL LIABILITIES", colTextDim, 10.5, true),
		spacerH(4),
		txt(fmtMoney(store.TotalLiabilities()), colNegative, 18, true),
		spacerH(3),
		txt(itoa(len(store.LiabilityAccounts()))+" accounts", colTextDim, 10, false),
	)
	// Center the shorter side blocks against the taller net-worth block so the
	// row reads as one aligned band rather than three floating columns.
	vCenter := func(o fyne.CanvasObject) fyne.CanvasObject {
		return container.NewVBox(spacerV(), o, spacerV())
	}
	right := container.NewGridWithColumns(2, vCenter(assetsBlock), vCenter(liabBlock))
	inner := container.New(&columnsLayout{Weights: []float32{1.1, 2}, Gap: 24}, netBlock, right)
	return panel(container.New(padCell(10, 12), inner))
}

// accountGroup renders a titled table panel: a header with the subtotal, a
// column header row, and one tappable line per account (mirroring the Debts
// table so the two screens read the same).
func accountGroup(title string, accts []Account, subtotal int, liability bool) fyne.CanvasObject {
	subCol := colPositive
	if liability {
		subCol = colNegative
	}
	head := container.NewBorder(nil, nil,
		txt(title, colText, 13.5, true),
		container.NewHBox(
			txt(itoa(len(accts))+" account"+plural(len(accts), "", "s"), colTextDim, 11, false),
			spacerW(10),
			txt(fmtMoney(subtotal), subCol, 13.5, true),
		),
	)

	if len(accts) == 0 {
		return panel(container.NewVBox(container.New(padCell(4, 8), head), spacerH(4),
			container.New(padCell(16, 8), emptyState("No "+title+" yet"))))
	}

	cols := accountCols()
	h := func(s string) fyne.CanvasObject { return txt(s, colTextDim, 10.5, true) }
	header := container.New(cols,
		h("Account"), h("Share"),
		alignRight(txt("Txns", colTextDim, 10.5, true)),
		alignRight(txt("Balance", colTextDim, 10.5, true)),
		h(""))

	table := container.NewVBox(
		container.New(padCell(4, 8), head),
		spacerH(6),
		container.New(padCell(2, 8), header),
		divider(),
	)
	for i, a := range accts {
		table.Add(accountRow(a, liability, subtotal, i))
	}
	return panel(container.New(padCell(6, 6), table))
}

// accountCols is the shared column layout for the account header and rows.
func accountCols() *columnsLayout {
	return &columnsLayout{Weights: []float32{2.9, 1.9, 0.7, 1.5, 0.4}, Gap: 10}
}

// accountRow is one account as a table line: name + institution, its share of
// the group's total, transaction count, balance, and the ⋮ menu. Tapping the
// row opens the register; right-click opens the same menu.
func accountRow(a Account, liability bool, subtotal, idx int) fyne.CanvasObject {
	amtCol := colPositive
	if liability {
		amtCol = colNegative
	}
	bal := store.DisplayBalance(a)
	entries := store.TxnCountForAccount(a.ID)

	nameCell := container.NewHBox(container.NewCenter(txt(a.Name, colText, 12.5, true)))
	if a.Institution != "" {
		nameCell.Add(container.NewCenter(txt(" · "+a.Institution, colTextDim, 11.5, false)))
	}
	if a.OffBudget {
		nameCell.Add(spacerW(6))
		nameCell.Add(container.NewCenter(miniBadge("Off-budget")))
	}

	// Share of the group's total — the at-a-glance composition the old card
	// grid never offered. Indigo on purpose: it's a proportion, not a verdict.
	ratio, pct := 0.0, "—"
	if subtotal > 0 && bal > 0 {
		ratio = float64(bal) / float64(subtotal)
		pct = itoa(int(ratio*100+0.5)) + "%"
	}
	bar := container.NewVBox(spacerV(), container.New(padCell(0, 4), progressBar(ratio, colPrimary)), spacerV())
	shareCell := container.NewBorder(nil, nil, nil,
		container.NewCenter(container.NewGridWrap(fyne.NewSize(38, 16), alignRight(mono(pct, colTextDim, 11, false)))),
		bar,
	)

	menuBtn := widget.NewButtonWithIcon("", theme.MoreVerticalIcon(), nil)
	menuBtn.Importance = widget.LowImportance
	menuBtn.OnTapped = func() {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn)
		pos.Y += menuBtn.Size().Height
		showContextMenu(pos, accountMenuItems(a)...)
	}

	cells := container.New(accountCols(),
		nameCell,
		shareCell,
		alignRight(mono(itoa(entries), colTextDim, 12, false)),
		alignRight(mono(fmtMoney(bal), amtCol, 12.5, false)),
		container.NewCenter(menuBtn),
	)

	var fill = colSurface
	if idx%2 == 1 {
		fill = colAltRow
	}
	row := newTappableRow(container.New(padCell(6, 8), cells), fill,
		func() { showRegisterDialog(a) })
	row.SetOnSecondary(func(pos fyne.Position) { showContextMenu(pos, accountMenuItems(a)...) })
	return row
}

// miniBadge is a compact inline chip that doesn't inflate a table row's
// height the way the full-size badge would.
func miniBadge(label string) fyne.CanvasObject {
	bg := canvas.NewRectangle(withAlpha(colTextDim, 0x1f))
	bg.CornerRadius = 6
	return container.NewStack(bg, container.New(padCell(1, 6), txt(label, colTextDim, 9.5, true)))
}

// spacerV is an expanding vertical filler used to center fixed-height content
// inside a stretched table cell.
func spacerV() fyne.CanvasObject { return layout.NewSpacer() }

// accountMenuItems is the shared right-click / ⋮ menu for an account.
func accountMenuItems(a Account) []*fyne.MenuItem {
	newTxn := fyne.NewMenuItem("New Transaction…", func() { TransactionForm("", a.Name) })
	newTxn.Icon = theme.ContentAddIcon()
	viewReg := fyne.NewMenuItem("View Register", func() { showRegisterDialog(a) })
	viewReg.Icon = theme.ListIcon()
	reconcile := fyne.NewMenuItem("Reconcile…", func() { reconcileAccount(a) })
	reconcile.Icon = theme.ViewRefreshIcon()
	edit := fyne.NewMenuItem("Edit Account…", func() { AccountForm(&a) })
	edit.Icon = theme.DocumentCreateIcon()
	del := fyne.NewMenuItem("Delete Account", func() { deleteAccount(a) })
	del.Icon = theme.DeleteIcon()

	return []*fyne.MenuItem{newTxn, viewReg, reconcile, fyne.NewMenuItemSeparator(), edit, del}
}

func deleteAccount(a Account) {
	if store.TxnCountForAccount(a.ID) > 0 {
		showInfo("Cannot delete", "“"+a.Name+"” has transactions. Remove them first.")
		return
	}
	confirmDelete("account “"+a.Name+"”", func() { store.DeleteAccount(a.ID) })
}

// showRegisterDialog opens the running-balance register for an account.
func showRegisterDialog(a Account) {
	if win == nil {
		return
	}
	amtCol := colPositive
	if a.Type == Liability {
		amtCol = colNegative
	}
	reg := store.Register(a.ID)

	header := container.NewBorder(nil, nil,
		container.NewVBox(
			txt(a.Name, colText, 16, true),
			txt(string(a.Type)+" · "+orDash(a.Institution), colTextDim, 11, false),
		),
		container.NewVBox(
			alignRight(txt(fmtMoney(store.DisplayBalance(a)), amtCol, 18, true)),
			alignRight(txt(itoa(len(reg))+" transactions", colTextDim, 10, false)),
		),
	)

	actions := container.NewHBox(
		secondaryButton("New Transaction", theme.ContentAddIcon(), func() { TransactionForm("", a.Name) }),
		secondaryButton("Reconcile", theme.ViewRefreshIcon(), func() { reconcileAccount(a) }),
		secondaryButton("Edit Account", theme.DocumentCreateIcon(), func() { AccountForm(&a) }),
	)

	body := container.NewBorder(
		container.NewVBox(header, spacerH(6), actions, spacerH(6), divider()),
		nil, nil, nil,
		container.NewScroll(registerTable(a)),
	)

	d := newModalWithClose("Register — "+a.Name, "Close", body)
	d.SetCardSize(fyne.NewSize(820, 560))
	d.Show()
}

// registerTable builds the running-balance ledger for one account.
func registerTable(a Account) fyne.CanvasObject {
	reg := store.Register(a.ID)
	if len(reg) == 0 {
		return container.New(padCell(20, 10), emptyState("No transactions for this account"))
	}
	spec := tableSpec{
		Headers: []string{"Date", "Payee", "Contra account", "Change", "Balance"},
		Weights: []float32{1, 1.8, 1.8, 1.2, 1.3},
		Aligns: []fyne.TextAlign{
			fyne.TextAlignLeading, fyne.TextAlignLeading, fyne.TextAlignLeading,
			fyne.TextAlignTrailing, fyne.TextAlignTrailing,
		},
	}
	for _, r := range reg {
		chCol := colPositive
		chStr := "+" + fmtMoney(r.Amount)
		if r.Amount < 0 {
			chCol, chStr = colNegative, "−"+fmtMoney(-r.Amount)
		}
		spec.Rows = append(spec.Rows, []cell{
			tc(fmtSerialDate(r.Txn.Date)),
			tc(orDash(r.Txn.Payee)),
			tcc(contraNames(r.Txn, a.ID), colTextDim),
			tcc(chStr, chCol),
			tcc(fmtMoney(r.Balance), colText),
		})
	}
	return spec.Build()
}

// contraNames lists the other accounts touched by a transaction.
func contraNames(t Transaction, self string) string {
	var names []string
	for _, p := range t.Posts {
		if p.AccountID == self {
			continue
		}
		if a := store.AccountByID(p.AccountID); a != nil {
			names = append(names, a.Name)
		}
	}
	if len(names) == 0 {
		return "—"
	}
	out := names[0]
	for _, n := range names[1:] {
		out += ", " + n
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// AccountForm shows the add/edit dialog for a money account. Pass nil to add.
func AccountForm(existing *Account) {
	name := widget.NewEntry()
	inst := widget.NewEntry()
	typ := widget.NewSelect([]string{"Asset", "Liability"}, nil)
	notes := widget.NewEntry()
	opening := newAmountEntry()
	// On-budget by default; uncheck for tracking accounts (investments, mortgage)
	// whose balance shouldn't count as money you can assign in the budget.
	inBudget := widget.NewCheck("Include this account's balance in the budget", nil)
	inBudget.SetChecked(true)

	title := "Add Account"
	items := []*widget.FormItem{
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Institution", inst),
		widget.NewFormItem("Type", typ),
		widget.NewFormItem("Notes", notes),
		widget.NewFormItem("Budget", inBudget),
	}
	if existing != nil {
		title = "Edit Account"
		name.SetText(existing.Name)
		inst.SetText(existing.Institution)
		typ.SetSelected(string(existing.Type))
		notes.SetText(existing.Notes)
		inBudget.SetChecked(!existing.OffBudget)
	} else {
		typ.SetSelected("Asset")
		items = append(items, widget.NewFormItem("Opening balance", opening))
	}

	showForm(title, items, func() {
		if name.Text == "" || typ.Selected == "" {
			return
		}
		acc := Account{Name: name.Text, Institution: inst.Text, Type: AcctType(typ.Selected),
			Notes: notes.Text, OffBudget: !inBudget.Checked}
		if existing != nil {
			store.UpdateAccount(existing.ID, acc)
			return
		}
		acc.ID = slugify(name.Text)
		store.AddAccount(acc)
		if bal := parseAmount(opening.Text); bal > 0 {
			if acc.Type == Asset {
				store.AddTransaction(Transaction{Date: todaySerial, Payee: "Opening Balance",
					Posts: []Posting{p(acc.ID, bal), p("opening", -bal)}})
			} else {
				store.AddTransaction(Transaction{Date: todaySerial, Payee: "Loan Opening Balance",
					Posts: []Posting{p(acc.ID, -bal), p("opening", bal)}})
			}
		}
	})
}
