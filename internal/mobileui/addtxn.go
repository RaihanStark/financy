package mobileui

import (
	"errors"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"

	"github.com/raihanstark/financy/internal/core"
)

// openAdd shows the add-transaction form as a full-screen page (overlays don't
// get keyboard avoidance). Category/To options track the chosen type; postings
// are built by core so double-entry stays correct.
func (m *mobileApp) openAdd(prefill *core.Account) { m.openAddThen(prefill, nil) }

// openAddThen opens the add form; onAdded runs after a successful add instead of
// the default "restore the previous page", so a caller like the account-detail
// page can rebuild itself to show the new transaction.
func (m *mobileApp) openAddThen(prefill *core.Account, onAdded func()) {
	m.txnForm("Add transaction", "Add", prefill, nil, onAdded)
}

// editTransaction opens the same form pre-filled to edit an existing
// transaction (commit calls UpdateTransaction); onDone runs after success.
func (m *mobileApp) editTransaction(t core.Transaction, onDone func()) {
	m.txnForm("Edit transaction", "Save", nil, &t, onDone)
}

// txnForm is the shared add/edit transaction form. existing != nil switches it
// to edit mode (pre-filled; commit updates instead of adds).
func (m *mobileApp) txnForm(title, action string, prefill *core.Account, existing *core.Transaction, onDone func()) {
	if m.store == nil {
		return
	}
	s := m.store

	kind := newSelect([]string{"Expense", "Income", "Transfer"}, nil)
	amount := newAmountEntry()
	account := newSelect(accountNames(s.MoneyAccounts()), nil)
	if prefill != nil {
		account.SetSelected(prefill.Name)
		account.Disable()
	} else if a := s.MoneyAccounts(); len(a) > 0 {
		account.SetSelected(a[0].Name)
	}
	other := newSelect(nil, nil)
	payee := newEntry()
	payee.SetPlaceHolder("Optional")
	initialDate := core.TodaySerial
	if existing != nil {
		initialDate = existing.Date
	}
	dateF, dateSerial := m.dateField("Date", initialDate)
	memo := newEntry()
	memo.SetPlaceHolder("Optional")

	otherLabel := newText("Category", colInkDim, 12, false)
	fillOther := func(k string) {
		switch k {
		case "Transfer":
			otherLabel.Text = "To account"
			other.Options = accountNames(s.MoneyAccounts())
		case "Income":
			otherLabel.Text = "Category"
			other.Options = accountNames(s.IncomeAccounts())
		default:
			otherLabel.Text = "Category"
			other.Options = accountNames(s.ExpenseAccounts())
		}
		otherLabel.Refresh()
		if len(other.Options) > 0 {
			other.SetSelected(other.Options[0])
		} else {
			other.Selected = ""
			other.Refresh()
		}
	}
	kind.OnChanged = fillOther
	kind.SetSelected("Expense")

	if existing != nil {
		if e := deriveEdit(s, *existing); e.ok {
			kind.SetSelected(e.kind) // repopulates the category/to options
			account.SetSelected(e.money)
			other.SetSelected(e.other)
			amount.SetText(core.FmtMoneyInput(e.amount))
		}
		payee.SetText(existing.Payee)
		memo.SetText(existing.Memo)
	}

	commit := func() {
		amt := core.ParseAmount(amount.Text)
		serial := dateSerial()
		if amt <= 0 || serial == 0 || account.Selected == "" || other.Selected == "" {
			dialog.ShowError(errors.New("enter an amount, accounts, and a valid date"), m.win)
			return
		}
		moneyID := idByName(s, account.Selected)
		otherID := idByName(s, other.Selected)
		if kind.Selected == "Transfer" && moneyID == otherID {
			dialog.ShowError(errors.New("choose two different accounts for a transfer"), m.win)
			return
		}
		posts := core.PostingsFor(kind.Selected, moneyID, otherID, amt)
		nt := core.Transaction{Date: serial, Payee: payee.Text, Memo: memo.Text, Posts: posts}
		ok := false
		if existing != nil {
			ok = s.UpdateTransaction(existing.ID, nt)
		} else {
			ok = s.AddTransaction(nt)
		}
		if !ok {
			dialog.ShowError(errors.New("could not save that transaction"), m.win)
			return
		}
		m.back()
		if onDone != nil {
			onDone()
		}
	}

	typeF := field("Type", kind)
	amountF := field("Amount", amount)
	accountF := field("Account", account)
	otherF := container.NewVBox(otherLabel, other, gap(8))
	payeeF := field("Payee", payee)
	memoF := field("Memo", memo)

	fields := container.NewVBox(typeF, amountF, accountF, otherF, payeeF, dateF, memoF)
	m.pushView(m.formPage(title, action, fields, commit, m.back))
}

func accountNames(accts []core.Account) []string {
	out := make([]string, len(accts))
	for i, a := range accts {
		out[i] = a.Name
	}
	return out
}

func idByName(s *core.Store, name string) string {
	if a := s.AccountByName(name); a != nil {
		return a.ID
	}
	return ""
}
