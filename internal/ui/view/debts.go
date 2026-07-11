package view

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// ScreenDebts shows every tracked debt — BNPL plans, amortizing loans, credit
// cards and IOUs — under an overview dashboard, as a compact table: one row
// per debt (kind, APR, balance, progress, status). Tapping a row opens that
// debt's detail dialog.
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

	head := container.NewVBox(debtDashboard(), statementBanner(), debtDueBanner())
	return container.NewVBox(
		bar,
		container.NewPadded(head),
		container.NewPadded(panel(debtsTable(ds))),
	)
}

// debtsTable is the compact all-debts list: a header row plus one tappable
// line per debt. Tapping a row opens its detail dialog.
func debtsTable(ds []Debt) fyne.CanvasObject {
	cols := &columnsLayout{Weights: []float32{2.4, 1.0, 0.7, 1.4, 1.6, 1.7}, Gap: 10}
	h := func(s string) fyne.CanvasObject { return txt(s, colTextDim, 10.5, true) }
	header := container.New(cols,
		h("Debt"), h("Type"), alignRight(txt("APR", colTextDim, 10.5, true)),
		alignRight(txt("Balance", colTextDim, 10.5, true)), h("Progress"), h("Status"))

	table := container.NewVBox(container.New(padCell(2, 8), header), divider())
	for i, d := range ds {
		table.Add(debtsTableRow(d, cols, i))
	}
	return container.New(padCell(8, 8), table)
}

func debtsTableRow(d Debt, cols *columnsLayout, idx int) fyne.CanvasObject {
	bal := store.DebtBalance(d.ID)
	ratio, ratioCol := debtProgress(d, bal)
	status, statusCol := debtStatusLine(d, bal)

	apr := "—"
	if d.APRBps > 0 {
		apr = fmtAPRBps(d.APRBps)
	}
	cells := container.New(cols,
		txt(d.Name, colText, 12.5, true),
		txt(d.Type, colTextDim, 12, false),
		alignRight(mono(apr, colTextDim, 12, false)),
		alignRight(mono(fmtMoney(bal), colText, 12.5, false)),
		container.New(padCell(2, 6), progressBar(ratio, ratioCol)),
		txt(status, statusCol, 12, false),
	)

	var fill color.Color = colSurface
	if idx%2 == 1 {
		fill = colAltRow
	}
	return newTappableRow(container.New(padCell(8, 8), cells), fill, func() {
		showDebtDetail(d.ID)
	})
}

// debtProgress is the row's progress ratio and color: schedule and informal
// debts show how much is repaid; revolving shows credit utilization (tinted
// as it climbs).
func debtProgress(d Debt, bal int) (float64, color.Color) {
	if d.IsRevolving() {
		if d.CreditLimit <= 0 {
			return 0, colPositive
		}
		u := float64(bal) / float64(d.CreditLimit)
		switch {
		case u > 0.7:
			return u, colNegative
		case u > 0.3:
			return u, colWarning
		default:
			return u, colPositive
		}
	}
	if d.HasSchedule() {
		st := store.DebtStatus(d.ID, todaySerial)
		if st.Total <= 0 {
			return 0, colPositive
		}
		return float64(st.Paid) / float64(st.Total), colPositive
	}
	if d.Principal <= 0 {
		return 0, colPositive
	}
	return float64(d.Principal-bal) / float64(d.Principal), colPositive
}

// debtStatusLine is the row's status text: what needs doing next, tinted when
// it needs attention.
func debtStatusLine(d Debt, bal int) (string, color.Color) {
	switch {
	case d.IsRevolving():
		if d.NextStatement > 0 && d.NextStatement <= todaySerial {
			return "statement to review", colWarning
		}
		if bal <= 0 {
			return "paid off", colPositive
		}
		return "min due " + fmtMoney(store.MinPaymentDue(d)), colTextDim
	case d.HasSchedule():
		st := store.DebtStatus(d.ID, todaySerial)
		switch {
		case st.PaidCount == st.Count:
			return "paid off", colPositive
		case st.OverdueCount > 0:
			return itoa(st.OverdueCount) + " overdue", colWarning
		default:
			return "next " + fmtSerialDate(st.NextDue), colTextDim
		}
	default:
		switch {
		case bal <= 0:
			return "settled", colPositive
		case d.DueDate > 0 && d.DueDate <= todaySerial:
			return "due " + fmtSerialDate(d.DueDate), colWarning
		case d.DueDate > 0:
			return "due " + fmtSerialDate(d.DueDate), colTextDim
		default:
			return "no due date", colTextDim
		}
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

// debtDueBanner nudges when payments need attention.
func debtDueBanner() fyne.CanvasObject {
	n := store.DueDebtCount(todaySerial)
	if n == 0 {
		return spacerH(0)
	}
	msg := itoa(n) + " payment" + plural(n, "", "s") + " due"
	return container.NewHBox(spacerH(0), txt("●  ", colWarning, 13, true), txt(msg, colText, 12.5, true))
}
