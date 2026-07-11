package view

import (
	"testing"

	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// seedAllDebtTypes adds one debt of every kind to the store.
func seedAllDebtTypes(t *testing.T, s *core.Store) {
	t.Helper()
	s.AddDebt(core.Debt{Name: "Phone", Type: core.DebtBNPL, Lender: "PayLater", AcctMoney: "checking"},
		core.GenerateInstallments(12000, 12, todaySerial, "Monthly"))
	loan, ok := core.GenerateLoanSchedule(1_000_000, 1200, 88_849, todaySerial, "Monthly")
	if !ok {
		t.Fatal("loan schedule")
	}
	s.AddDebt(core.Debt{Name: "Car loan", Type: core.DebtLoan, Lender: "Bank", AcctMoney: "checking",
		APRBps: 1200, Freq: "Monthly", PaymentAmount: 88_849}, loan)
	s.AddDebt(core.Debt{Name: "Visa", Type: core.DebtRevolving, Lender: "BigBank", AcctMoney: "checking",
		APRBps: 1999, CreditLimit: 5_000_000, StatementDay: 5, PayDueDay: 25,
		MinPayBps: 500, MinPayFloor: 50_000, Principal: 800_000}, nil)
	s.AddDebt(core.Debt{Name: "IOU Andi", Type: core.DebtInformal, Lender: "Andi", AcctMoney: "checking",
		Principal: 250_000}, nil)
}

// The Debts screen renders the overview dashboard and a tab per debt of every
// kind.
func TestScreenDebtsRendersAllTypes(t *testing.T) {
	s, w := newViewTest(t)
	seedAllDebtTypes(t, s)

	scr := mount(w, ScreenDebts())
	assertText(t, scr,
		"Total owed", "Average APR", "Due now", "Debt-free by",
		"Phone", "Car loan", "Visa", "IOU Andi",
		"Strategies", "Add Debt",
	)
}

// The loan detail pane shows the balance, principal/interest columns and the
// extra-payment entry point.
func TestLoanDetailPane(t *testing.T) {
	s, w := newViewTest(t)
	seedAllDebtTypes(t, s)
	var loan core.Debt
	for _, d := range s.Debts() {
		if d.Type == core.DebtLoan {
			loan = d
		}
	}

	pane := mount(w, debtTabContent(loan))
	assertText(t, pane, "Balance", "Interest left", "Payoff", "Principal", "Interest", "Extra payment")
}

// The revolving detail pane shows utilization, the minimum payment and the
// record-payment action; informal shows the owed balance.
func TestRevolvingAndInformalPanes(t *testing.T) {
	s, w := newViewTest(t)
	seedAllDebtTypes(t, s)
	var card, iou core.Debt
	for _, d := range s.Debts() {
		switch d.Type {
		case core.DebtRevolving:
			card = d
		case core.DebtInformal:
			iou = d
		}
	}

	pane := mount(w, debtTabContent(card))
	assertText(t, pane, "Min payment", "Next statement", "Record payment", "of", "limit")

	pane = mount(w, debtTabContent(iou))
	assertText(t, pane, "Owed", "Borrowed", "Record payment")
}

// The add form is type-driven: each kind shows only its own fields — a bare
// minimum up front, everything optional folded into the Advanced accordion.
func TestDebtFormTypeSwitching(t *testing.T) {
	_, w := newViewTest(t)

	DebtForm(nil)
	dlg := dialogContent(w)
	if dlg == nil {
		t.Fatal("debt form did not open")
	}
	sels := findSelects(dlg)
	if len(sels) == 0 {
		t.Fatal("no type select in debt form")
	}
	kind := sels[0]

	assertText(t, dlg, "Total amount", "Installments", "Advanced setup") // BNPL default

	kind.SetSelected(core.DebtLoan)
	assertText(t, dlg, "Principal owed", "APR %", "Term")

	kind.SetSelected(core.DebtRevolving)
	assertText(t, dlg, "Current balance", "APR %")

	kind.SetSelected(core.DebtInformal)
	assertText(t, dlg, "Amount owed")

	// The simple form stays minimal; the detail lives behind Advanced setup,
	// pre-filled with working defaults.
	labels := func(items []*widget.FormItem) map[string]bool {
		m := map[string]bool{}
		for _, it := range items {
			m[it.Text] = true
		}
		return m
	}
	cases := []struct {
		kind            string
		basic, advanced []string
		maxBasic        int
	}{
		{core.DebtBNPL, []string{"Name", "Total amount", "Installments", "First due"},
			[]string{"Lender", "Pay from", "Purchase date", "Frequency", "Note"}, 5},
		{core.DebtLoan, []string{"Name", "Principal owed", "APR %", "Term", "First due"},
			[]string{"Lender", "Pay from", "Set up", "Payment", "As of", "Frequency", "Interest category", "Note"}, 6},
		{core.DebtRevolving, []string{"Name", "Current balance", "APR %"},
			[]string{"Issuer", "Pay from", "Card account", "Credit limit", "Statement day", "Min payment %", "Note"}, 4},
		{core.DebtInformal, []string{"Name", "Amount owed"},
			[]string{"Owed to", "Pay from", "Due date", "Deposited to", "Note"}, 2},
	}
	for _, c := range cases {
		f := newDebtFields(c.kind, nil)
		if len(f.items) > c.maxBasic {
			t.Errorf("%s: %d basic fields, want at most %d (keep the simple form simple)", c.kind, len(f.items), c.maxBasic)
		}
		basic, adv := labels(f.items), labels(f.advanced)
		for _, want := range c.basic {
			if !basic[want] {
				t.Errorf("%s: basic form missing %q", c.kind, want)
			}
		}
		for _, want := range c.advanced {
			if !adv[want] {
				t.Errorf("%s: advanced section missing %q", c.kind, want)
			}
			if basic[want] {
				t.Errorf("%s: %q should live in Advanced, not the simple form", c.kind, want)
			}
		}
	}
}

// The simple form saves without ever opening Advanced: defaults (pay-from
// account, dates, frequency) fill the gaps.
func TestDebtFormSimpleModeSaves(t *testing.T) {
	s, _ := newViewTest(t)

	f := newDebtFields(core.DebtBNPL, nil)
	byLabel := map[string]*widget.FormItem{}
	for _, it := range f.items {
		byLabel[it.Text] = it
	}
	byLabel["Name"].Widget.(*widget.Entry).SetText("Fridge")
	byLabel["Total amount"].Widget.(*widget.Entry).SetText(fmtMoneyInput(120_000))
	if !f.commit() {
		t.Fatal("simple BNPL form did not save with defaults")
	}
	var d *core.Debt
	for _, dd := range s.Debts() {
		if dd.Name == "Fridge" {
			d = &dd
			break
		}
	}
	if d == nil {
		t.Fatal("debt not created")
	}
	if d.AcctMoney == "" {
		t.Error("pay-from account did not default")
	}
	if got := len(s.Installments(d.ID)); got != 12 {
		t.Errorf("installments = %d, want the default 12", got)
	}
}

// Committing the loan form computes the payment from the term and creates the
// full amortization schedule.
func TestLoanFormComputesPayment(t *testing.T) {
	s, _ := newViewTest(t)

	f := newDebtFields(core.DebtLoan, nil)
	byLabel := map[string]*widget.FormItem{}
	for _, it := range append(append([]*widget.FormItem{}, f.items...), f.advanced...) {
		byLabel[it.Text] = it
	}
	byLabel["Name"].Widget.(*widget.Entry).SetText("Car")
	byLabel["Pay from"].Widget.(*widget.Select).SetSelected(acctLabel("Checking"))
	byLabel["Principal owed"].Widget.(*widget.Entry).SetText(fmtMoneyInput(1_000_000))
	byLabel["APR %"].Widget.(*widget.Entry).SetText("12")
	byLabel["Term"].Widget.(*widget.Entry).SetText("12")

	if !f.commit() {
		t.Fatal("loan form commit failed")
	}
	var loan *core.Debt
	for _, d := range s.Debts() {
		if d.Name == "Car" {
			loan = &d
			break
		}
	}
	if loan == nil {
		t.Fatal("loan not created")
	}
	if loan.PaymentAmount != 88_849 {
		t.Errorf("computed payment = %d, want 88849", loan.PaymentAmount)
	}
	insts := s.Installments(loan.ID)
	if len(insts) != 12 || insts[0].Interest != 10_000 {
		t.Errorf("schedule = %d rows, first interest %d; want 12 rows / 10000", len(insts), insts[0].Interest)
	}
	if s.DebtBalance(loan.ID) != 1_000_000 {
		t.Errorf("balance = %d, want 1000000", s.DebtBalance(loan.ID))
	}
}

// Recording a card payment posts a budget-neutral transfer and reduces the
// balance.
func TestRecordCardPaymentPostsTransfer(t *testing.T) {
	s, w := newViewTest(t)
	seedAllDebtTypes(t, s)
	var card core.Debt
	for _, d := range s.Debts() {
		if d.Type == core.DebtRevolving {
			card = d
		}
	}
	before := s.DebtBalance(card.ID)

	recordPaymentDialog(card)
	dlg := dialogContent(w)
	if dlg == nil {
		t.Fatal("record payment dialog did not open")
	}
	if !tapButton(dlg, "Save") {
		t.Fatal("no Save button")
	}
	// The default amount is the minimum due.
	want := before - 50_000
	if got := s.DebtBalance(card.ID); got != want {
		t.Errorf("card balance after payment = %d, want %d", got, want)
	}
	// The payment must not touch any expense category (budget-neutral).
	last := s.Transactions()[0]
	for _, p := range last.Posts {
		if a := s.AccountByID(p.AccountID); a != nil && a.Type == core.Expense {
			t.Errorf("card payment touched expense category %s", a.Name)
		}
	}
}

// The statement review posts the (editable) interest and advances the cycle.
func TestStatementReviewPostsInterest(t *testing.T) {
	s, w := newViewTest(t)
	seedAllDebtTypes(t, s)
	var card core.Debt
	for _, d := range s.Debts() {
		if d.Type == core.DebtRevolving {
			card = d
		}
	}
	stmt := s.DebtByID(card.ID).NextStatement
	pend := s.PendingStatements(stmt)
	if len(pend) == 0 {
		t.Fatal("no pending statement at its own date")
	}
	balBefore := s.DebtBalance(card.ID)

	statementReviewDialog(pend[0])
	dlg := dialogContent(w)
	if dlg == nil {
		t.Fatal("statement dialog did not open")
	}
	if !tapButton(dlg, "Post interest") {
		t.Fatal("no Post interest button")
	}
	if got := s.DebtBalance(card.ID); got != balBefore+pend[0].Interest {
		t.Errorf("balance after posting = %d, want %d", got, balBefore+pend[0].Interest)
	}
	if got := s.DebtByID(card.ID).NextStatement; got <= stmt {
		t.Errorf("NextStatement not advanced: %d", got)
	}
}

// The strategy dialog renders both plans with payoff order and totals.
func TestStrategyDialogRenders(t *testing.T) {
	s, w := newViewTest(t)
	seedAllDebtTypes(t, s)

	strategyDialog()
	dlg := dialogContent(w)
	if dlg == nil {
		t.Fatal("strategy dialog did not open")
	}
	assertText(t, dlg, "Snowball", "Avalanche", "Payoff order", "Total interest", "Debt-free")
}
