package view

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// DebtForm is the add/edit dialog. It's type-driven: pick the debt kind first
// and only that kind's fields appear, with a live preview (computed payment,
// payoff date, total interest, schedule head) before anything is saved. On
// edit the kind is fixed; terms stay editable (a loan regenerates its unpaid
// schedule when APR / payment / frequency change).
func DebtForm(existing *Debt) {
	if win == nil {
		return
	}
	if existing != nil {
		editDebtForm(*existing)
		return
	}

	kind := widget.NewSelect(debtTypes(), nil)
	body := container.NewVBox()
	footer := container.NewVBox()

	var dlg *dialog.CustomDialog
	rebuild := func(k string) {
		f := newDebtFields(k, nil)
		body.Objects = []fyne.CanvasObject{f.grid()}
		body.Refresh()

		cancel := secondaryButton("Cancel", nil, func() { dlg.Hide() })
		save := primaryButton("Save", theme.ConfirmIcon(), func() {
			if f.commit() {
				dlg.Hide()
			}
		})
		footer.Objects = []fyne.CanvasObject{container.NewHBox(layout.NewSpacer(), cancel, save)}
		footer.Refresh()
	}
	kind.OnChanged = rebuild
	kind.SetSelected(DebtBNPL)

	head := container.New(layout.NewFormLayout(), widget.NewLabel("Type"), kind)
	content := container.NewVBox(head, spacerH(4), body, spacerH(8), footer)
	dlg = dialog.NewCustomWithoutButtons("Add Debt", container.NewScroll(content), win)
	dlg.Resize(fyne.NewSize(520, 560))
	dlg.Show()
}

func editDebtForm(d Debt) {
	f := newDebtFields(d.Type, &d)
	var dlg *dialog.CustomDialog
	cancel := secondaryButton("Cancel", nil, func() { dlg.Hide() })
	save := primaryButton("Save", theme.ConfirmIcon(), func() {
		if f.commit() {
			dlg.Hide()
		}
	})
	content := container.NewVBox(
		container.New(layout.NewFormLayout(), widget.NewLabel("Type"), widget.NewLabel(d.Type)),
		spacerH(4), f.grid(), spacerH(8),
		container.NewHBox(layout.NewSpacer(), cancel, save),
	)
	dlg = dialog.NewCustomWithoutButtons("Edit Debt", container.NewScroll(content), win)
	dlg.Resize(fyne.NewSize(520, 520))
	dlg.Show()
}

// debtFields is one kind's assembled form: its items and the commit closure
// that validates and saves. grid() lays the items out like a widget form.
type debtFields struct {
	items  []*widget.FormItem
	commit func() bool
}

func (f *debtFields) grid() fyne.CanvasObject {
	g := container.New(layout.NewFormLayout())
	for _, it := range f.items {
		g.Add(widget.NewLabel(it.Text))
		g.Add(it.Widget)
	}
	return g
}

func newDebtFields(kind string, existing *Debt) *debtFields {
	switch kind {
	case DebtLoan:
		return loanFields(existing)
	case DebtRevolving:
		return revolvingFields(existing)
	case DebtInformal:
		return informalFields(existing)
	default:
		return bnplFields(existing)
	}
}

// ---- shared field builders ----

func debtNameEntry(placeholder string, existing *Debt) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	if existing != nil {
		e.SetText(existing.Name)
	}
	return e
}

func debtLenderEntry(placeholder string, existing *Debt) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	if existing != nil {
		e.SetText(existing.Lender)
	}
	return e
}

func debtMoneySelect(existing *Debt) *widget.Select {
	sel := widget.NewSelect(groupedLabels(store.MoneyAccounts()), nil)
	guardGroupHeaders(sel)
	if existing != nil {
		sel.SetSelected(labelOf(existing.AcctMoney))
	}
	return sel
}

func debtNoteEntry(existing *Debt) *widget.Entry {
	e := widget.NewEntry()
	if existing != nil {
		e.SetText(existing.Note)
	}
	return e
}

func aprEntry(existing *Debt) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder("e.g. 12.5")
	if existing != nil && existing.APRBps > 0 {
		e.SetText(strconv.FormatFloat(float64(existing.APRBps)/100, 'f', -1, 64))
	}
	return e
}

