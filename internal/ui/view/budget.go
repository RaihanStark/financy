package view

import (
	"errors"
	"image/color"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// budgetMonth is the month currently shown on the Budget screen ("YYYY-MM").
// It persists across re-renders within a session and defaults to this month.
var budgetMonth = currentMonthKey()

// ScreenBudget is the zero-based ("envelope") budget, modelled on YNAB: a
// "Ready to Assign" banner on top, then every category with how much was
// Assigned, its Activity, and the rolling Available balance.
func ScreenBudget() fyne.CanvasObject {
	if budgetMonth == "" {
		budgetMonth = currentMonthKey()
	}
	bm := store.BudgetFor(budgetMonth)

	subtitle := "Give every " + store.Currency() + " a job · " + fmtMonthLong(budgetMonth)
	actions := []fyne.CanvasObject{budgetMonthNav()}
	if budgetLocked() {
		subtitle = fmtMonthLong(budgetMonth) + " · 🔒 Locked — past month, view only"
	} else {
		// Auto-Assign only makes sense for a month you can still change.
		actions = append(actions, secondaryButton("Auto-Assign", theme.ContentCopyIcon(), autoAssignLastMonth))
	}
	bar := appBar("Budget", subtitle, actions...)

	body := container.NewVBox(
		readyToAssignHero(bm),
		spacerH(4),
		overspendBanner(bm),
		spacerH(4),
		budgetTable(bm),
		spacerH(8),
		budgetFootnote(),
	)
	return container.NewBorder(bar, nil, nil, nil, container.NewPadded(body))
}

// overspendBanner calls out envelopes in the red so they can't be missed;
// hidden when everything is funded. Its Cover… button walks the overspent
// categories one at a time (the banner re-renders after each move).
func overspendBanner(bm BudgetMonth) fyne.CanvasObject {
	var over []BudgetCategory
	for _, c := range bm.Categories {
		if c.Available < 0 {
			over = append(over, c)
		}
	}
	if len(over) == 0 {
		return spacerH(0)
	}
	msg := itoa(len(over)) + " categor" + plural(len(over), "y", "ies") + " overspent — cover the shortfall from another envelope"
	var action fyne.CanvasObject
	if !budgetLocked() {
		first := over[0]
		action = secondaryButton("Cover…", nil, func() { coverOverspendDialog(first) })
	}
	return alertBanner(theme.WarningIcon(), msg, colNegative, action)
}

// coverOverspendDialog moves money into an overspent envelope from another
// category (or from Ready to Assign) — YNAB's "cover overspending" flow.
func coverOverspendDialog(c BudgetCategory) {
	if win == nil || budgetLocked() || c.Available >= 0 {
		return
	}
	shortfall := -c.Available
	bm := store.BudgetFor(budgetMonth)

	// Possible sources: Ready to Assign first, then every envelope with money
	// available, richest first — the label carries what each can give.
	type source struct {
		id    string
		avail int
	}
	byLabel := map[string]source{}
	var labels []string
	addSource := func(label string, s source) {
		byLabel[label] = s
		labels = append(labels, label)
	}
	if bm.ReadyToAssign > 0 {
		addSource("Ready to Assign  ·  "+fmtMoney(bm.ReadyToAssign), source{"", bm.ReadyToAssign})
	}
	cats := append([]BudgetCategory(nil), bm.Categories...)
	sort.SliceStable(cats, func(i, j int) bool { return cats[i].Available > cats[j].Available })
	for _, o := range cats {
		if o.ID != c.ID && o.Available > 0 {
			addSource(o.Name+"  ·  "+fmtMoney(o.Available)+" available", source{o.ID, o.Available})
		}
	}
	if len(labels) == 0 {
		showInfo("Cover overspending", "No envelope has money available to move. Add funds or reduce other assignments first.")
		return
	}

	sel := widget.NewSelect(labels, nil)
	sel.SetSelectedIndex(0)
	entry := newAmountEntry()
	entry.SetText(fmtMoneyInput(shortfall))

	form := container.NewVBox(
		detailField("Category", c.Name, colText),
		detailField("Overspent by", fmtMoney(shortfall), colNegative),
		spacerH(10),
		txt("MOVE FROM", colTextDim, 10.5, true),
		sel,
		spacerH(8),
		txt("AMOUNT", colTextDim, 10.5, true),
		entry,
	)

	var m *modal
	move := primaryButton("Move", theme.ConfirmIcon(), func() {
		src, ok := byLabel[sel.Selected]
		if !ok {
			return
		}
		amt := parseAmount(entry.Text)
		switch {
		case amt <= 0:
			showError(errors.New("enter an amount greater than zero"))
			return
		case amt > src.avail:
			showError(errors.New("that envelope only has " + fmtMoney(src.avail) + " available"))
			return
		}
		m.Hide()
		store.MoveAssigned(budgetMonth, src.id, c.ID, amt)
	})
	cancel := secondaryButton("Cancel", nil, func() { m.Hide() })
	m = newModal("Cover overspending — "+c.Name, form)
	m.SetButtons(cancel, move)
	m.SetCardSize(fyne.NewSize(460, 0))
	m.Show()
}

// budgetMonthNav is the ◀ Month ▶ control plus a jump-to-today button.
func budgetMonthNav() fyne.CanvasObject {
	prev := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		budgetMonth = shiftMonth(budgetMonth, -1)
		render()
	})
	prev.Importance = widget.LowImportance

	next := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		budgetMonth = shiftMonth(budgetMonth, 1)
		render()
	})
	next.Importance = widget.LowImportance

	today := widget.NewButton("This Month", func() {
		budgetMonth = currentMonthKey()
		render()
	})
	today.Importance = widget.LowImportance

	// Match the label box height to the buttons so the centered text lines up
	// with them vertically (HBox top-aligns children of differing heights).
	h := prev.MinSize().Height
	label := container.NewGridWrap(fyne.NewSize(150, h),
		container.NewCenter(txt(fmtMonthLong(budgetMonth), colText, 13.5, true)))

	return container.NewHBox(prev, label, next, spacerW(6), today)
}

