package mobileui

import (
	"errors"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// openAccount drills into a single account's detail page (balance hero, quick
// actions, and a running-balance register). The Back button returns to Home.
func (m *mobileApp) openAccount(a core.Account) {
	m.pushView(m.accountContent(a))
}

// showAccount re-renders the account detail in place after a mutation so the
// hero, title, and register stay in sync (does not change the back stack).
func (m *mobileApp) showAccount(a core.Account) {
	m.replaceView(m.accountContent(a))
}

func (m *mobileApp) accountContent(a core.Account) fyne.CanvasObject {
	reg := m.store.Register(a.ID) // newest-first
	head := m.drilldownBar(a.Name)
	top := container.NewVBox(gap(12), m.acctHero(a, reg), gap(14), m.acctActions(a), gap(6))

	var center fyne.CanvasObject
	if len(reg) == 0 {
		center = container.NewVScroll(insets(mutedCard("No transactions for this account"), 4, 0, 16, 16))
	} else {
		center = m.acctRegister(a, reg)
	}
	body := container.NewBorder(top, nil, nil, nil, center)
	return container.NewBorder(head, nil, nil, nil,
		container.NewStack(canvas.NewRectangle(colBg), insets(body, 0, 0, 14, 14)))
}

// acctHero is a tonal, role-tinted balance card (green Asset / red Liability),
// distinct from Home's deep-indigo net-worth hero.
func (m *mobileApp) acctHero(a core.Account, reg []core.RegisterRow) fyne.CanvasObject {
	c := colPos
	caption := "CURRENT BALANCE"
	if a.Type == core.Liability {
		c = colNeg
		caption = "AMOUNT OWED"
	}
	meta := string(a.Type)
	if inst := strings.TrimSpace(a.Institution); inst != "" {
		meta = string(a.Type) + " · " + inst
	}
	metaL := newText(ellipsize(meta, 28), colInkDim, 12, false)
	metaR := newText(pluralCount(len(reg), "transaction"), colInkFaint, 12, false)
	metaR.Alignment = fyne.TextAlignTrailing

	inner := container.NewVBox(
		newText(caption, colInkDim, 10, true),
		gap(2),
		newText(core.FmtMoney(m.store.DisplayBalance(a)), c, 30, true),
		gap(10),
		// metaR fixed on the right; metaL flexible + ellipsized so a long
		// institution name can't force the row wider than the card.
		container.NewBorder(nil, nil, nil, metaR, flexBox(metaL)),
	)
	return container.NewStack(rounded(withAlpha(c, 0x14), 20), insets(inner, 18, 18, 18, 18))
}

// acctActions is the segmented quick-action chip row: Add / Reconcile / Edit /
// Delete, wrapped in one white card so it reads as a single control.
func (m *mobileApp) acctActions(a core.Account) fyne.CanvasObject {
	grid := container.NewGridWithColumns(4,
		acctChip(theme.ContentAddIcon(), "Add", colPrimary, func() {
			m.openAddThen(&a, func() { m.showAccount(a) }) // rebuild so the register/balance update
		}),
		acctChip(theme.ViewRefreshIcon(), "Reconcile", colPrimary, func() { m.reconcileAccount(a) }),
		acctChip(theme.DocumentCreateIcon(), "Edit", colInkDim, func() { m.editAccount(a) }),
		acctChip(theme.DeleteIcon(), "Delete", colNeg, func() { m.deleteAccount(a) }),
	)
	return card(grid)
}

func acctChip(res fyne.Resource, label string, accent color.NRGBA, tap func()) fyne.CanvasObject {
	ic := iconCircle(res, accent)
	lbl := newText(label, colInkDim, 11, false)
	lbl.Alignment = fyne.TextAlignCenter
	col := container.NewVBox(container.NewCenter(ic), gap(4), container.NewCenter(lbl))
	return newTappableCard(insets(col, 10, 10, 4, 4), tap)
}

// acctRegister is the recycling register list; each row shows the payee, date ·
// contra account, signed change (by amount sign), and running balance.
func (m *mobileApp) acctRegister(a core.Account, reg []core.RegisterRow) fyne.CanvasObject {
	store := m.store
	list := widget.NewList(
		func() int { return len(reg) },
		func() fyne.CanvasObject { return newRegRowWidget() },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*regRowWidget).set(store, a, reg[i])
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		list.Unselect(id)
		m.openTransaction(reg[id].Txn)
	}
	list.HideSeparators = true
	return list
}

// ---- recycled register row ----

