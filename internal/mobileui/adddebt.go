package mobileui

import (
	"errors"
	"strconv"
	"strings"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"

	"github.com/raihanstark/financy/internal/core"
)

func (m *mobileApp) addDebt()             { m.debtForm(nil) }
func (m *mobileApp) editDebt(d core.Debt) { m.debtForm(&d) }

// debtForm is the full-screen add/edit debt form. On add it captures the whole
// plan (total/count/dates/frequency) and generates the schedule; on edit it
// changes only the metadata (the schedule is fixed at creation, like desktop).
func (m *mobileApp) debtForm(existing *core.Debt) {
	if m.store == nil {
		return
	}
	s := m.store
	money := accountLabels(s.MoneyAccounts())

	kind := newSelect(core.DebtTypes(), nil)
	kind.SetSelected(core.DebtTypes()[0])
	name := newEntry()
	name.SetPlaceHolder("e.g. iPhone 15")
	lender := newEntry()
	lender.SetPlaceHolder("e.g. Shopee PayLater")
	acct := newSelect(money, nil)
	guardGroupHeaders(acct)
	if first := core.FirstSelectable(money); first != "" {
		acct.SetSelected(first)
	}
	note := newEntry()
	note.SetPlaceHolder("Optional")

	nameF, lenderF, noteF := field("Name", name), field("Lender", lender), field("Note", note)

	if existing != nil {
		kind.SetSelected(existing.Type)
		name.SetText(existing.Name)
		lender.SetText(existing.Lender)
		if a := s.AccountByID(existing.AcctMoney); a != nil {
			acct.SetSelected(core.AccountLabel(*a))
		}
		note.SetText(existing.Note)

		fields := container.NewVBox(
			field("Type", kind), nameF, lenderF, field("Pay from", acct), noteF,
		)
		save := func() {
			if name.Text == "" || acct.Selected == "" {
				dialog.ShowError(errors.New("enter a name and a pay-from account"), m.win)
				return
			}
			nd := *existing
			nd.Name, nd.Type, nd.Lender = name.Text, kind.Selected, lender.Text
			nd.AcctMoney, nd.Note = idForLabel(s, acct.Selected), note.Text
			s.UpdateDebt(existing.ID, nd)
			m.back()
			m.rebuildDebt(existing.ID)
		}
		m.pushView(m.formPage("Edit debt", "Save", fields, save, m.back))
		return
	}

	// Add: full schedule.
	total := newAmountEntry()
	count := newEntry()
	count.SetPlaceHolder("e.g. 12")
	purchaseF, purchaseSerial := m.dateField("Purchase date", core.TodaySerial)
	firstF, firstSerial := m.dateField("First due", core.TodaySerial)
	freq := newSelect(core.Frequencies(), nil)
	freq.SetSelected("Monthly")

	totalF := field("Total amount", total)
	countF := field("Installments", count)

	fields := container.NewVBox(
		field("Type", kind), nameF, lenderF, field("Pay from", acct),
		totalF, countF, purchaseF, firstF, field("Frequency", freq), noteF,
	)
	save := func() {
		amt := core.ParseAmount(total.Text)
		n, _ := strconv.Atoi(strings.TrimSpace(count.Text))
		due := firstSerial()
		purch := purchaseSerial()
		if name.Text == "" || acct.Selected == "" || amt <= 0 || n < 1 || due == 0 {
			dialog.ShowError(errors.New("enter a name, pay-from account, amount, installment count, and a valid first-due date"), m.win)
			return
		}
		if purch == 0 {
			purch = due
		}
		d := core.Debt{
			Name: name.Text, Type: kind.Selected, Lender: lender.Text,
			AcctMoney: idForLabel(s, acct.Selected), PurchaseDate: purch, Note: note.Text,
		}
		s.AddDebt(d, core.GenerateInstallments(amt, n, due, freq.Selected))
		m.back() // back to the Debts tab; the store change refreshes it
	}
	m.pushView(m.formPage("Add debt", "Add", fields, save, m.back))
}