// readyToAssignHero is the prominent banner telling the user how much money is
// still waiting to be given a job.
func readyToAssignHero(bm BudgetMonth) fyne.CanvasObject {
	rta := bm.ReadyToAssign

	var tone color.Color
	var headline, sub string
	switch {
	case rta > 0:
		tone, headline, sub = colPositive, "Ready to Assign", "Money waiting for a job"
	case rta == 0:
		tone, headline, sub = colTextDim, "All Money Assigned", "Every "+store.Currency()+" has a job 🎉"
	default:
		tone, headline, sub = colNegative, "You've Assigned More Than You Have", "Reduce assignments to fix this"
	}

	left := container.NewVBox(
		txt(headline, colTextDim, 11, true),
		spacerH(4),
		txt(fmtMoney(rta), tone, 30, true),
		spacerH(3),
		txt(sub, colTextDim, 11, false),
	)
	// Show where the number comes from: on-budget funds − money already committed
	// to categories (across every month). These are the global position, so the
	// figure is the same whatever month you're viewing.
	right := container.NewVBox(
		alignRight(txt("BUDGET FUNDS  "+fmtMoney(bm.Funds), colText, 11.5, true)),
		alignRight(txt("−  ASSIGNED (ALL MONTHS)  "+fmtMoney(bm.Funds-rta), colTextDim, 11.5, false)),
		alignRight(txt("=  READY TO ASSIGN  "+fmtMoney(rta), tone, 11.5, true)),
		spacerH(4),
		alignRight(txt("Assigned this month  "+fmtMoney(bm.TotalAssigned), colTextDim, 10.5, false)),
	)
	vCentered := container.NewVBox(spacerV(), right, spacerV())

	inner := container.New(&columnsLayout{Weights: []float32{2, 1.6}, Gap: 16}, left, vCentered)
	return panel(container.New(padCell(10, 12), inner))
}

// budgetTable renders the Category / Assigned / Activity / Available grid.
func budgetTable(bm BudgetMonth) fyne.CanvasObject {
	header := budgetHeaderRow()

	rows := []fyne.CanvasObject{header}
	if len(bm.Categories) == 0 {
		rows = append(rows, container.New(padCell(18, 10), emptyState("No categories yet — add Expense categories to start budgeting")))
	}
	// Spending categories first, then the "Add category" line, then the debts you
	// owe under their own subheader — each debt is an envelope you fund as you pay
	// it down. (Categories arrive grouped: expense accounts before debt envelopes.)
	debtsStarted := false
	for i, c := range bm.Categories {
		if c.IsDebt && !debtsStarted {
			debtsStarted = true
			rows = append(rows, budgetAddRow(), budgetSubheader("DEBT PAYMENTS"))
		}
		rows = append(rows, budgetCategoryRow(c, i))
	}
	if !debtsStarted {
		rows = append(rows, budgetAddRow())
	}
	rows = append(rows, budgetTotalsRow(bm))

	return panel(container.NewVBox(rows...))
}