type regRowWidget struct {
	widget.BaseWidget
	circle  *canvas.Circle
	letter  *canvas.Text
	title   *canvas.Text
	sub     *canvas.Text
	change  *canvas.Text
	running *canvas.Text
}

func newRegRowWidget() *regRowWidget {
	r := &regRowWidget{}
	r.ExtendBaseWidget(r)
	return r
}

func (r *regRowWidget) CreateRenderer() fyne.WidgetRenderer {
	r.circle = canvas.NewCircle(color.Transparent)
	r.letter = newText("", colInk, 14, true)
	av := container.NewGridWrap(fyne.NewSize(36, 36),
		container.NewStack(r.circle, container.NewCenter(r.letter)))

	r.title = newText("", colInk, 14, true)
	r.sub = newText("", colInkDim, 11, false)
	left := flexBox(container.NewVBox(r.title, r.sub))

	r.change = newText("", colInk, 14, true)
	r.change.Alignment = fyne.TextAlignTrailing
	r.running = newText("", colInkDim, 11, false)
	r.running.Alignment = fyne.TextAlignTrailing
	right := container.NewVBox(r.change, r.running)

	row := container.NewBorder(nil, nil, av, right, insets(left, 0, 0, 10, 10))
	return widget.NewSimpleRenderer(insets(row, 7, 7, 12, 12))
}

func (r *regRowWidget) set(s *core.Store, a core.Account, row core.RegisterRow) {
	d := deriveTxn(s, row.Txn)
	c := accentFor(d.kind)
	r.circle.FillColor = withAlpha(c, 0x22)
	r.circle.Refresh()
	r.letter.Text = initialOf(d.title)
	r.letter.Color = c
	r.letter.Refresh()

	r.title.Text = ellipsize(d.title, 26)
	r.title.Refresh()
	r.sub.Text = ellipsize(shortDate(row.Txn.Date)+" · "+contraNames(s, row.Txn, a.ID), 26)
	r.sub.Refresh()

	// Change column: color/sign strictly by the raw posting amount (NOT kind).
	if row.Amount < 0 {
		r.change.Text = "−" + core.FmtMoney(-row.Amount)
		r.change.Color = colNeg
	} else {
		r.change.Text = "+" + core.FmtMoney(row.Amount)
		r.change.Color = colPos
	}
	r.change.Refresh()

	r.running.Text = core.FmtMoney(row.Balance)
	r.running.Refresh()
}

// contraNames lists the names of the other accounts a transaction touches
// (everything except self), mirroring the desktop register's Contra column.
func contraNames(s *core.Store, t core.Transaction, self string) string {
	var names []string
	for _, p := range t.Posts {
		if p.AccountID == self {
			continue
		}
		if acc := s.AccountByID(p.AccountID); acc != nil {
			names = append(names, acc.Name)
		}
	}
	if len(names) == 0 {
		return "—"
	}
	return strings.Join(names, ", ")
}

// signColor mirrors the desktop's moneyColor (internal/ui is not importable):
// positive → green, negative → red, zero → ink.
func signColor(minor int) color.Color {
	switch {
	case minor > 0:
		return colPos
	case minor < 0:
		return colNeg
	default:
		return colInk
	}
}

// ---- sub-forms (pushed over the account page) ----

// reconcileAccount asks for the actual balance and an as-of date, then posts an
// adjustment via core.Reconcile and rebuilds the detail page.
func (m *mobileApp) reconcileAccount(a core.Account) {
	current := m.store.DisplayBalance(a)
	noun := "balance"
	if a.Type == core.Liability {
		noun = "amount owed"
	}

	currentT := newText(core.FmtMoney(current), signColor(current), 20, true)
	currentControl := insets(container.NewHBox(currentT), 8, 8, 0, 0)

	actual := newAmountEntry()
	actual.SetText(core.FmtMoneyInput(current))
	dateF, dateSerial := m.dateField("As of", core.TodaySerial)

	currentF := field("Financy "+noun, currentControl)
	actualF := field("Actual "+noun, actual)
	body := container.NewVBox(currentF, actualF, dateF)
	targets := map[fyne.Focusable]fyne.CanvasObject{actual: actualF}

	commit := func() {
		target := core.ParseAmount(actual.Text)
		d := dateSerial()
		if d == 0 {
			d = core.TodaySerial
		}
		delta, created := m.store.Reconcile(a.ID, target, d)
		if !created {
			dialog.ShowInformation("Already balanced", "No adjustment was needed.", m.win)
			return
		}
		sign := ""
		if delta > 0 {
			sign = "+"
		}
		dialog.ShowInformation("Reconciled",
			"Adjustment "+sign+core.FmtMoney(delta)+"\nNew "+noun+": "+core.FmtMoney(target), m.win)
		m.back()
		if u := m.store.AccountByID(a.ID); u != nil {
			m.showAccount(*u)
		}
	}

	m.pushView(m.formPage("Reconcile "+a.Name, "Reconcile", body, targets, commit, m.back))
}

