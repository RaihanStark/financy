package view

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// debtsTab is the ID of the debt whose tab is selected. It's preserved across the
// screen rebuilds that follow every pay / undo / edit, so you stay on the debt you
// were looking at instead of being thrown back to the first tab.
var debtsTab string

// ScreenDebts shows every tracked debt — BNPL plans, amortizing loans, credit
// cards and IOUs — under an overview dashboard, as a tabbed master/detail: a
// tab per debt down the side, each opening a detail pane matched to its kind.
func ScreenDebts() fyne.CanvasObject {
	bar := appBar("Debts", "Loans, cards, BNPL and IOUs in one place",
		secondaryButton("Strategies", theme.ViewRefreshIcon(), func() { strategyDialog() }),
		primaryButton("Add Debt", theme.ContentAddIcon(), func() { DebtForm(nil) }),
	)

	ds := store.Debts()
	if len(ds) == 0 {
		body := panel(emptyState("No debts yet. Track a BNPL plan, a loan, a credit card, or money owed to a friend…"))
		return container.NewBorder(bar, nil, nil, nil, container.NewPadded(body))
	}

	tabs := container.NewAppTabs()
	tabs.SetTabLocation(container.TabLocationLeading)
	selected := 0
	for i, d := range ds {
		title := d.Name
		if debtNeedsAttention(d) {
			title = "● " + title // amber dot hint: overdue, or a statement to review
		}
		tabs.Append(container.NewTabItem(title, debtTabContent(d)))
		if d.ID == debtsTab {
			selected = i
		}
	}
	tabs.SelectIndex(selected)
	debtsTab = ds[selected].ID
	tabs.OnSelected = func(*container.TabItem) {
		if i := tabs.SelectedIndex(); i >= 0 && i < len(ds) {
			debtsTab = ds[i].ID
		}
	}

	head := container.NewPadded(container.NewVBox(debtDashboard(), statementBanner(), debtDueBanner()))
	return container.NewBorder(
		container.NewVBox(bar, head), nil, nil, nil,
		container.NewPadded(tabs),
	)
}

// debtNeedsAttention flags a debt for the tab-title dot: an overdue installment
// on schedule kinds, a statement awaiting review on revolving, a blown due date
// on informal.
func debtNeedsAttention(d Debt) bool {
	switch {
	case d.IsRevolving():
		return d.NextStatement > 0 && d.NextStatement <= todaySerial
	case d.HasSchedule():
		return store.DebtStatus(d.ID, todaySerial).OverdueCount > 0
	default:
		return d.DueDate > 0 && d.DueDate <= todaySerial && store.DebtBalance(d.ID) > 0
	}
}

// debtDashboard is the overview header: the aggregate position across every
// debt.
func debtDashboard() fyne.CanvasObject {
	o := store.DebtOverview(todaySerial)
	b := store.DebtBuckets(todaySerial)

	apr := "—"
	if o.WeightedAPRBps > 0 {
		apr = fmtAPRBps(o.WeightedAPRBps)
	}
	debtFree := "—"
	if o.TotalOwed == 0 {
		debtFree = "you're debt-free"
	} else if o.DebtFreeDate > 0 {
		debtFree = fmtSerialDate(o.DebtFreeDate)
	}
	dueCol := colTextDim
	if b.Due > 0 {
		dueCol = colWarning
	}
	return container.NewGridWithColumns(4,
		statCard("Total owed", fmtMoney(o.TotalOwed), colNegative, ""),
		statCard("Average APR", apr, colText, "balance-weighted"),
		statCard("Due now", fmtMoney(b.Due), dueCol, ""),
		statCard("Debt-free by", debtFree, colPositive, "at current plans"),
	)
}

// statementBanner surfaces revolving statements awaiting review.
func statementBanner() fyne.CanvasObject {
	pending := store.PendingStatements(todaySerial)
	if len(pending) == 0 {
		return spacerH(0)
	}
	msg := itoa(len(pending)) + " card statement" + plural(len(pending), "", "s") + " to review"
	review := secondaryButton("Review", theme.SearchIcon(), func() {
		statementReviewDialog(pending[0])
	})
	return container.NewVBox(spacerH(6), alertBanner(theme.InfoIcon(), msg, colWarning, review))
}

// debtStat is one inline "label  value" pair for the Debts header.
func debtStat(label, value string, valueCol color.Color) fyne.CanvasObject {
	return container.NewHBox(
		txt(label+"  ", colTextDim, 12, true),
		txt(value, valueCol, 15, true),
	)
}

// debtDueBanner nudges when payments need attention.
func debtDueBanner() fyne.CanvasObject {
	n := store.DueDebtCount(todaySerial)
	if n == 0 {
		return spacerH(0)
	}
	msg := itoa(n) + " payment" + plural(n, "", "s") + " due"
	return container.NewHBox(spacerH(0), txt("●  ", colWarning, 13, true), txt(msg, colText, 12.5, true))
}