func dayEntry(def int) *widget.Entry {
	e := widget.NewEntry()
	if def >= 1 {
		e.SetText(itoa(def))
	}
	e.SetPlaceHolder("1–28")
	return e
}

// ---- BNPL ----

func bnplFields(existing *Debt) *debtFields {
	name := debtNameEntry("e.g. iPhone 15", existing)
	lender := debtLenderEntry("e.g. Shopee PayLater", existing)
	acctMoney := debtMoneySelect(existing)
	note := debtNoteEntry(existing)

	if existing != nil {
		// Edit: metadata only — the schedule is edited per installment.
		return &debtFields{
			items: []*widget.FormItem{
				widget.NewFormItem("Name", name),
				widget.NewFormItem("Lender", lender),
				widget.NewFormItem("Pay from", acctMoney),
				widget.NewFormItem("Note", note),
			},
			commit: func() bool {
				if name.Text == "" || acctMoney.Selected == "" {
					return false
				}
				upd := *existing
				upd.Name, upd.Lender, upd.Note = name.Text, lender.Text, note.Text
				upd.AcctMoney = idOf(acctMoney.Selected)
				store.UpdateDebt(existing.ID, upd)
				return true
			},
		}
	}

	total := newAmountEntry()
	total.Validator = amountValidator
	count := widget.NewEntry()
	count.SetText("12")
	purchase := newDateEntry(todaySerial)
	first := newDateEntry(todaySerial)
	freq := widget.NewSelect(frequencies(), nil)
	freq.SetSelected("Monthly")

	preview := widget.NewLabel("—")
	update := func() {
		amt := parseAmount(total.Text)
		n, _ := strconv.Atoi(count.Text)
		due := parseDateSerial(first.Text)
		if amt <= 0 || n <= 0 || due == 0 {
			preview.SetText("—")
			return
		}
		insts := generateInstallments(amt, n, due, freq.Selected)
		preview.SetText(itoa(n) + " × " + fmtMoney(insts[0].Amount) +
			" · last due " + fmtSerialDate(insts[len(insts)-1].DueDate))
	}
	for _, w := range []*widget.Entry{total, count, first} {
		w.OnChanged = func(string) { update() }
	}
	freq.OnChanged = func(string) { update() }
	update()

	return &debtFields{
		items: []*widget.FormItem{
			widget.NewFormItem("Name", name),
			widget.NewFormItem("Lender", lender),
			widget.NewFormItem("Pay from", acctMoney),
			widget.NewFormItem("Total amount", total),
			widget.NewFormItem("Installments", count),
			widget.NewFormItem("Purchase date", purchase),
			widget.NewFormItem("First due", first),
			widget.NewFormItem("Frequency", freq),
			widget.NewFormItem("Plan", preview),
			widget.NewFormItem("Note", note),
		},
		commit: func() bool {
			amt := parseAmount(total.Text)
			n, _ := strconv.Atoi(count.Text)
			due := parseDateSerial(first.Text)
			bought := parseDateSerial(purchase.Text)
			if name.Text == "" || acctMoney.Selected == "" ||
				amt <= 0 || n <= 0 || due == 0 || bought == 0 {
				return false
			}
			store.AddDebt(Debt{
				Name: name.Text, Type: DebtBNPL, Lender: lender.Text,
				AcctMoney:    idOf(acctMoney.Selected),
				PurchaseDate: bought,
				Note:         note.Text,
			}, generateInstallments(amt, n, due, freq.Selected))
			return true
		},
	}
}

// ---- Loan ----

const (
	loanByTerm    = "By term (payment is computed)"
	loanByPayment = "By payment (term is computed)"
)

