package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// strategyDialog compares the two classic payoff strategies side by side:
// snowball (smallest balance first — quick wins) vs avalanche (highest APR
// first — least interest). Enter what extra you can put toward debts each
// month; every required payment keeps being paid either way, and each cleared
// debt's payment rolls into the next target.
func strategyDialog() {
	if win == nil {
		return
	}
	if store.TotalDebtOutstanding() == 0 {
		showInfo("Debt-free", "Nothing to strategize — you owe nothing.")
		return
	}

	extra := newAmountEntry()
	extra.SetText(fmtMoneyInput(0))
	results := container.NewGridWithColumns(2)

	update := func() {
		snow, aval := store.CompareStrategies(todaySerial, parseAmount(extra.Text))
		cheaper := ""
		if snow.Feasible && aval.Feasible && aval.TotalInterest < snow.TotalInterest {
			cheaper = aval.Strategy
		} else if snow.Feasible && aval.Feasible && snow.TotalInterest < aval.TotalInterest {
			cheaper = snow.Strategy
		}
		results.Objects = []fyne.CanvasObject{
			strategyPanel(snow, "Smallest balance first — quick wins keep you motivated.", cheaper == snow.Strategy),
			strategyPanel(aval, "Highest APR first — the least total interest.", cheaper == aval.Strategy),
		}
		results.Refresh()
	}
	extra.OnChanged = func(string) { update() }
	update()

	head := container.NewBorder(nil, nil,
		txt("Extra per month, on top of all required payments:  ", colText, 12, false), nil, extra)

	var dlg *modal
	closeBtn := secondaryButton("Close", theme.CancelIcon(), func() { dlg.Hide() })
	content := container.NewBorder(
		container.NewVBox(head, spacerH(12)),
		container.NewVBox(spacerH(10), container.NewHBox(layout.NewSpacer(), closeBtn)),
		nil, nil, results,
	)
	dlg = newModal("Payoff strategies", content)
	dlg.SetCardSize(fyne.NewSize(860, 520))
	dlg.Show()
}

// strategyPanel renders one strategy's outcome: the headline numbers and the
// payoff order.
func strategyPanel(r StrategyResult, blurb string, cheapest bool) fyne.CanvasObject {
	title := container.NewHBox(txt(r.Strategy, colText, 15, true))
	if cheapest {
		title.Add(spacerW(8))
		title.Add(badge("Cheapest", colPositive))
	}

	box := container.NewVBox(
		title,
		txt(blurb, colTextDim, 10.5, false),
		spacerH(10),
	)
	if !r.Feasible {
		box.Add(txt("These payments never pay the debts off — add an extra monthly amount.", colWarning, 11.5, false))
		return panel(container.New(padCell(14, 12), box))
	}

	box.Add(container.NewHBox(
		pillStat("Debt-free", fmtSerialDate(r.DebtFreeDate), colPositive), spacerW(24),
		pillStat("Total interest", fmtMoney(r.TotalInterest), colText),
	))
	box.Add(spacerH(10))
	box.Add(divider())
	box.Add(spacerH(6))
	box.Add(txt("Payoff order", colTextDim, 10.5, true))

	for i, p := range r.Order {
		row := container.NewBorder(nil, nil,
			container.NewHBox(
				txt(itoa(i+1)+".", colTextDim, 12, true), spacerW(6),
				txt(p.Name, colText, 12.5, false),
			),
			container.NewHBox(
				mono(fmtSerialDate(p.PayoffDate), colTextDim, 11.5, false), spacerW(10),
				mono(fmtMoney(p.InterestPaid)+" interest", colTextDim, 11.5, false),
			),
		)
		box.Add(container.New(padCell(4, 4), row))
	}
	return panel(container.New(padCell(14, 12), box))
}
