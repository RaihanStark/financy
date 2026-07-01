package mobileui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/raihanstark/financy/internal/core"
)

// debtsScreen lists each debt with its outstanding balance and payment progress.
// Tapping a debt opens its detail page; the header has an Add-debt action.
func (m *mobileApp) debtsScreen() fyne.CanvasObject {
	s := m.store
	debts := s.Debts()

	header := insets(screenHeader("Debts", "Outstanding "+core.FmtMoney(s.TotalDebtOutstanding())), 12, 8, 16, 16)

	if len(debts) == 0 {
		return container.NewBorder(header, nil, nil, nil,
			container.NewVScroll(insets(mutedCard("No debts tracked — tap Add to track one."), 4, 0, 16, 16)))
	}

	col := container.NewVBox(gap(4))
	for _, d := range debts {
		d := d
		col.Add(newTappableCard(m.debtCard(d), func() { m.openDebt(d) }))
		col.Add(gap(10))
	}
	col.Add(gap(110))
	return container.NewBorder(header, nil, nil, nil, container.NewVScroll(insets(col, 0, 0, 16, 16)))
}

func (m *mobileApp) debtCard(d core.Debt) fyne.CanvasObject {
	st := m.store.DebtStatus(d.ID, core.TodaySerial)

	sub := strings.TrimSpace(d.Lender)
	if sub == "" {
		sub = d.Type
	}
	name := newText(d.Name, colInk, 15, true)
	lender := newText(sub, colInkFaint, 11, false)
	remain := newText(core.FmtMoney(st.Remaining), colInk, 16, true)
	remain.Alignment = fyne.TextAlignTrailing

	frac := float64(st.Paid) / float64(max1(st.Total))
	progress := miniBar(frac)

	status := strconv.Itoa(st.PaidCount) + " of " + strconv.Itoa(st.Count) + " paid"
	if st.OverdueCount > 0 {
		status += "  ·  " + strconv.Itoa(st.OverdueCount) + " overdue"
	}
	statusText := newText(status, colInkDim, 11, false)
	if st.OverdueCount > 0 {
		statusText.Color = colNeg
	}

	top := container.NewBorder(nil, nil, container.NewVBox(name, lender), container.NewCenter(remain))
	inner := container.NewVBox(top, gap(10), progress, gap(6), statusText)
	return container.NewStack(rounded(colCard, 16), insets(inner, 16, 16, 16, 16))
}