func loanFields(existing *Debt) *debtFields {
	name := debtNameEntry("e.g. Car loan", existing)
	lender := debtLenderEntry("e.g. BCA", existing)
	acctMoney := debtMoneySelect(existing)
	note := debtNoteEntry(existing)
	apr := aprEntry(existing)

	catNames := groupedLabels(store.ExpenseAccounts())
	interestSel := widget.NewSelect(catNames, nil)
	guardGroupHeaders(interestSel)
	interestID := "loanfees"
	if existing != nil && existing.AcctInterest != "" {
		interestID = existing.AcctInterest
	}
	if lbl := labelOf(interestID); lbl != "" {
		interestSel.SetSelected(lbl)
	}

	if existing != nil {
		// Edit: metadata + terms. Changing APR / payment / frequency regenerates
		// the unpaid schedule (UpdateDebt handles it); warn inline.
		payment := newAmountEntry()
		payment.SetText(fmtMoneyInput(existing.PaymentAmount))
		freq := widget.NewSelect(frequencies(), nil)
		freq.SetSelected(orMonthly(existing.Freq))
		warn := widget.NewLabel("Changing APR, payment, or frequency regenerates the unpaid schedule.")
		warn.Wrapping = fyne.TextWrapWord

		return &debtFields{
			items: []*widget.FormItem{
				widget.NewFormItem("Name", name),
				widget.NewFormItem("Lender", lender),
				widget.NewFormItem("Pay from", acctMoney),
				widget.NewFormItem("APR %", apr),
				widget.NewFormItem("Payment", payment),
				widget.NewFormItem("Frequency", freq),
				widget.NewFormItem("Interest category", interestSel),
				widget.NewFormItem("Note", note),
				widget.NewFormItem("", warn),
			},
			commit: func() bool {
				pay := parseAmount(payment.Text)
				if name.Text == "" || acctMoney.Selected == "" || pay <= 0 {
					return false
				}
				upd := *existing
				upd.Name, upd.Lender, upd.Note = name.Text, lender.Text, note.Text
				upd.AcctMoney = idOf(acctMoney.Selected)
				upd.APRBps = parseAPRBps(apr.Text)
				upd.PaymentAmount = pay
				upd.Freq = freq.Selected
				upd.AcctInterest = idOf(interestSel.Selected)
				store.UpdateDebt(existing.ID, upd)
				return true
			},
		}
	}

	principal := newAmountEntry()
	principal.Validator = amountValidator
	asOf := newDateEntry(todaySerial)
	first := newDateEntry(todaySerial)
	freq := widget.NewSelect(frequencies(), nil)
	freq.SetSelected("Monthly")
	mode := widget.NewSelect([]string{loanByTerm, loanByPayment}, nil)
	mode.SetSelected(loanByTerm)
	term := widget.NewEntry()
	term.SetText("12")
	term.SetPlaceHolder("number of payments")
	payment := newAmountEntry()

	preview := widget.NewLabel("—")
	preview.Wrapping = fyne.TextWrapWord

	// solve returns the schedule the current inputs imply, or nil.
	solve := func() ([]Installment, int) {
		p := parseAmount(principal.Text)
		due := parseDateSerial(first.Text)
		bps := parseAPRBps(apr.Text)
		if p <= 0 || due == 0 {
			return nil, 0
		}
		pay := parseAmount(payment.Text)
		if mode.Selected == loanByTerm {
			n, _ := strconv.Atoi(term.Text)
			if n <= 0 {
				return nil, 0
			}
			pay = amortizedPayment(p, bps, n, freq.Selected)
		}
		if pay <= 0 {
			return nil, 0
		}
		insts, ok := generateLoanSchedule(p, bps, pay, due, freq.Selected)
		if !ok {
			return nil, pay
		}
		return insts, pay
	}
	update := func() {
		term.Disable()
		payment.Disable()
		if mode.Selected == loanByTerm {
			term.Enable()
		} else {
			payment.Enable()
		}
		insts, pay := solve()
		if insts == nil {
			if pay > 0 {
				preview.SetText("That payment doesn't cover the interest — the loan would never be paid off.")
			} else {
				preview.SetText("—")
			}
			return
		}
		last := insts[len(insts)-1]
		preview.SetText(fmtMoney(pay) + " × " + itoa(len(insts)) +
			" · payoff " + fmtSerialDate(last.DueDate) +
			" · total interest " + fmtMoney(scheduleInterest(insts)))
	}
	for _, w := range []*widget.Entry{principal, apr, term, payment, first} {
		w.OnChanged = func(string) { update() }
	}
	mode.OnChanged = func(string) { update() }
	freq.OnChanged = func(string) { update() }
	update()

	return &debtFields{
		items: []*widget.FormItem{
			widget.NewFormItem("Name", name),
			widget.NewFormItem("Lender", lender),
			widget.NewFormItem("Pay from", acctMoney),
			widget.NewFormItem("Principal owed", principal),
			widget.NewFormItem("As of", asOf),
			widget.NewFormItem("APR %", apr),
			widget.NewFormItem("Set up", mode),
			widget.NewFormItem("Term", term),
			widget.NewFormItem("Payment", payment),
			widget.NewFormItem("First due", first),
			widget.NewFormItem("Frequency", freq),
			widget.NewFormItem("Interest category", interestSel),
			widget.NewFormItem("Plan", preview),
			widget.NewFormItem("Note", note),
		},
		commit: func() bool {
			insts, pay := solve()
			bought := parseDateSerial(asOf.Text)
			if name.Text == "" || acctMoney.Selected == "" || insts == nil || bought == 0 {
				return false
			}
			store.AddDebt(Debt{
				Name: name.Text, Type: DebtLoan, Lender: lender.Text,
				AcctMoney:     idOf(acctMoney.Selected),
				PurchaseDate:  bought,
				APRBps:        parseAPRBps(apr.Text),
				Freq:          freq.Selected,
				PaymentAmount: pay,
				AcctInterest:  idOf(interestSel.Selected),
				Note:          note.Text,
			}, insts)
			return true
		},
	}
}