// budgetSubheader is a small section label dividing the category list (e.g. the
// "Debt Payments" group that separates debt envelopes from spending categories).
func budgetSubheader(label string) fyne.CanvasObject {
	return container.NewVBox(divider(), container.New(padCell(8, 6), txt(label, colTextDim, 11, true)))
}

// budgetAddRow is the inline "+ Add Category" line at the bottom of the list,
// mirroring YNAB's in-place category creation.
func budgetAddRow() fyne.CanvasObject {
	label := container.NewHBox(
		widget.NewIcon(theme.ContentAddIcon()),
		txt("Add Category", colPrimary, 12.5, true),
	)
	return newTappableRow(container.New(padCell(8, 6), label), colSurface, AddBudgetCategory)
}

func budgetHeaderRow() fyne.CanvasObject {
	cells := []fyne.CanvasObject{
		txt("CATEGORY", colTextDim, 11, true),
		alignRight(txt("ASSIGNED", colTextDim, 11, true)),
		alignRight(txt("SPENT", colTextDim, 11, true)),
		txt("   USED", colTextDim, 11, true),
		alignRight(txt("AVAILABLE", colTextDim, 11, true)),
	}
	grid := container.New(&columnsLayout{Weights: budgetWeights(), Gap: 14}, cells...)
	return container.NewVBox(container.New(padCell(4, 6), grid), divider())
}

func budgetWeights() []float32 { return []float32{2.2, 1, 1, 1.5, 1.2} }

// spentOf is the money that left the envelope this month, as a positive figure.
func spentOf(c BudgetCategory) int {
	if c.Activity < 0 {
		return -c.Activity
	}
	return 0
}

// spentCell shows spending without sign noise: "—" when untouched, the amount
// when money went out, a green "+" figure for refunds/credits.
func spentCell(activity int) fyne.CanvasObject {
	switch {
	case activity < 0:
		return alignRight(mono(fmtMoney(-activity), colText, 12.5, false))
	case activity > 0:
		return alignRight(mono("+"+fmtMoney(activity), colPositive, 12.5, false))
	default:
		return alignRight(mono("—", colTextDim, 12.5, false))
	}
}

// usageBar is the traffic-light envelope gauge: emerald while healthy, amber
// when nearly (or fully) used, rose when overspent. This is the column that
// makes trouble visible without reading a single number.
func usageBar(c BudgetCategory) fyne.CanvasObject {
	spent := spentOf(c)
	total := spent + c.Available // the envelope's funds this month

	ratio, col := 0.0, colPositive
	switch {
	case c.Available < 0:
		ratio, col = 1, colNegative
	case total <= 0:
		ratio = 0 // empty envelope, nothing spent — bare track
	default:
		ratio = float64(spent) / float64(total)
		if ratio >= 0.85 {
			col = colWarning
		}
	}
	bar := container.NewVBox(spacerV(), container.New(padCell(0, 6), progressBar(ratio, col)), spacerV())
	return bar
}

// budgetCategoryRow is one tappable category line; tapping opens the assigner.
func budgetCategoryRow(c BudgetCategory, idx int) fyne.CanvasObject {
	name := container.NewCenter(txt(c.Name, colText, 13, false))
	nameCell := container.NewHBox(name)

	assigned := alignRight(mono(fmtMoney(c.Assigned), assignedColor(c.Assigned), 12.5, false))
	if c.Assigned == 0 {
		assigned = alignRight(mono("—", colTextDim, 12.5, false))
	}

	grid := container.New(&columnsLayout{Weights: budgetWeights(), Gap: 14},
		nameCell, vCenterCell(assigned), vCenterCell(spentCell(c.Activity)), usageBar(c), vCenterCell(availablePill(c)),
	)

	fill := colSurface
	if idx%2 == 1 {
		fill = colAltRow
	}
	row := newTappableRow(container.New(padCell(7, 6), grid), fill, func() { assignDialog(c) })
	// Past months are locked: tapping still shows a read-only breakdown, but the
	// editing context menu is only wired up for editable months.
	if !budgetLocked() {
		row.SetOnSecondary(func(pos fyne.Position) { showContextMenu(pos, budgetMenuItems(c)...) })
	}
	return row
}