// editAccount edits Name/Institution/Type/Notes/Budget via core.UpdateAccount,
// then rebuilds the detail page (title may change).
func (m *mobileApp) editAccount(a core.Account) {
	name := widget.NewEntry()
	name.SetText(a.Name)
	inst := widget.NewEntry()
	inst.SetText(a.Institution)
	typeSel := widget.NewSelect([]string{"Asset", "Liability"}, nil)
	typeSel.SetSelected(string(a.Type))
	notes := widget.NewEntry()
	notes.SetText(a.Notes)
	chk := widget.NewCheck("Include this account's balance in the budget", nil)
	chk.SetChecked(!a.OffBudget)

	nameF := field("Name", name)
	instF := field("Institution", inst)
	typeF := field("Type", typeSel)
	notesF := field("Notes", notes)
	budgetF := container.NewVBox(chk, gap(8))
	body := container.NewVBox(nameF, instF, typeF, notesF, budgetF)
	targets := map[fyne.Focusable]fyne.CanvasObject{name: nameF, inst: instF, notes: notesF}

	commit := func() {
		if strings.TrimSpace(name.Text) == "" || typeSel.Selected == "" {
			dialog.ShowError(errors.New("name and type are required"), m.win)
			return
		}
		m.store.UpdateAccount(a.ID, core.Account{
			Name:        name.Text,
			Institution: inst.Text,
			Type:        core.AcctType(typeSel.Selected),
			Notes:       notes.Text,
			OffBudget:   !chk.Checked,
		})
		m.back()
		if u := m.store.AccountByID(a.ID); u != nil {
			m.showAccount(*u)
		}
	}

	m.pushView(m.formPage("Edit account", "Save", body, targets, commit, m.back))
}

// addAccount opens a full-screen form to create a new account (used by the Home
// FAB). On save it returns to Home, which refreshes on the store change.
func (m *mobileApp) addAccount() {
	if m.store == nil {
		return
	}
	name := widget.NewEntry()
	name.SetPlaceHolder("e.g. Checking")
	inst := widget.NewEntry()
	inst.SetPlaceHolder("e.g. Demo Bank")
	typeSel := widget.NewSelect([]string{"Asset", "Liability"}, nil)
	typeSel.SetSelected("Asset")
	notes := widget.NewEntry()
	notes.SetPlaceHolder("Optional")
	chk := widget.NewCheck("Include this account's balance in the budget", nil)
	chk.SetChecked(true)

	nameF := field("Name", name)
	instF := field("Institution", inst)
	notesF := field("Notes", notes)
	body := container.NewVBox(nameF, instF, field("Type", typeSel), notesF, container.NewVBox(chk, gap(8)))
	targets := map[fyne.Focusable]fyne.CanvasObject{name: nameF, inst: instF, notes: notesF}

	commit := func() {
		if strings.TrimSpace(name.Text) == "" || typeSel.Selected == "" {
			dialog.ShowError(errors.New("name and type are required"), m.win)
			return
		}
		m.store.AddAccount(core.Account{
			Name:        name.Text,
			Institution: inst.Text,
			Type:        core.AcctType(typeSel.Selected),
			Notes:       notes.Text,
			OffBudget:   !chk.Checked,
		})
		m.back()
	}

	m.pushView(m.formPage("Add account", "Add", body, targets, commit, m.back))
}

// deleteAccount blocks deletion when the account has transactions; otherwise it
// confirms on a full-screen page and returns to Home on success.
func (m *mobileApp) deleteAccount(a core.Account) {
	if m.store.TxnCountForAccount(a.ID) > 0 {
		dialog.ShowInformation("Cannot delete",
			a.Name+" has transactions. Remove them first.", m.win)
		return
	}
	m.pushPage(func(closeConfirm func()) fyne.CanvasObject {
		del := widget.NewButton("Delete account", func() {
			closeConfirm() // pop the confirm page
			if m.store.DeleteAccount(a.ID) {
				m.back() // pop the account page → Home
			}
		})
		del.Importance = widget.HighImportance
		cancel := widget.NewButton("Cancel", closeConfirm)
		body := container.NewVBox(
			wrapText("Delete "+a.Name+"? This can't be undone."),
			gap(18), del, gap(6), cancel,
		)
		return m.formPage("Delete account?", "", body, nil, nil, closeConfirm)
	})
}