// ---- Revolving ----

const newCardAccountOption = "➕ Create a new account"

func revolvingFields(existing *Debt) *debtFields {
	name := debtNameEntry("e.g. Visa Platinum", existing)
	lender := debtLenderEntry("e.g. BigBank", existing)
	acctMoney := debtMoneySelect(existing)
	note := debtNoteEntry(existing)
	apr := aprEntry(existing)
	limit := newAmountEntry()
	stmtDay := dayEntry(1)
	dueDay := dayEntry(15)
	minPct := widget.NewEntry()
	minPct.SetPlaceHolder("e.g. 5")
	minFloor := newAmountEntry()
	if existing != nil {
		limit.SetText(fmtMoneyInput(existing.CreditLimit))
		stmtDay.SetText(itoa(existing.StatementDay))
		dueDay.SetText(itoa(existing.PayDueDay))
		if existing.MinPayBps > 0 {
			minPct.SetText(strconv.FormatFloat(float64(existing.MinPayBps)/100, 'f', -1, 64))
		}
		minFloor.SetText(fmtMoneyInput(existing.MinPayFloor))
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Issuer", lender),
		widget.NewFormItem("Pay from", acctMoney),
	}

	// Attach an existing card account, or create a fresh one with a balance.
	var acctSel *widget.Select
	balance := newAmountEntry()
	if existing == nil {
		opts := []string{newCardAccountOption}
		for _, a := range store.LiabilityAccounts() {
			if !debtBoundAccount(a.ID) {
				opts = append(opts, accountLabel(a))
			}
		}
		acctSel = widget.NewSelect(opts, nil)
		acctSel.SetSelected(newCardAccountOption)
		acctSel.OnChanged = func(v string) {
			if v == newCardAccountOption {
				balance.Enable()
			} else {
				balance.Disable()
			}
		}
		items = append(items,
			widget.NewFormItem("Card account", acctSel),
			widget.NewFormItem("Current balance", balance),
		)
	}

	preview := widget.NewLabel("—")
	update := func() {
		bal := parseAmount(balance.Text)
		if existing != nil {
			bal = store.DebtBalance(existing.ID)
		} else if acctSel != nil && acctSel.Selected != newCardAccountOption {
			bal = -store.Balance(idOf(acctSel.Selected))
		}
		bps := parseAPRBps(apr.Text)
		minBps := parseAPRBps(minPct.Text)
		min := int((int64(bal)*int64(minBps) + 5000) / 10000)
		if f := parseAmount(minFloor.Text); min < f {
			min = f
		}
		if min > bal {
			min = bal
		}
		if bal <= 0 {
			preview.SetText("—")
			return
		}
		preview.SetText("min payment " + fmtMoney(min) +
			" · ≈" + fmtMoney(periodInterest(bal, bps, 12)) + " interest/month")
	}
	for _, w := range []*widget.Entry{apr, minPct, minFloor, balance} {
		w.OnChanged = func(string) { update() }
	}
	if acctSel != nil {
		prev := acctSel.OnChanged
		acctSel.OnChanged = func(v string) {
			if prev != nil {
				prev(v)
			}
			update()
		}
	}
	update()

	items = append(items,
		widget.NewFormItem("APR %", apr),
		widget.NewFormItem("Credit limit", limit),
		widget.NewFormItem("Statement day", stmtDay),
		widget.NewFormItem("Payment due day", dueDay),
		widget.NewFormItem("Min payment %", minPct),
		widget.NewFormItem("Min payment floor", minFloor),
		widget.NewFormItem("Preview", preview),
		widget.NewFormItem("Note", note),
	)

	return &debtFields{
		items: items,
		commit: func() bool {
			sd, _ := strconv.Atoi(stmtDay.Text)
			dd, _ := strconv.Atoi(dueDay.Text)
			if name.Text == "" || acctMoney.Selected == "" || sd < 1 || sd > 28 {
				return false
			}
			d := Debt{
				Name: name.Text, Type: DebtRevolving, Lender: lender.Text,
				AcctMoney:    idOf(acctMoney.Selected),
				APRBps:       parseAPRBps(apr.Text),
				CreditLimit:  parseAmount(limit.Text),
				StatementDay: sd, PayDueDay: dd,
				MinPayBps:   parseAPRBps(minPct.Text),
				MinPayFloor: parseAmount(minFloor.Text),
				Note:        note.Text,
			}
			if existing != nil {
				store.UpdateDebt(existing.ID, d)
				return true
			}
			if acctSel.Selected != newCardAccountOption {
				d.AcctLiability = idOf(acctSel.Selected)
			} else {
				d.Principal = parseAmount(balance.Text)
			}
			store.AddDebt(d, nil)
			return true
		},
	}
}

