package view

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// reportsPeriod is the selected window for the period-based statements (income
// statement, cash flow); the balance sheet is always "as of" the period end.
var reportsPeriod = periodLast6

// ScreenReports presents the three financial statements as tabs. Everything is
// read-only and derived from the journal.
func ScreenReports() fyne.CanvasObject {
	sel := widget.NewSelect(
		[]string{periodThisMonth, periodLast3, periodLast6, periodLast12, periodYTD}, nil)
	sel.SetSelected(reportsPeriod)
	sel.OnChanged = func(v string) { reportsPeriod = v; render() }

	start, end := analyticsRange(reportsPeriod)
	bar := appBar("Reports", "Financial statements · "+fmtSerialDate(start)+" → "+fmtSerialDate(end), sel)

	tabs := container.NewAppTabs(
		container.NewTabItem("Income Statement", incomeStatementTab(store.IncomeStatementFor(start, end))),
		container.NewTabItem("Balance Sheet", balanceSheetTab(store.BalanceSheetAsOf(end))),
		container.NewTabItem("Cash Flow", cashFlowTab(store.CashFlowFor(start, end))),
	)
	return container.NewBorder(bar, nil, nil, nil, container.NewPadded(tabs))
}

// ---- shared rendering ----

func stmtRow(name string, amount int, bold bool, col color.Color) fyne.CanvasObject {
	return container.NewBorder(nil, nil,
		txt(name, colText, 12.5, bold),
		alignRight(mono(fmtMoney(amount), col, 12.5, bold)), nil)
}

func stmtSection(title string, lines []StmtLine, totalLabel string, total int) fyne.CanvasObject {
	rows := []fyne.CanvasObject{sectionTitle(title), spacerH(4)}
	if len(lines) == 0 {
		rows = append(rows, txt("—", colTextDim, 12, false))
	}
	for _, l := range lines {
		rows = append(rows, stmtRow(l.Name, l.Amount, false, colText))
	}
	rows = append(rows, spacerH(2), divider(), stmtRow(totalLabel, total, true, colText))
	return container.NewVBox(rows...)
}

// ---- income statement ----

func incomeStatementTab(is IncomeStatement) fyne.CanvasObject {
	net := container.NewBorder(nil, nil,
		txt("Net Income", colText, 14, true),
		alignRight(mono(fmtMoney(is.NetIncome), moneyColor(is.NetIncome), 14, true)), nil)
	return container.NewPadded(panel(container.NewVBox(
		stmtSection("Income", is.Income, "Total Income", is.TotalIncome),
		spacerH(14),
		stmtSection("Expenses", is.Expenses, "Total Expenses", is.TotalExpenses),
		spacerH(14), divider(), spacerH(6), net,
	)))
}

// ---- balance sheet ----

func balanceSheetTab(bs BalanceSheet) fyne.CanvasObject {
	return container.NewPadded(panel(container.NewVBox(
		txt("As of "+fmtSerialDate(bs.AsOf), colTextDim, 11, false), spacerH(8),
		stmtSection("Assets", bs.Assets, "Total Assets", bs.TotalAssets),
		spacerH(14),
		stmtSection("Liabilities", bs.Liabilities, "Total Liabilities", bs.TotalLiabilities),
		spacerH(14),
		stmtSection("Equity", bs.Equity, "Total Equity", bs.TotalEquity),
	)))
}

// ---- cash flow ----

func cashFlowTab(cf CashFlow) fyne.CanvasObject {
	tbl := tableSpec{
		Headers: []string{"Cash account", "Opening", "Closing", "Change"},
		Weights: []float32{2, 1, 1, 1},
		Aligns:  []fyne.TextAlign{fyne.TextAlignLeading, fyne.TextAlignTrailing, fyne.TextAlignTrailing, fyne.TextAlignTrailing},
	}
	for _, a := range cf.Accounts {
		tbl.Rows = append(tbl.Rows, []cell{
			tc(a.Name),
			tcc(fmtMoney(a.Opening), colTextDim),
			tcc(fmtMoney(a.Closing), colTextDim),
			tcc(fmtMoney(a.Change), moneyColor(a.Change)),
		})
	}
	tbl.Rows = append(tbl.Rows, []cell{
		tcb("Total cash", colText),
		tcb(fmtMoney(cf.OpeningCash), colText),
		tcb(fmtMoney(cf.ClosingCash), colText),
		tcb(fmtMoney(cf.NetChange), moneyColor(cf.NetChange)),
	})

	context := container.NewGridWithColumns(2,
		statCard("INCOME (period)", fmtMoney(cf.Income), colPositive, ""),
		statCard("EXPENSES (period)", fmtMoney(cf.Expenses), colNegative, ""),
	)
	return container.NewPadded(container.NewVBox(
		context, spacerH(8),
		panel(container.NewVBox(sectionTitle("Change in cash"), spacerH(8), tbl.Build())),
		spacerH(6),
		txt("Cash covers your asset accounts; a full operating/investing/financing breakdown isn't shown (Financy doesn't classify transactions).",
			colTextDim, 10.5, false),
	))
}