// vCenterCell vertically centers a fixed-height cell within a stretched row.
func vCenterCell(o fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(spacerV(), o, spacerV())
}

// budgetLocked reports whether the month being viewed is a locked past month.
func budgetLocked() bool { return !monthEditable(budgetMonth) }

// availablePill colours the rolling balance: green when funded, a filled rose
// pill when overspent (so red rows jump out), amber when there's spending but
// nothing left, grey when zero.
func availablePill(c BudgetCategory) fyne.CanvasObject {
	switch {
	case c.Available < 0:
		bg := canvas.NewRectangle(withAlpha(colNegative, 0x1c))
		bg.CornerRadius = 7
		pill := container.NewStack(bg, container.New(padCell(2, 7),
			mono(fmtMoney(c.Available), colNegative, 12.5, true)))
		return container.NewHBox(layout.NewSpacer(), pill)
	case c.Available == 0 && c.Activity < 0:
		return alignRight(mono(fmtMoney(c.Available), colWarning, 12.5, true))
	case c.Available == 0:
		return alignRight(mono(fmtMoney(c.Available), colTextDim, 12.5, false))
	default:
		return alignRight(mono(fmtMoney(c.Available), colPositive, 12.5, true))
	}
}

func budgetTotalsRow(bm BudgetMonth) fyne.CanvasObject {
	totalBar := usageBar(BudgetCategory{Activity: bm.TotalActivity, Available: bm.TotalAvailable})
	cells := []fyne.CanvasObject{
		vCenterCell(txt("Total", colText, 13, true)),
		vCenterCell(alignRight(mono(fmtMoney(bm.TotalAssigned), colText, 12.5, true))),
		vCenterCell(spentCell(bm.TotalActivity)),
		totalBar,
		vCenterCell(alignRight(mono(fmtMoney(bm.TotalAvailable), moneyColor(bm.TotalAvailable), 12.5, true))),
	}
	grid := container.New(&columnsLayout{Weights: budgetWeights(), Gap: 14}, cells...)
	return container.NewVBox(divider(), container.New(padCell(7, 6), grid))
}

func budgetFootnote() fyne.CanvasObject {
	return txt("Available rolls forward each month. Assigned + Activity adjusts the envelope; "+
		"a negative Available (red) means that category is overspent.", colTextDim, 10.5, false)
}

// ---- assign dialog ----

// assignDialog lets the user set the amount assigned to a category this month,
// with one-tap suggestions like YNAB's quick-budget.
func assignDialog(c BudgetCategory) {
	if win == nil {
		return
	}
	// Locked past month: show a read-only breakdown only.
	if budgetLocked() {
		ro := container.NewVBox(
			detailField("Category", c.Name, colText),
			detailField("Assigned", fmtMoney(c.Assigned), colText),
			detailField("Activity", fmtMoney(c.Activity), moneyColor(c.Activity)),
			detailField("Available", fmtMoney(c.Available), moneyColor(c.Available)),
			spacerH(8),
			txt(fmtMonthLong(budgetMonth)+" is locked — past months can't be changed.", colTextDim, 11, false),
		)
		showDetail(c.Name+" — "+fmtMonthLong(budgetMonth), container.NewPadded(ro))
		return
	}
	entry := newAmountEntry()
	entry.SetText(fmtMoneyInput(c.Assigned))

	lastAssigned := store.AssignedLastMonth(budgetMonth, c.ID)
	lastSpent := store.SpentLastMonth(budgetMonth, c.ID)
	avgSpent := store.AverageSpent(budgetMonth, c.ID, 3)

	set := func(v int) { entry.SetText(fmtMoneyInput(v)) }

	quick := container.NewVBox(
		quickAssignBtn("Assigned last month", lastAssigned, set),
		quickAssignBtn("Spent last month", lastSpent, set),
		quickAssignBtn("Average spent (3 mo)", avgSpent, set),
		quickAssignBtn("Reset to zero", 0, set),
	)

	info := container.NewVBox(
		detailField("Category", c.Name, colText),
		detailField("Activity this month", fmtMoney(c.Activity), moneyColor(c.Activity)),
		detailField("Available", fmtMoney(c.Available), moneyColor(c.Available)),
	)

	form := container.NewVBox(
		info,
		spacerH(8),
		txt("Assign for "+fmtMonthLong(budgetMonth), colTextDim, 11, true),
		entry,
		spacerH(8),
		txt("QUICK ASSIGN", colTextDim, 10.5, true),
		quick,
	)

	d := dialogWithSave("Assign — "+c.Name, form, func() {
		store.SetAssigned(budgetMonth, c.ID, parseAmount(entry.Text))
	})
	d.Show()
}