// debtBoundAccount reports whether a liability account already backs a debt.
func debtBoundAccount(acctID string) bool {
	for _, d := range store.Debts() {
		if d.AcctLiability == acctID {
			return true
		}
	}
	return false
}

// ---- Informal ----

func informalFields(existing *Debt) *debtFields {
	name := debtNameEntry("e.g. Loan from Andi", existing)
	lender := debtLenderEntry("who you owe", existing)
	acctMoney := debtMoneySelect(existing)
	note := debtNoteEntry(existing)
	amount := newAmountEntry()
	amount.Validator = amountValidator
	dueSerial := 0
	if existing != nil {
		amount.SetText(fmtMoneyInput(existing.Principal))
		dueSerial = existing.DueDate
	}
	due := newDateEntry(dueSerial)
	due.SetPlaceHolder("optional")

	items := []*widget.FormItem{
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Owed to", lender),
		widget.NewFormItem("Pay from", acctMoney),
		widget.NewFormItem("Amount owed", amount),
		widget.NewFormItem("Due date", due),
	}

	// New IOUs can record where the borrowed money landed, so an actual cash
	// loan raises your asset and the debt together.
	var originSel *widget.Select
	if existing == nil {
		opts := []string{"— nowhere (a purchase, a favor) —"}
		opts = append(opts, groupedLabels(store.AssetAccounts())...)
		originSel = widget.NewSelect(opts, nil)
		guardGroupHeaders(originSel)
		originSel.SetSelected(opts[0])
		items = append(items, widget.NewFormItem("Deposited to", originSel))
	}
	items = append(items, widget.NewFormItem("Note", note))

	return &debtFields{
		items: items,
		commit: func() bool {
			amt := parseAmount(amount.Text)
			if name.Text == "" || acctMoney.Selected == "" || amt <= 0 {
				return false
			}
			d := Debt{
				Name: name.Text, Type: DebtInformal, Lender: lender.Text,
				AcctMoney: idOf(acctMoney.Selected),
				Principal: amt,
				DueDate:   parseDateSerial(due.Text),
				Note:      note.Text,
			}
			if existing != nil {
				d.AcctOrigin = existing.AcctOrigin
				store.UpdateDebt(existing.ID, d)
				return true
			}
			if originSel != nil && originSel.SelectedIndex() > 0 {
				d.AcctOrigin = idOf(originSel.Selected)
			}
			store.AddDebt(d, nil)
			return true
		},
	}
}

func orMonthly(freq string) string {
	if freq == "" {
		return "Monthly"
	}
	return freq
}
