package mobileui

import (
	"errors"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// budgetScreen shows the selected month's ready-to-assign figure and each
// category's envelope. Months are navigable; tapping a category assigns money.
func (m *mobileApp) budgetScreen() fyne.CanvasObject {
	if m.budgetMonth == "" {
		m.budgetMonth = core.CurrentMonthKey()
	}
	month := m.budgetMonth
	bm := m.store.BudgetFor(month)

	header := insets(container.NewVBox(
		screenHeader("Budget", ""),
		gap(6),
		m.budgetMonthNav(month),
	), 12, 8, 16, 16)

	col := container.NewVBox(gap(4), m.readyToAssignCard(bm))
	if len(bm.Categories) == 0 {
		col.Add(gap(10))
		col.Add(mutedCard("No budget categories yet"))
	} else {
		// Split spending categories (Expense accounts) from debt-payment envelopes
		// (Liability accounts) into their own sections.
		var spending, debts []core.BudgetCategory
		for _, c := range bm.Categories {
			if c.IsDebt {
				debts = append(debts, c)
			} else {
				spending = append(spending, c)
			}
		}

		catCard := func(cats []core.BudgetCategory) fyne.CanvasObject {
			rows := make([]fyne.CanvasObject, 0, len(cats)*2)
			for i, c := range cats {
				c := c
				if i > 0 {
					rows = append(rows, rowDivider())
				}
				rows = append(rows, newTappableCard(budgetRow(c), func() { m.assignCategory(month, c) }))
			}
			return listCard(rows...)
		}

		if len(spending) > 0 {
			col.Add(gap(14))
			col.Add(sectionHeader("Spending", "Auto-assign", func() { m.autoAssign(month) }))
			col.Add(gap(2))
			col.Add(catCard(spending))
		}
		if len(debts) > 0 {
			col.Add(gap(16))
			col.Add(sectionHeader("Debt payments", "", nil))
			col.Add(gap(2))
			col.Add(catCard(debts))
		}
	}
	col.Add(gap(120))
	return container.NewBorder(header, nil, nil, nil, container.NewVScroll(insets(col, 0, 0, 16, 16)))
}

// budgetMonthNav is the ◀ Month ▶ selector.
func (m *mobileApp) budgetMonthNav(month string) fyne.CanvasObject {
	prev := iconButton(theme.NavigateBackIcon(), func() {
		m.budgetMonth = shiftMonth(month, -1)
		m.render()
	})
	next := iconButton(theme.NavigateNextIcon(), func() {
		m.budgetMonth = shiftMonth(month, 1)
		m.render()
	})
	label := newText(core.FmtMonthLong(month), colInk, 14, true)
	label.Alignment = fyne.TextAlignCenter
	return container.NewBorder(nil, nil, prev, next, container.NewCenter(label))
}

// autoAssign previews the changes copying last month's assignments would make,
// then asks the user to confirm before overwriting any current values.
func (m *mobileApp) autoAssign(month string) {
	changes := m.store.PreviewAutoAssign(month)
	if len(changes) == 0 {
		dialog.ShowInformation("Auto-assign",
			"Nothing to copy — "+core.FmtMonthLong(shiftMonth(month, -1))+" has no assignments.", m.win)
		return
	}

	m.pushPage(func(close func()) fyne.CanvasObject {
		rows := make([]fyne.CanvasObject, 0, len(changes)*2)
		for i, ch := range changes {
			if i > 0 {
				rows = append(rows, thinLine())
			}
			change := core.FmtMoney(ch.From) + "  →  " + core.FmtMoney(ch.To)
			if ch.From == 0 {
				change = "—  →  " + core.FmtMoney(ch.To) // newly assigned
			}
			rows = append(rows, detailRow(ch.Name, change))
		}
		list := container.NewStack(rounded(colCard, 16), insets(container.NewVBox(rows...), 4, 4, 6, 6))

		assign := widget.NewButton("Assign", func() {
			close()
			m.store.AssignLastMonthsAmounts(month)
			m.render()
		})
		assign.Importance = widget.HighImportance
		cancel := widget.NewButton("Cancel", close)

		body := container.NewVBox(
			wrapText("Copy assignments from "+core.FmtMonthLong(shiftMonth(month, -1))+
				" into "+core.FmtMonthLong(month)+"? Current values on the left will be overwritten."),
			gap(12),
			list,
			gap(18),
			assign,
			gap(6),
			cancel,
		)
		return m.formPage("Auto-assign", "", body, nil, close)
	})
}

// addCategory creates a new budget category (an Expense account) — the Budget
// tab's FAB action.
func (m *mobileApp) addCategory() {
	if m.store == nil {
		return
	}
	name := newEntry()
	name.SetPlaceHolder("e.g. Groceries")
	nameF := field("Category name", name)
	body := container.NewVBox(
		nameF,
		wrapText("Categories are your Expense categories. After adding one, tap it on the Budget tab to assign money."),
	)

	commit := func() {
		if strings.TrimSpace(name.Text) == "" {
			dialog.ShowError(errors.New("enter a category name"), m.win)
			return
		}
		m.store.AddAccount(core.Account{Name: name.Text, Type: core.Expense})
		m.back()
	}
	m.pushView(m.formPage("Add category", "Add", body, commit, m.back))
}

// assignCategory is the full-screen assign page: an amount field plus one-tap
// quick-assign suggestions (YNAB-style), saving via core.SetAssigned.
func (m *mobileApp) assignCategory(month string, c core.BudgetCategory) {
	amount := newAmountEntry()
	amount.SetText(core.FmtMoneyInput(c.Assigned))

	set := func(v int) { amount.SetText(core.FmtMoneyInput(v)) }
	lastAssigned := m.store.AssignedLastMonth(month, c.ID)
	lastSpent := m.store.SpentLastMonth(month, c.ID)
	avgSpent := m.store.AverageSpent(month, c.ID, 3)

	info := container.NewStack(rounded(colCard, 16), insets(container.NewVBox(
		detailRow("Activity this month", core.FmtMoney(c.Activity)),
		thinLine(),
		detailRow("Available", core.FmtMoney(c.Available)),
	), 6, 6, 14, 14))

	quick := container.NewVBox(
		quickAssignBtn("Assigned last month", lastAssigned, set),
		gap(6),
		quickAssignBtn("Spent last month", lastSpent, set),
		gap(6),
		quickAssignBtn("Average spent (3 mo)", avgSpent, set),
		gap(6),
		quickAssignBtn("Reset to zero", 0, set),
	)

	amountF := field("Assign for "+core.FmtMonthLong(month), amount)
	body := container.NewVBox(
		info,
		gap(14),
		amountF,
		gap(6),
		newText("QUICK ASSIGN", colInkDim, 10, true),
		gap(6),
		quick,
	)

	m.pushPage(func(close func()) fyne.CanvasObject {
		save := func() {
			m.store.SetAssigned(month, c.ID, core.ParseAmount(amount.Text))
			close()
			m.render() // refresh the budget tab behind the page
		}
		return m.formPage("Assign — "+c.Name, "Save", body, save, close)
	})
}

func quickAssignBtn(label string, amount int, set func(int)) fyne.CanvasObject {
	b := widget.NewButton(label+"  ·  "+core.FmtMoney(amount), func() { set(amount) })
	b.Alignment = widget.ButtonAlignLeading
	b.Importance = widget.LowImportance
	return b
}

// shiftMonth returns the "YYYY-MM" key delta months from key.
func shiftMonth(key string, delta int) string {
	t, err := time.Parse("2006-01", key)
	if err != nil {
		return key
	}
	return t.AddDate(0, delta, 0).Format("2006-01")
}

func (m *mobileApp) readyToAssignCard(bm core.BudgetMonth) fyne.CanvasObject {
	inner := container.NewVBox(
		newText("READY TO ASSIGN", withAlpha(white, 0xC4), 11, true),
		gap(2),
		newText(core.FmtMoney(bm.ReadyToAssign), white, 26, true),
	)
	return container.NewStack(rounded(colHero, 20), insets(inner, 18, 18, 20, 20))
}

func budgetRow(c core.BudgetCategory) fyne.CanvasObject {
	availCol := colInk
	if c.Available < 0 {
		availCol = colNeg
	} else if c.Available > 0 {
		availCol = colPos
	}
	title := newText(ellipsize(c.Name, 24), colInk, 14, true)
	meta := newText("assigned "+core.FmtMoney(c.Assigned), colInkDim, 11, false)
	avail := newText(core.FmtMoney(c.Available), availCol, 15, true)
	avail.Alignment = fyne.TextAlignTrailing

	// Fill = how much of the assigned envelope has been spent this month.
	spent := -c.Activity
	frac := 0.0
	if c.Assigned > 0 {
		frac = float64(spent) / float64(c.Assigned)
	}
	bar := miniBar(frac)

	top := container.NewBorder(nil, nil, flexBox(container.NewVBox(title, meta)), container.NewCenter(avail))
	inner := container.NewVBox(top, gap(8), bar)
	return insets(inner, 10, 10, 12, 12)
}

// miniBar is a thin rounded progress track filled to frac (0..1, clamped).
func miniBar(frac float64) fyne.CanvasObject {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	track := rounded(withAlpha(colInkDim, 0x22), 4)
	track.SetMinSize(fyne.NewSize(0, 7))
	fill := rounded(colPrimary, 4)
	fill.SetMinSize(fyne.NewSize(0, 7))
	left := int(frac*1000 + 0.5)
	grid := container.New(newRatioLayout(left, 1000-left), fill, gap(0))
	return container.NewStack(track, grid)
}
