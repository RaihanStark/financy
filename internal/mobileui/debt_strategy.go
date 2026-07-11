package mobileui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/raihanstark/financy/internal/core"
)

// strategyPage compares the two classic payoff strategies: snowball (smallest
// balance first — quick wins) vs avalanche (highest APR first — least
// interest). Enter what extra you can put toward debts each month; required
// payments keep being paid either way, and each cleared debt's payment rolls
// into the next target.
func (m *mobileApp) strategyPage() {
	extra := newAmountEntry()
	extra.SetText(core.FmtMoneyInput(0))
	results := container.NewVBox()

	update := func() {
		snow, aval := m.store.CompareStrategies(core.TodaySerial, core.ParseAmount(extra.Text))
		cheaper := ""
		if snow.Feasible && aval.Feasible {
			switch {
			case aval.TotalInterest < snow.TotalInterest:
				cheaper = aval.Strategy
			case snow.TotalInterest < aval.TotalInterest:
				cheaper = snow.Strategy
			}
		}
		results.Objects = []fyne.CanvasObject{
			m.strategyCard(snow, "Smallest balance first — quick wins keep you motivated.", cheaper == snow.Strategy),
			gap(12),
			m.strategyCard(aval, "Highest APR first — the least total interest.", cheaper == aval.Strategy),
		}
		results.Refresh()
	}
	extra.OnChanged = func(string) { update() }
	update()

	m.pushPage(func(close func()) fyne.CanvasObject {
		body := container.NewVBox(
			field("Extra per month", extra),
			wrapText("On top of all required payments."),
			gap(12),
			results,
			gap(16),
		)
		return m.formPage("Payoff strategies", "", body, nil, close)
	})
}

// strategyCard renders one strategy's outcome: headline numbers + payoff order.
func (m *mobileApp) strategyCard(r core.StrategyResult, blurb string, cheapest bool) fyne.CanvasObject {
	title := r.Strategy
	if cheapest {
		title += "  ·  cheapest"
	}
	titleT := newText(title, colInk, 15, true)
	if cheapest {
		titleT.Color = colPos
	}
	inner := container.NewVBox(
		titleT,
		newText(blurb, colInkFaint, 11, false),
		gap(10),
	)
	if !r.Feasible {
		inner.Add(newText("These payments never pay the debts off —", colNeg, 12, false))
		inner.Add(newText("add an extra monthly amount.", colNeg, 12, false))
		return container.NewStack(rounded(colCard, 16), insets(inner, 14, 14, 16, 16))
	}

	inner.Add(newText("Debt-free "+core.FmtSerialDate(r.DebtFreeDate)+
		"  ·  "+core.FmtMoney(r.TotalInterest)+" interest", colInk, 13, true))
	inner.Add(gap(8))
	for i, p := range r.Order {
		line := strconv.Itoa(i+1) + ". " + ellipsize(p.Name, 18)
		left := newText(line, colInk, 12, false)
		right := newText(core.FmtSerialDate(p.PayoffDate), colInkDim, 11, false)
		right.Alignment = fyne.TextAlignTrailing
		inner.Add(container.NewBorder(nil, nil, left, right))
		inner.Add(gap(4))
	}
	return container.NewStack(rounded(colCard, 16), insets(inner, 14, 14, 16, 16))
}
