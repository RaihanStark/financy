package view

import (
	"testing"

	"fyne.io/fyne/v2/container"
)

// The reports screen renders the three statement tabs.
func TestScreenReportsRendersTabs(t *testing.T) {
	_, w := newViewTest(t)
	screen := mount(w, ScreenReports())
	assertText(t, screen, "Reports", "Income Statement", "Balance Sheet", "Cash Flow")
}

// The income statement tab shows totals and net income derived from the journal.
func TestIncomeStatementTab(t *testing.T) {
	s, w := newViewTest(t)
	start, end := analyticsRange(reportsPeriod)
	is := s.IncomeStatementFor(start, end)

	tab := mount(w, container.NewPadded(incomeStatementTab(is)))
	assertText(t, tab,
		"Income", "Total Income", "Expenses", "Total Expenses", "Net Income",
		fmtMoney(is.TotalIncome), fmtMoney(is.NetIncome),
	)
}

// The balance sheet tab always balances: Assets = Liabilities + Equity.
func TestBalanceSheetTabBalances(t *testing.T) {
	s, w := newViewTest(t)
	_, end := analyticsRange(reportsPeriod)
	bs := s.BalanceSheetAsOf(end)

	tab := mount(w, container.NewPadded(balanceSheetTab(bs)))
	assertText(t, tab, "Assets", "Liabilities", "Equity", "Total Assets")

	if bs.TotalAssets != bs.TotalLiabilities+bs.TotalEquity {
		t.Fatalf("balance sheet does not balance: assets %d != liabilities %d + equity %d",
			bs.TotalAssets, bs.TotalLiabilities, bs.TotalEquity)
	}
}

// The cash-flow tab lists each cash account and the period income/expense context.
func TestCashFlowTab(t *testing.T) {
	s, w := newViewTest(t)
	start, end := analyticsRange(reportsPeriod)
	cf := s.CashFlowFor(start, end)

	tab := mount(w, container.NewPadded(cashFlowTab(cf)))
	assertText(t, tab,
		"Cash account", "Opening", "Closing", "Change",
		"Total cash", "INCOME (period)", "EXPENSES (period)",
	)
}
