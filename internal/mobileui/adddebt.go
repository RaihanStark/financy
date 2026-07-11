package mobileui

import (
	"errors"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

func (m *mobileApp) addDebt()             { m.debtForm(nil) }
func (m *mobileApp) editDebt(d core.Debt) { m.debtForm(&d) }

// debtForm is the full-screen add/edit debt form, type-driven like the
// desktop: pick a kind and only its fields appear, with a live plan preview
// (computed payment or term, payoff date, total interest) before saving. On
// edit the kind is fixed; a loan's terms stay editable (its unpaid schedule
// regenerates), a card's profile too.
func (m *mobileApp) debtForm(existing *core.Debt) {
	if m.store == nil {
		return
	}

	if existing != nil {
		fields, save := m.debtKindFields(existing.Type, existing)
		m.pushView(m.formPage("Edit debt", "Save", fields, func() {
			if save() {
				m.back()
				m.rebuildDebt(existing.ID)
			}
		}, m.back))
		return
	}

	kind := newSelect(core.DebtTypes(), nil)
	dyn := container.NewVBox()
	save := func() bool { return false }
	kind.OnChanged = func(k string) {
		var fields fyne.CanvasObject
		fields, save = m.debtKindFields(k, nil)
		dyn.Objects = []fyne.CanvasObject{fields}
		dyn.Refresh()
	}
	kind.SetSelected(core.DebtBNPL)

	body := container.NewVBox(field("Type", kind), dyn)
	m.pushView(m.formPage("Add debt", "Add", body, func() {
		if save() {
			m.back() // back to the Debts tab; the store change refreshes it
		}
	}, m.back))
}

// debtKindFields assembles one kind's form and its save closure (returns
// whether the save succeeded).
func (m *mobileApp) debtKindFields(kind string, existing *core.Debt) (fyne.CanvasObject, func() bool) {
	switch kind {
	case core.DebtLoan:
		return m.loanFormFields(existing)
	case core.DebtRevolving:
		return m.revolvingFormFields(existing)
	case core.DebtInformal:
		return m.informalFormFields(existing)
	default:
		return m.bnplFormFields(existing)
	}
}

// debtCommonFields are the entries every kind shares.
type debtCommonFields struct {
	name, lender, note *scrollEntry
	acct               *scrollSelect
}

func (m *mobileApp) debtCommon(namePH, lenderPH string, existing *core.Debt) debtCommonFields {
	money := accountLabels(m.store.MoneyAccounts())
	c := debtCommonFields{
		name: newEntry(), lender: newEntry(), note: newEntry(),
		acct: newSelect(money, nil),
	}
	c.name.SetPlaceHolder(namePH)
	c.lender.SetPlaceHolder(lenderPH)
	c.note.SetPlaceHolder("Optional")
	guardGroupHeaders(c.acct)
	if first := core.FirstSelectable(money); first != "" {
		c.acct.SetSelected(first)
	}
	if existing != nil {
		c.name.SetText(existing.Name)
		c.lender.SetText(existing.Lender)
		c.note.SetText(existing.Note)
		if a := m.store.AccountByID(existing.AcctMoney); a != nil {
			c.acct.SetSelected(core.AccountLabel(*a))
		}
	}
	return c
}

func (c debtCommonFields) valid() bool { return c.name.Text != "" && c.acct.Selected != "" }

func (c debtCommonFields) apply(s *core.Store, d *core.Debt) {
	d.Name, d.Lender, d.Note = c.name.Text, c.lender.Text, c.note.Text
	d.AcctMoney = idForLabel(s, c.acct.Selected)
}

func (m *mobileApp) debtFormError(msg string) { dialog.ShowError(errors.New(msg), m.win) }

// ---- BNPL ----

func (m *mobileApp) bnplFormFields(existing *core.Debt) (fyne.CanvasObject, func() bool) {
	s := m.store
	c := m.debtCommon("e.g. iPhone 15", "e.g. Shopee PayLater", existing)
	base := container.NewVBox(
		field("Name", c.name), field("Lender", c.lender), field("Pay from", c.acct),
	)

	if existing != nil {
		save := func() bool {
			if !c.valid() {
				m.debtFormError("enter a name and a pay-from account")
				return false
			}
			nd := *existing
			c.apply(s, &nd)
			s.UpdateDebt(existing.ID, nd)
			return true
		}
		base.Add(field("Note", c.note))
		return base, save
	}

	total := newAmountEntry()
	count := newEntry()
	count.SetPlaceHolder("e.g. 12")
	purchaseF, purchaseSerial := m.dateField("Purchase date", core.TodaySerial)
	firstF, firstSerial := m.dateField("First due", core.TodaySerial)
	freq := newSelect(core.Frequencies(), nil)
	freq.SetSelected("Monthly")

	preview := newText("—", colInkDim, 12, false)
	update := func() {
		amt := core.ParseAmount(total.Text)
		n, _ := strconv.Atoi(strings.TrimSpace(count.Text))
		if amt <= 0 || n < 1 {
			preview.Text = "—"
		} else {
			insts := core.GenerateInstallments(amt, n, firstSerial(), freq.Selected)
			preview.Text = strconv.Itoa(n) + " × " + core.FmtMoney(insts[0].Amount) +
				" · last due " + core.FmtSerialDate(insts[len(insts)-1].DueDate)
		}
		preview.Refresh()
	}
	total.OnChanged = func(string) { update() }
	count.OnChanged = func(string) { update() }
	freq.OnChanged = func(string) { update() }

	base.Add(field("Total amount", total))
	base.Add(field("Installments", count))
	base.Add(purchaseF)
	base.Add(firstF)
	base.Add(field("Frequency", freq))
	base.Add(field("Plan", insets(preview, 4, 4, 2, 2)))
	base.Add(field("Note", c.note))

	save := func() bool {
		amt := core.ParseAmount(total.Text)
		n, _ := strconv.Atoi(strings.TrimSpace(count.Text))
		due := firstSerial()
		if !c.valid() || amt <= 0 || n < 1 || due == 0 {
			m.debtFormError("enter a name, pay-from account, amount, installment count, and a valid first-due date")
			return false
		}
		d := core.Debt{Type: core.DebtBNPL, PurchaseDate: purchaseSerial()}
		c.apply(s, &d)
		s.AddDebt(d, core.GenerateInstallments(amt, n, due, freq.Selected))
		return true
	}
	return base, save
}

// ---- Loan ----

const (
	loanByTerm    = "By term (compute the payment)"
	loanByPayment = "By payment (compute the term)"
)

func (m *mobileApp) loanFormFields(existing *core.Debt) (fyne.CanvasObject, func() bool) {
	s := m.store
	c := m.debtCommon("e.g. Car loan", "e.g. BCA", existing)
	apr := newEntry()
	apr.SetPlaceHolder("e.g. 12.5")
	if existing != nil && existing.APRBps > 0 {
		apr.SetText(strconv.FormatFloat(float64(existing.APRBps)/100, 'f', -1, 64))
	}
	freq := newSelect(core.Frequencies(), nil)
	if existing != nil && existing.Freq != "" {
		freq.SetSelected(existing.Freq)
	} else {
		freq.SetSelected("Monthly")
	}

	base := container.NewVBox(
		field("Name", c.name), field("Lender", c.lender), field("Pay from", c.acct),
	)

	if existing != nil {
		payment := newAmountEntry()
		payment.SetText(core.FmtMoneyInput(existing.PaymentAmount))
		base.Add(field("APR %", apr))
		base.Add(field("Payment", payment))
		base.Add(field("Frequency", freq))
		base.Add(field("Note", c.note))
		base.Add(insets(wrapText("Changing APR, payment, or frequency regenerates the unpaid schedule."), 4, 0, 2, 2))
		save := func() bool {
			pay := core.ParseAmount(payment.Text)
			if !c.valid() || pay <= 0 {
				m.debtFormError("enter a name, pay-from account, and a payment above zero")
				return false
			}
			nd := *existing
			c.apply(s, &nd)
			nd.APRBps = core.ParseAPRBps(apr.Text)
			nd.PaymentAmount = pay
			nd.Freq = freq.Selected
			s.UpdateDebt(existing.ID, nd)
			return true
		}
		return base, save
	}

	principal := newAmountEntry()
	asOfF, asOfSerial := m.dateField("As of", core.TodaySerial)
	firstF, firstSerial := m.dateField("First due", core.TodaySerial)
	mode := newSelect([]string{loanByTerm, loanByPayment}, nil)
	mode.SetSelected(loanByTerm)
	term := newEntry()
	term.SetText("12")
	payment := newAmountEntry()

	preview := newText("—", colInkDim, 12, false)
	solve := func() ([]core.Installment, int) {
		p := core.ParseAmount(principal.Text)
		if p <= 0 {
			return nil, 0
		}
		bps := core.ParseAPRBps(apr.Text)
		pay := core.ParseAmount(payment.Text)
		if mode.Selected == loanByTerm {
			n, _ := strconv.Atoi(strings.TrimSpace(term.Text))
			if n <= 0 {
				return nil, 0
			}
			pay = core.AmortizedPayment(p, bps, n, freq.Selected)
		}
		if pay <= 0 {
			return nil, 0
		}
		insts, ok := core.GenerateLoanSchedule(p, bps, pay, firstSerial(), freq.Selected)
		if !ok {
			return nil, pay
		}
		return insts, pay
	}
	update := func() {
		insts, pay := solve()
		switch {
		case insts == nil && pay > 0:
			preview.Text = "payment doesn't cover the interest"
			preview.Color = colNeg
		case insts == nil:
			preview.Text = "—"
			preview.Color = colInkDim
		default:
			preview.Text = core.FmtMoney(pay) + " × " + strconv.Itoa(len(insts)) +
				" · payoff " + core.FmtSerialDate(insts[len(insts)-1].DueDate) +
				" · interest " + core.FmtMoney(core.ScheduleInterestTotal(insts))
			preview.Color = colInkDim
		}
		preview.Refresh()
	}
	for _, e := range []*scrollEntry{principal, apr, term, payment} {
		e.OnChanged = func(string) { update() }
	}
	mode.OnChanged = func(string) { update() }
	freq.OnChanged = func(string) { update() }

	base.Add(field("Principal owed", principal))
	base.Add(asOfF)
	base.Add(field("APR %", apr))
	base.Add(field("Set up", mode))
	base.Add(field("Term (payments)", term))
	base.Add(field("Payment", payment))
	base.Add(firstF)
	base.Add(field("Frequency", freq))
	base.Add(field("Plan", insets(preview, 4, 4, 2, 2)))
	base.Add(field("Note", c.note))

	save := func() bool {
		insts, pay := solve()
		if !c.valid() || insts == nil {
			m.debtFormError("enter a name, pay-from account, principal, and a plan that pays the loan off")
			return false
		}
		d := core.Debt{
			Type: core.DebtLoan, PurchaseDate: asOfSerial(),
			APRBps: core.ParseAPRBps(apr.Text), Freq: freq.Selected, PaymentAmount: pay,
		}
		c.apply(s, &d)
		s.AddDebt(d, insts)
		return true
	}
	return base, save
}

// ---- Revolving ----

const newCardAccountOption = "➕ Create a new account"

func (m *mobileApp) revolvingFormFields(existing *core.Debt) (fyne.CanvasObject, func() bool) {
	s := m.store
	c := m.debtCommon("e.g. Visa Platinum", "e.g. BigBank", existing)
	apr := newEntry()
	apr.SetPlaceHolder("e.g. 19.99")
	limit := newAmountEntry()
	stmtDay := newEntry()
	stmtDay.SetPlaceHolder("1–28")
	dueDay := newEntry()
	dueDay.SetPlaceHolder("1–28")
	minPct := newEntry()
	minPct.SetPlaceHolder("e.g. 5")
	minFloor := newAmountEntry()
	if existing != nil {
		if existing.APRBps > 0 {
			apr.SetText(strconv.FormatFloat(float64(existing.APRBps)/100, 'f', -1, 64))
		}
		limit.SetText(core.FmtMoneyInput(existing.CreditLimit))
		stmtDay.SetText(strconv.Itoa(existing.StatementDay))
		dueDay.SetText(strconv.Itoa(existing.PayDueDay))
		if existing.MinPayBps > 0 {
			minPct.SetText(strconv.FormatFloat(float64(existing.MinPayBps)/100, 'f', -1, 64))
		}
		minFloor.SetText(core.FmtMoneyInput(existing.MinPayFloor))
	} else {
		stmtDay.SetText("1")
		dueDay.SetText("15")
	}

	base := container.NewVBox(
		field("Name", c.name), field("Issuer", c.lender), field("Pay from", c.acct),
	)

	// Attach an existing card account, or create a fresh one with a balance.
	var acctSel *scrollSelect
	balance := newAmountEntry()
	if existing == nil {
		opts := []string{newCardAccountOption}
		for _, a := range s.LiabilityAccounts() {
			if !m.debtBoundAccount(a.ID) {
				opts = append(opts, core.AccountLabel(a))
			}
		}
		acctSel = newSelect(opts, nil)
		acctSel.SetSelected(newCardAccountOption)
		base.Add(field("Card account", acctSel))
		base.Add(field("Current balance", balance))
	}

	base.Add(field("APR %", apr))
	base.Add(field("Credit limit", limit))
	base.Add(field("Statement day", stmtDay))
	base.Add(field("Payment due day", dueDay))
	base.Add(field("Min payment %", minPct))
	base.Add(field("Min payment floor", minFloor))
	base.Add(field("Note", c.note))

	save := func() bool {
		sd, _ := strconv.Atoi(strings.TrimSpace(stmtDay.Text))
		dd, _ := strconv.Atoi(strings.TrimSpace(dueDay.Text))
		if !c.valid() || sd < 1 || sd > 28 {
			m.debtFormError("enter a name, pay-from account, and a statement day between 1 and 28")
			return false
		}
		d := core.Debt{
			Type:         core.DebtRevolving,
			APRBps:       core.ParseAPRBps(apr.Text),
			CreditLimit:  core.ParseAmount(limit.Text),
			StatementDay: sd, PayDueDay: dd,
			MinPayBps:   core.ParseAPRBps(minPct.Text),
			MinPayFloor: core.ParseAmount(minFloor.Text),
		}
		c.apply(s, &d)
		if existing != nil {
			s.UpdateDebt(existing.ID, d)
			return true
		}
		if acctSel.Selected != newCardAccountOption {
			d.AcctLiability = idForLabel(s, acctSel.Selected)
		} else {
			d.Principal = core.ParseAmount(balance.Text)
		}
		s.AddDebt(d, nil)
		return true
	}
	return base, save
}

// debtBoundAccount reports whether a liability account already backs a debt.
func (m *mobileApp) debtBoundAccount(acctID string) bool {
	for _, d := range m.store.Debts() {
		if d.AcctLiability == acctID {
			return true
		}
	}
	return false
}

// ---- Informal ----

func (m *mobileApp) informalFormFields(existing *core.Debt) (fyne.CanvasObject, func() bool) {
	s := m.store
	c := m.debtCommon("e.g. Loan from Andi", "who you owe", existing)
	amount := newAmountEntry()
	hasDue := widget.NewCheck("Has a due date", nil)
	dueF, dueSerial := m.dateField("Due date", core.TodaySerial)
	dueBox := container.NewVBox()
	hasDue.OnChanged = func(on bool) {
		dueBox.Objects = nil
		if on {
			dueBox.Objects = []fyne.CanvasObject{dueF}
		}
		dueBox.Refresh()
	}
	if existing != nil {
		amount.SetText(core.FmtMoneyInput(existing.Principal))
		if existing.DueDate > 0 {
			hasDue.SetChecked(true)
		}
	}

	base := container.NewVBox(
		field("Name", c.name), field("Owed to", c.lender), field("Pay from", c.acct),
		field("Amount owed", amount),
		insets(hasDue, 4, 0, 2, 2), dueBox,
	)

	// New IOUs can record where the borrowed money landed, so an actual cash
	// loan raises your asset and the debt together.
	var originSel *scrollSelect
	if existing == nil {
		opts := []string{"— nowhere (a purchase, a favor) —"}
		opts = append(opts, accountLabels(s.AssetAccounts())...)
		originSel = newSelect(opts, nil)
		guardGroupHeaders(originSel)
		originSel.SetSelected(opts[0])
		base.Add(field("Deposited to", originSel))
	}
	base.Add(field("Note", c.note))

	save := func() bool {
		amt := core.ParseAmount(amount.Text)
		if !c.valid() || amt <= 0 {
			m.debtFormError("enter a name, pay-from account, and the amount owed")
			return false
		}
		d := core.Debt{Type: core.DebtInformal, Principal: amt}
		c.apply(s, &d)
		if hasDue.Checked {
			d.DueDate = dueSerial()
		}
		if existing != nil {
			d.AcctOrigin = existing.AcctOrigin
			s.UpdateDebt(existing.ID, d)
			return true
		}
		if originSel != nil && originSel.Selected != originSel.Options[0] {
			d.AcctOrigin = idForLabel(s, originSel.Selected)
		}
		s.AddDebt(d, nil)
		return true
	}
	return base, save
}
