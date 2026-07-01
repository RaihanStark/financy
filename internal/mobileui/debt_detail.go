package mobileui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// openDebt drills into a debt's detail page from the Debts tab.
func (m *mobileApp) openDebt(d core.Debt) {
	m.pushView(m.debtContent(d))
}

// showDebt re-renders the detail page in place after a mutation.
func (m *mobileApp) showDebt(d core.Debt) {
	m.replaceView(m.debtContent(d))
}

// rebuildDebt re-renders the detail page with fresh data (after pay/undo/edit).
func (m *mobileApp) rebuildDebt(id string) {
	if nd := m.store.DebtByID(id); nd != nil {
		m.showDebt(*nd)
	}
}

func (m *mobileApp) debtContent(d core.Debt) fyne.CanvasObject {
	st := m.store.DebtStatus(d.ID, core.TodaySerial)
	insts := m.store.Installments(d.ID)

	head := m.drilldownBar(d.Name)

	rows := make([]fyne.CanvasObject, 0, len(insts)*2)
	for i, in := range insts {
		if i > 0 {
			rows = append(rows, rowDivider())
		}
		rows = append(rows, m.installmentRow(d, in))
	}

	col := container.NewVBox(
		gap(12),
		m.debtHero(d, st),
		gap(12),
		m.debtActions(d),
		gap(14),
		sectionHeader("Schedule", "", nil),
		gap(4),
		listCard(rows...),
		gap(20),
	)
	body := container.NewVScroll(insets(col, 0, 0, 14, 14))
	return container.NewBorder(head, nil, nil, nil, container.NewStack(canvas.NewRectangle(colBg), body))
}

func (m *mobileApp) debtHero(d core.Debt, st core.DebtStatus) fyne.CanvasObject {
	status := strconv.Itoa(st.PaidCount) + " of " + strconv.Itoa(st.Count) + " paid"
	if st.NextDue > 0 {
		status += "  ·  next " + core.FmtSerialDate(st.NextDue)
	}
	statusT := newText(status, colInkDim, 12, false)
	if st.OverdueCount > 0 {
		statusT = newText(strconv.Itoa(st.OverdueCount)+" overdue  ·  "+status, colNeg, 12, false)
	}

	lender := d.Lender
	if lender == "" {
		lender = d.Type
	}
	inner := container.NewVBox(
		newText("OUTSTANDING", colInkDim, 10, true),
		gap(2),
		newText(core.FmtMoney(st.Remaining), colNeg, 30, true),
		newText(ellipsize(lender, 30), colInkDim, 12, false),
		gap(12),
		miniBar(float64(st.Paid)/float64(max1(st.Total))),
		gap(8),
		statusT,
	)
	return container.NewStack(rounded(withAlpha(colNeg, 0x12), 20), insets(inner, 18, 18, 18, 18))
}

func (m *mobileApp) debtActions(d core.Debt) fyne.CanvasObject {
	edit := widget.NewButton("Edit", func() { m.editDebt(d) })
	edit.Importance = widget.HighImportance
	del := widget.NewButton("Delete", func() { m.confirmDeleteDebt(d) })
	del.Importance = widget.DangerImportance
	return container.NewGridWithColumns(2, edit, del)
}

// installmentRow shows one scheduled payment with a Pay / Undo action.
func (m *mobileApp) installmentRow(d core.Debt, in core.Installment) fyne.CanvasObject {
	overdue := !in.Paid && in.DueDate <= core.TodaySerial
	dotCol := colInkDim
	switch {
	case in.Paid:
		dotCol = colPos
	case overdue:
		dotCol = colNeg
	}
	dot := initialCircle("#"+strconv.Itoa(in.Seq), dotCol, 34)

	due := newText(core.FmtSerialDate(in.DueDate), colInk, 13, true)
	amt := newText(core.FmtMoney(in.Amount), colInkDim, 11, false)
	left := flexBox(container.NewVBox(due, amt))

	var action *widget.Button
	if in.Paid {
		action = widget.NewButton("Undo", func() {
			m.store.UnpayInstallment(in.ID)
			m.rebuildDebt(d.ID)
		})
	} else {
		action = widget.NewButton("Pay", func() {
			m.store.PayInstallment(in.ID, core.TodaySerial)
			m.rebuildDebt(d.ID)
		})
		action.Importance = widget.HighImportance
	}

	row := container.NewBorder(nil, nil, dot, action, insets(left, 0, 0, 10, 10))
	return insets(row, 7, 7, 8, 8)
}

func (m *mobileApp) confirmDeleteDebt(d core.Debt) {
	m.pushPage(func(close func()) fyne.CanvasObject {
		delBtn := widget.NewButton("Delete", func() {
			close() // pop the confirm page
			m.store.DeleteDebt(d.ID)
			m.back() // pop the debt detail → Debts tab
		})
		delBtn.Importance = widget.DangerImportance
		cancel := widget.NewButton("Cancel", close)
		body := container.NewVBox(
			wrapText("Delete “"+d.Name+"”? Its schedule and liability account are removed. This can't be undone."),
			gap(18),
			delBtn,
			gap(6),
			cancel,
		)
		return m.formPage("Delete debt?", "", body, nil, close)
	})
}
