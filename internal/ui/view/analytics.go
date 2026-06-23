package view

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// analyticsPeriod is the selected reporting window; it persists across the
// screen rebuilds that the live store triggers (mirrors filterMonth in the
// transactions view).
var analyticsPeriod = periodLast6

const (
	periodThisMonth = "This month"
	periodLast3     = "Last 3 months"
	periodLast6     = "Last 6 months"
	periodLast12    = "Last 12 months"
	periodYTD       = "Year to date"
)

// analyticsRange turns the selected period into an inclusive [start, end] serial
// range ending today.
func analyticsRange(sel string) (int, int) {
	end := todaySerial
	now := serialToTime(end)
	firstThis := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	switch sel {
	case periodThisMonth:
		return timeToSerial(firstThis), end
	case periodLast3:
		return timeToSerial(firstThis.AddDate(0, -2, 0)), end
	case periodLast12:
		return timeToSerial(firstThis.AddDate(0, -11, 0)), end
	case periodYTD:
		return timeToSerial(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)), end
	default: // periodLast6
		return timeToSerial(firstThis.AddDate(0, -5, 0)), end
	}
}

// ScreenAnalytics is the read-only insights dashboard: KPIs, cash-flow trend,
// net-worth trend and spending by category over the chosen period.
func ScreenAnalytics() fyne.CanvasObject {
	sel := widget.NewSelect(
		[]string{periodThisMonth, periodLast3, periodLast6, periodLast12, periodYTD},
		nil,
	)
	sel.SetSelected(analyticsPeriod)
	// Assign OnChanged only after SetSelected, so seeding the value doesn't
	// trigger a render → rebuild → SetSelected recursion.
	sel.OnChanged = func(v string) { analyticsPeriod = v; render() }

	start, end := analyticsRange(analyticsPeriod)
	bar := appBar("Analytics", periodSubtitle(start, end), sel)

	body := container.NewVBox(
		kpiRow(store.AnalyticsSummaryFor(start, end)),
		spacerH(8),
		incomeExpensePanel(store.MonthlyFlows(start, end)),
		spacerH(8),
		netWorthPanel(store.NetWorthSeries(start, end)),
		spacerH(8),
		spendingPanel(store.CategoryBreakdown(start, end, Expense)),
	)
	return container.NewBorder(bar, nil, nil, nil, container.NewPadded(body))
}

func periodSubtitle(start, end int) string {
	return fmtSerialDate(start) + " → " + fmtSerialDate(end) + " · all figures derived from the journal"
}

// monthYear formats a month-start serial as e.g. "Mar 2026" for tooltips.
func monthYear(serial int) string { return serialToTime(serial).Format("Jan 2006") }

// ---- KPI cards ----

func kpiRow(s AnalyticsSummary) fyne.CanvasObject {
	deltaArrow := "▲"
	if s.NetWorthDelta < 0 {
		deltaArrow = "▼"
	}
	netWorth := statCard("NET WORTH", fmtMoney(s.NetWorth), moneyColor(s.NetWorth),
		deltaArrow+" "+fmtMoney(s.NetWorthDelta)+" this period")
	income := statCard("INCOME", fmtMoney(s.Income), colPositive, "over "+itoa(s.Months)+" months")
	expense := statCard("EXPENSES", fmtMoney(s.Expense), colNegative, "avg "+fmtMoney(s.AvgMonthlySpend)+"/mo")
	saved := statCard("NET SAVED", fmtMoney(s.Net), moneyColor(s.Net), "income − expenses")

	rateCol := colPositive
	if s.SavingsRate < 0 {
		rateCol = colNegative
	}
	rate := statCard("SAVINGS RATE", fmt.Sprintf("%.0f%%", s.SavingsRate*100), rateCol, "of income kept")

	return container.NewGridWithColumns(5, netWorth, income, expense, saved, rate)
}

// ---- income vs expense ----

func incomeExpensePanel(flows []MonthFlow) fyne.CanvasObject {
	if len(flows) == 0 {
		return panel(emptyState("No activity in this period"))
	}
	incomes := make([]int, len(flows))
	expenses := make([]int, len(flows))
	tips := make([]string, len(flows))
	labels := make([]fyne.CanvasObject, len(flows))
	for i, f := range flows {
		incomes[i], expenses[i] = f.Income, f.Expense
		tips[i] = fmt.Sprintf("%s — Income %s · Expenses %s · Net %s",
			monthYear(f.MonthStart), fmtMoney(f.Income), fmtMoney(f.Expense), fmtMoney(f.Net()))
		labels[i] = container.NewCenter(txt(f.Label, colTextDim, 10, false))
	}
	header := container.NewBorder(nil, nil,
		sectionTitle("Income vs Expenses"),
		container.NewHBox(legendSwatch("Income", colPositive), spacerW(10), legendSwatch("Expenses", colNegative)),
	)
	chart := barPairChart(incomes, expenses, colPositive, colNegative, tips, fmtMoneyShort)
	return panel(container.NewVBox(
		header, spacerH(8), chart, spacerH(4),
		axisLabels(labels),
	))
}

// ---- net worth trend ----

func netWorthPanel(points []NetWorthPoint) fyne.CanvasObject {
	if len(points) == 0 {
		return panel(emptyState("No history in this period"))
	}
	vals := make([]int, len(points))
	tips := make([]string, len(points))
	labels := make([]fyne.CanvasObject, len(points))
	for i, p := range points {
		vals[i] = p.NetWorth
		tips[i] = monthYear(p.MonthStart) + " — Net worth " + fmtMoney(p.NetWorth)
		labels[i] = container.NewCenter(txt(p.Label, colTextDim, 10, false))
	}
	header := container.NewBorder(nil, nil,
		sectionTitle("Net Worth Over Time"),
		txt(fmtMoney(vals[len(vals)-1]), moneyColor(vals[len(vals)-1]), 13, true),
	)
	return panel(container.NewVBox(
		header, spacerH(8), lineChart(vals, colPrimary, tips, fmtMoneyShort), spacerH(4),
		axisLabels(labels),
	))
}

// ---- spending by category ----

func spendingPanel(cats []CategorySlice) fyne.CanvasObject {
	if len(cats) == 0 {
		return panel(container.NewVBox(sectionTitle("Spending by Category"), spacerH(8), emptyState("No spending in this period")))
	}
	top := cats[0].Total
	rows := make([]fyne.CanvasObject, 0, len(cats)+2)
	rows = append(rows, sectionTitle("Spending by Category"), spacerH(8))
	for _, c := range cats {
		ratio := 0.0
		if top > 0 {
			ratio = float64(c.Total) / float64(top)
		}
		label := fmt.Sprintf("%s  ·  %.0f%%", c.Account.Name, c.Pct)
		rows = append(rows, hbar(label, fmtMoney(c.Total), ratio, colPrimary), spacerH(6))
	}
	return panel(container.NewVBox(rows...))
}

// axisLabels lays the X-axis labels out under the plot, inset by the chart's
// Y-axis gutter so each label sits beneath its bar/point.
func axisLabels(labels []fyne.CanvasObject) fyne.CanvasObject {
	grid := container.NewGridWithColumns(len(labels), labels...)
	return container.NewBorder(nil, nil, spacerW(chartGutter), nil, grid)
}

// legendSwatch is a small colored square with a label, for chart legends.
func legendSwatch(label string, col color.Color) fyne.CanvasObject {
	sq := canvas.NewRectangle(col)
	sq.SetMinSize(fyne.NewSize(10, 10))
	return container.NewHBox(container.NewCenter(sq), spacerW(4), txt(label, colTextDim, 11, false))
}
