package mobileui

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// bulkRow is one editable entry in the bulk-add list: a full transaction's worth
// of controls (type, amount, money account, category/to-account, date, payee,
// memo).
type bulkRow struct {
	kind     *widget.Select
	amount   *widget.Entry
	account  *widget.Select
	other    *widget.Select
	otherCap *canvas.Text
	dateOf   func() int
	payee    *widget.Entry
	memo     *widget.Entry
	box      fyne.CanvasObject
}

// openBulkAdd shows a scrollable list of editable transaction rows so a backlog
// can be entered in one pass and committed as a single batch (the mobile
// counterpart of the desktop Quick Add). Blank or incomplete rows are skipped on
// save; each row carries its own date so entries can span several days.
func (m *mobileApp) openBulkAdd() {
	if m.store == nil {
		return
	}
	s := m.store
	moneyNames := accountNames(s.MoneyAccounts())
	defaultMoney := ""
	if len(moneyNames) > 0 {
		defaultMoney = moneyNames[0]
	}

	var rows []*bulkRow
	list := container.NewVBox()

	rebuild := func() {
		objs := make([]fyne.CanvasObject, 0, len(rows)*2)
		for i, r := range rows {
			if i > 0 {
				objs = append(objs, gap(10))
			}
			objs = append(objs, r.box)
		}
		list.Objects = objs
		list.Refresh()
	}

	remove := func(target *bulkRow) {
		kept := rows[:0]
		for _, r := range rows {
			if r != target {
				kept = append(kept, r)
			}
		}
		rows = kept
		rebuild()
	}

	addRow := func() {
		r := &bulkRow{
			kind:    widget.NewSelect([]string{"Expense", "Income", "Transfer"}, nil),
			amount:  newAmountEntry(),
			account: widget.NewSelect(moneyNames, nil),
			other:   widget.NewSelect(nil, nil),
			payee:   widget.NewEntry(),
			memo:    widget.NewEntry(),
		}
		r.payee.SetPlaceHolder("Payee (optional)")
		r.memo.SetPlaceHolder("Memo (optional)")
		r.otherCap = newText("Category", colInkDim, 12, false)
		if defaultMoney != "" {
			r.account.SetSelected(defaultMoney)
		}

		// The "other" leg depends on the type: a category for income/expense, or a
		// second money account for a transfer.
		fillOther := func(k string) {
			switch k {
			case "Transfer":
				r.otherCap.Text = "To account"
				r.other.Options = accountNames(s.MoneyAccounts())
			case "Income":
				r.otherCap.Text = "Category"
				r.other.Options = accountNames(s.IncomeAccounts())
			default:
				r.otherCap.Text = "Category"
				r.other.Options = accountNames(s.ExpenseAccounts())
			}
			r.otherCap.Refresh()
			if len(r.other.Options) > 0 {
				r.other.SetSelected(r.other.Options[0])
			} else {
				r.other.Selected = ""
				r.other.Refresh()
			}
		}
		r.kind.OnChanged = fillOther
		r.kind.SetSelected("Expense")

		dateCtl, dateGet := m.dateField("Date", core.TodaySerial)
		r.dateOf = dateGet

		head := container.NewBorder(nil, nil,
			newText("Entry", colInkFaint, 11, true),
			iconButton(theme.DeleteIcon(), func() { remove(r) }))
		inner := container.NewVBox(
			head,
			container.NewGridWithColumns(2, field("Type", r.kind), field("Amount", r.amount)),
			field("Account", r.account),
			container.NewVBox(r.otherCap, r.other, gap(8)),
			dateCtl,
			field("Payee", r.payee),
			field("Memo", r.memo),
		)
		r.box = container.NewStack(rounded(colCard, 16), insets(inner, 10, 10, 12, 12))
		rows = append(rows, r)
		rebuild()
	}

	addRow() // start with one empty row

	commit := func() {
		var txns []core.Transaction
		for _, r := range rows {
			amt := core.ParseAmount(r.amount.Text)
			money := idByName(s, r.account.Selected)
			serial := r.dateOf()
			if amt <= 0 || money == "" || r.other.Selected == "" || serial == 0 {
				continue // skip blank/incomplete rows
			}
			otherID := idByName(s, r.other.Selected)
			if r.kind.Selected == "Transfer" && (otherID == "" || otherID == money) {
				continue // a transfer needs two different accounts
			}
			txns = append(txns, core.Transaction{
				Date:  serial,
				Payee: r.payee.Text,
				Memo:  r.memo.Text,
				Posts: core.PostingsFor(r.kind.Selected, money, otherID, amt),
			})
		}
		if len(txns) == 0 {
			dialog.ShowError(errors.New("nothing to save — fill in at least one row (amount, account, category)"), m.win)
			return
		}
		n, err := s.AddTransactions(txns)
		if err != nil {
			dialog.ShowError(err, m.win)
			return
		}
		m.back()
		dialog.ShowInformation("Added", pluralCount(n, "transaction")+" added.", m.win)
	}

	addBtn := widget.NewButtonWithIcon("Add row", theme.ContentAddIcon(), addRow)
	addBtn.Importance = widget.LowImportance

	body := container.NewVBox(
		wrapText("Enter several at once. Each row is its own transaction; blank or incomplete rows are skipped."),
		gap(10),
		list,
		gap(10),
		addBtn,
		gap(24),
	)
	m.pushView(m.formPage("Bulk add", "Save all", body, nil, commit, m.back))
}
