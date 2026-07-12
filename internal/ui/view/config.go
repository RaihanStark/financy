package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// OpenConfig shows the combined Preferences dialog with three tabs.
// tab: 0 = Configuration, 1 = Categories, 2 = Data Summary.
func OpenConfig(tab int) {
	if win == nil {
		return
	}

	cfgItem := container.NewTabItem("Configuration", widget.NewLabel(""))
	catItem := container.NewTabItem("Categories", widget.NewLabel(""))
	dataItem := container.NewTabItem("Data Summary", widget.NewLabel(""))

	tabs := container.NewAppTabs(cfgItem, catItem, dataItem)

	// Rebuild only the data-driven tabs (keeps unsaved Configuration edits intact).
	refreshData := func() {
		dataItem.Content = dataSummaryBody()
		tabs.Refresh()
	}
	var refreshCats func()
	refreshCats = func() {
		catItem.Content = categoriesBody(func() { refreshCats(); refreshData() })
		tabs.Refresh()
	}

	// Saving currency affects amounts shown on every tab, so refresh both.
	cfgItem.Content = configBody(func() { refreshCats(); refreshData() })
	refreshCats()
	refreshData()

	if tab >= 0 && tab < 3 {
		tabs.SelectIndex(tab)
	}

	content := container.NewGridWrap(fyne.NewSize(760, 540), tabs)
	d := newModalWithClose("Preferences", "Close", content)
	d.Show()
}

// ---- Configuration tab ----

func configBody(onSaved func()) fyne.CanvasObject {
	currency := widget.NewSelect([]string{"Rp", "$", "€", "£"}, nil)
	currency.SetSelected(store.Currency())

	const fmtDefault = "Currency default"
	numFmt := widget.NewSelect(append([]string{fmtDefault}, numberFormatStyles()...), nil)
	if store.NumberFormat() == "" {
		numFmt.SetSelected(fmtDefault)
	} else {
		numFmt.SetSelected(store.NumberFormat())
	}

	form := widget.NewForm(
		widget.NewFormItem("Currency", currency),
		widget.NewFormItem("Number format", numFmt),
	)

	saved := txt("", colPositive, 11.5, true)
	save := primaryButton("Save Changes", theme.DocumentSaveIcon(), func() {
		// Owner and year are not user-facing; preserve them.
		store.SetSettings(store.Owner(), currency.Selected, store.Year())
		style := numFmt.Selected
		if style == fmtDefault {
			style = ""
		}
		store.SetNumberFormat(style)
		saved.Text = "Saved."
		saved.Refresh()
		if onSaved != nil {
			onSaved()
		}
	})

	return container.NewPadded(container.NewVBox(
		sectionTitle("Currency & Format"),
		spacerH(6),
		form,
		spacerH(8),
		container.NewHBox(save, saved),
		spacerH(12),
		txt("The currency sets the symbol and decimals; the number format controls the",
			colTextDim, 11, false),
		txt("thousands and decimal separators (e.g. 1,234.56 vs 1.234,56).",
			colTextDim, 11, false),
	))
}

// ---- Categories tab ----

func categoriesBody(refresh func()) fyne.CanvasObject {
	add := primaryButton("Add Category", theme.ContentAddIcon(), func() { CategoryForm(nil, refresh) })

	head := container.NewBorder(nil, nil,
		txt("Labels used to classify transactions", colTextDim, 11.5, false),
		add)

	list := container.NewVBox(
		categorySection("INCOME", store.IncomeAccounts(), refresh),
		spacerH(8),
		categorySection("EXPENSES", store.ExpenseAccounts(), refresh),
	)

	return container.NewBorder(
		container.NewVBox(container.NewPadded(head), divider()),
		nil, nil, nil,
		container.NewScroll(container.NewPadded(list)),
	)
}

func categorySection(title string, accts []Account, refresh func()) fyne.CanvasObject {
	rows := container.NewVBox(txt(title, colTextDim, 10.5, true), spacerH(2), divider())
	if len(accts) == 0 {
		rows.Add(container.New(padCell(8, 6), emptyState("None yet")))
	}
	for _, a := range accts {
		a := a
		editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() { CategoryForm(&a, refresh) })
		editBtn.Importance = widget.LowImportance
		delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() { deleteCategory(a, refresh) })
		delBtn.Importance = widget.LowImportance

		left := container.NewVBox(txt(a.Name, colText, 12.5, true), txt(orDash(a.Notes), colTextDim, 10, false))
		rows.Add(container.New(padCell(5, 6), container.NewBorder(nil, nil, left, container.NewHBox(editBtn, delBtn))))
		rows.Add(divider())
	}
	return rows
}

func deleteCategory(a Account, refresh func()) {
	if store.TxnCountForAccount(a.ID) > 0 {
		showInfo("Cannot delete", "“"+a.Name+"” is used by transactions. Remove them first.")
		return
	}
	confirmDelete("category “"+a.Name+"”", func() {
		store.DeleteAccount(a.ID)
		if refresh != nil {
			refresh()
		}
	})
}

// CategoryForm adds or edits an income/expense category. Pass nil to add.
func CategoryForm(existing *Account, onDone func()) {
	name := widget.NewEntry()
	typ := widget.NewSelect([]string{"Expense", "Income"}, nil)
	notes := widget.NewEntry()

	title := "Add Category"
	if existing != nil {
		title = "Edit Category"
		name.SetText(existing.Name)
		typ.SetSelected(string(existing.Type))
		notes.SetText(existing.Notes)
	} else {
		typ.SetSelected("Expense")
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Type", typ),
		widget.NewFormItem("Notes", notes),
	}
	showForm(title, items, func() {
		if name.Text == "" || typ.Selected == "" {
			return
		}
		if existing != nil {
			store.UpdateAccount(existing.ID, Account{Name: name.Text, Type: AcctType(typ.Selected), Notes: notes.Text})
		} else {
			store.AddAccount(Account{Name: name.Text, Type: AcctType(typ.Selected), Notes: notes.Text})
		}
		if onDone != nil {
			onDone()
		}
	})
}

// ---- Data Summary tab ----

func dataSummaryBody() fyne.CanvasObject {
	rows := container.NewVBox(
		sectionTitle("Data Summary"), spacerH(4), divider(),
		detailField("Asset accounts", itoa(len(store.AssetAccounts())), colText),
		detailField("Liability accounts", itoa(len(store.LiabilityAccounts())), colText),
		detailField("Income categories", itoa(len(store.IncomeAccounts())), colText),
		detailField("Expense categories", itoa(len(store.ExpenseAccounts())), colText),
		detailField("Transactions", itoa(len(store.Transactions())), colText),
		spacerH(6), divider(), spacerH(2),
		detailField("Total assets", fmtMoney(store.TotalAssets()), colPositive),
		detailField("Total liabilities", fmtMoney(store.TotalLiabilities()), colNegative),
		detailField("Net worth", fmtMoney(store.NetWorth()), moneyColor(store.NetWorth())),
		detailField("Income (YTD)", fmtMoney(store.IncomeTotal()), colPositive),
		detailField("Expenses (YTD)", fmtMoney(store.ExpenseTotal()), colNegative),
		spacerH(10), divider(), spacerH(4),
		txt("Financy v"+version, colTextDim, 10.5, false),
	)
	return container.NewPadded(rows)
}