func quickAssignBtn(label string, amount int, set func(int)) fyne.CanvasObject {
	b := widget.NewButton(label+"  ·  "+fmtMoney(amount), func() { set(amount) })
	b.Alignment = widget.ButtonAlignLeading
	b.Importance = widget.LowImportance
	return b
}

// budgetMenuItems is the right-click menu for a category row; an overspent
// envelope gets "Cover overspending…" on top.
func budgetMenuItems(c BudgetCategory) []*fyne.MenuItem {
	assign := fyne.NewMenuItem("Assign…", func() { assignDialog(c) })
	assign.Icon = theme.DocumentCreateIcon()
	coverLast := fyne.NewMenuItem("Assign last month's amount", func() {
		store.SetAssigned(budgetMonth, c.ID, store.AssignedLastMonth(budgetMonth, c.ID))
	})
	coverLast.Icon = theme.ContentCopyIcon()
	clear := fyne.NewMenuItem("Clear assignment", func() {
		store.SetAssigned(budgetMonth, c.ID, 0)
	})
	clear.Icon = theme.ContentClearIcon()

	items := []*fyne.MenuItem{assign, coverLast, fyne.NewMenuItemSeparator(), clear}
	if c.Available < 0 {
		cover := fyne.NewMenuItem("Cover overspending…", func() { coverOverspendDialog(c) })
		cover.Icon = theme.MoveDownIcon()
		items = append([]*fyne.MenuItem{cover}, items...)
	}
	return items
}

// AddBudgetCategory is a fast, one-field way to create a new budget category
// (an Expense account) straight from the Budget screen. Exported so the toolbar
// "+" can reach it too.
func AddBudgetCategory() {
	if win == nil {
		return
	}
	name := widget.NewEntry()
	name.SetPlaceHolder("e.g. Vacation, Gifts, Car Maintenance")
	showForm("New Category", []*widget.FormItem{
		widget.NewFormItem("Category name", name),
	}, func() {
		n := strings.TrimSpace(name.Text)
		if n == "" {
			return
		}
		if store.AccountByName(n) != nil {
			showInfo("Category exists", "A category named “"+n+"” already exists.")
			return
		}
		store.AddAccount(Account{Name: n, Type: Expense})
	})
}

// autoAssignLastMonth copies the previous month's assignments into this month,
// but first shows a preview of exactly what will be assigned/adjusted so the
// user can confirm before any current values are overwritten.
func autoAssignLastMonth() {
	if budgetLocked() {
		return // locked past month — not editable
	}
	changes := store.PreviewAutoAssign(budgetMonth)
	if len(changes) == 0 {
		showInfo("Auto-Assign", "Nothing to copy — last month has no assignments.")
		return
	}

	header := widget.NewLabelWithStyle(
		"Copy assignments from "+fmtMonthLong(shiftMonth(budgetMonth, -1))+
			" into "+fmtMonthLong(budgetMonth)+"?",
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	rows := container.NewVBox()
	for _, ch := range changes {
		name := widget.NewLabel(ch.Name)
		var change string
		if ch.From == 0 {
			change = "— → " + fmtMoney(ch.To) // newly assigned
		} else {
			change = fmtMoney(ch.From) + " → " + fmtMoney(ch.To) // overwritten
		}
		val := widget.NewLabelWithStyle(change, fyne.TextAlignTrailing, fyne.TextStyle{})
		rows.Add(container.NewBorder(nil, nil, name, val))
	}

	note := widget.NewLabel(itoa(len(changes)) + " categor" + plural(len(changes), "y", "ies") +
		" will change. Current values shown on the left will be overwritten.")
	note.Wrapping = fyne.TextWrapWord

	content := container.NewBorder(header, note, nil, nil,
		container.NewVScroll(rows))

	showActionConfirm("Auto-Assign", "Assign", content, func() {
		store.AssignLastMonthsAmounts(budgetMonth)
	})
}

func assignedColor(v int) color.Color {
	if v == 0 {
		return colTextDim
	}
	return colText
}
