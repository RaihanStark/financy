package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func ScreenAccounts() fyne.CanvasObject {
	bar := appBar("Accounts", "Click a card to open its register · right-click for actions",
		primaryButton("Add Account", theme.ContentAddIcon(), func() { AccountForm(nil) }),
	)

	body := container.NewVBox(
		netWorthHero(),
		spacerH(6),
		accountGroup("Assets", store.AssetAccounts(), store.TotalAssets(), false),
		spacerH(6),
		accountGroup("Liabilities", store.LiabilityAccounts(), store.TotalLiabilities(), true),
	)
	return container.NewBorder(bar, nil, nil, nil, container.NewPadded(body))
}

// netWorthHero is the prominent summary banner at the top.
func netWorthHero() fyne.CanvasObject {
	nw := store.NetWorth()
	netBlock := container.NewVBox(
		txt("NET WORTH", colTextDim, 11, true),
		txt(fmtMoney(nw), moneyColor(nw), 30, true),
		txt("Assets − Liabilities", colTextDim, 11, false),
	)
	assetsBlock := container.NewVBox(
		txt("TOTAL ASSETS", colTextDim, 10.5, true),
		txt(fmtMoney(store.TotalAssets()), colPositive, 18, true),
		txt(itoa(len(store.AssetAccounts()))+" accounts", colTextDim, 10, false),
	)
	liabBlock := container.NewVBox(
		txt("TOTAL LIABILITIES", colTextDim, 10.5, true),
		txt(fmtMoney(store.TotalLiabilities()), colNegative, 18, true),
		txt(itoa(len(store.LiabilityAccounts()))+" accounts", colTextDim, 10, false),
	)
	right := container.NewGridWithColumns(2, assetsBlock, liabBlock)
	return panel(container.New(&columnsLayout{Weights: []float32{1.1, 2}, Gap: 24}, netBlock, right))
}

// accountGroup renders a titled section with a subtotal and a grid of cards.
func accountGroup(title string, accts []Account, subtotal int, liability bool) fyne.CanvasObject {
	subCol := colPositive
	if liability {
		subCol = colNegative
	}
	head := container.NewBorder(nil, nil,
		txt(title, colText, 14, true),
		container.NewHBox(txt(itoa(len(accts))+" accounts   ", colTextDim, 11, false),
			txt(fmtMoney(subtotal), subCol, 14, true)),
	)

	if len(accts) == 0 {
		return container.NewVBox(head, spacerH(4), panel(emptyState("No "+title+" yet")))
	}

	cards := make([]fyne.CanvasObject, 0, len(accts))
	for _, a := range accts {
		cards = append(cards, accountCard(a, liability))
	}
	grid := container.NewGridWithColumns(3, cards...)
	return container.NewVBox(head, spacerH(4), grid)
}

// accountCard is a single, scannable account tile with a context menu.
func accountCard(a Account, liability bool) fyne.CanvasObject {
	amtCol := colPositive
	if liability {
		amtCol = colNegative
	}
	bal := store.DisplayBalance(a)
	entries := store.TxnCountForAccount(a.ID)

	menuBtn := widget.NewButtonWithIcon("", theme.MoreVerticalIcon(), nil)
	menuBtn.Importance = widget.LowImportance
	menuBtn.OnTapped = func() {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn)
		pos.Y += menuBtn.Size().Height
		showContextMenu(pos, accountMenuItems(a)...)
	}

	top := container.NewBorder(nil, nil,
		txt(a.Name, colText, 14, true), menuBtn)
	sub := txt(string(a.Type)+" · "+orDash(a.Institution), colTextDim, 10.5, false)
	balance := txt(fmtMoney(bal), amtCol, 22, true)
	footer := txt(itoa(entries)+" transactions", colTextDim, 10, false)

	inner := container.NewVBox(top, sub, spacerH(8), balance, spacerH(2), footer)

	row := newTappableRow(container.New(padCell(10, 12), inner), colSurface,
		func() { showRegisterDialog(a) })
	row.SetBordered()
	row.SetOnSecondary(func(pos fyne.Position) { showContextMenu(pos, accountMenuItems(a)...) })
	return row
}

// accountMenuItems is the shared right-click / ⋮ menu for an account.
func accountMenuItems(a Account) []*fyne.MenuItem {
	newTxn := fyne.NewMenuItem("New Transaction…", func() { TransactionForm("", a.Name) })
	newTxn.Icon = theme.ContentAddIcon()
	viewReg := fyne.NewMenuItem("View Register", func() { showRegisterDialog(a) })
	viewReg.Icon = theme.ListIcon()
	edit := fyne.NewMenuItem("Edit Account…", func() { AccountForm(&a) })
	edit.Icon = theme.DocumentCreateIcon()
	del := fyne.NewMenuItem("Delete Account", func() { deleteAccount(a) })
	del.Icon = theme.DeleteIcon()

	return []*fyne.MenuItem{newTxn, viewReg, fyne.NewMenuItemSeparator(), edit, del}
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
		secondaryButton("Edit Account", theme.DocumentCreateIcon(), func() { AccountForm(&a) }),
	)

	body := container.NewBorder(
		container.NewVBox(header, spacerH(6), actions, spacerH(6), divider()),
		nil, nil, nil,
		container.NewScroll(registerTable(a)),
	)

	d := dialog.NewCustom("Register — "+a.Name, "Close", body, win)
	d.Resize(fyne.NewSize(820, 560))
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
	opening := widget.NewEntry()
	opening.SetPlaceHolder("0")

	title := "Add Account"
	items := []*widget.FormItem{
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Institution", inst),
		widget.NewFormItem("Type", typ),
		widget.NewFormItem("Notes", notes),
	}
	if existing != nil {
		title = "Edit Account"
		name.SetText(existing.Name)
		inst.SetText(existing.Institution)
		typ.SetSelected(string(existing.Type))
		notes.SetText(existing.Notes)
	} else {
		typ.SetSelected("Asset")
		items = append(items, widget.NewFormItem("Opening balance", opening))
	}

	showForm(title, items, func() {
		if name.Text == "" || typ.Selected == "" {
			return
		}
		acc := Account{Name: name.Text, Institution: inst.Text, Type: AcctType(typ.Selected), Notes: notes.Text}
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
