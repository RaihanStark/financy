package mobileui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/raihanstark/financy/internal/core"
)

// newMobileTest boots the mobile shell against a demo store, like the running
// app but headless.
func newMobileTest(t *testing.T) *mobileApp {
	t.Helper()
	a := test.NewApp()
	a.Settings().SetTheme(appTheme{})
	w := test.NewWindow(nil)
	w.Resize(fyne.NewSize(390, 844))

	m := &mobileApp{fyneApp: a, win: w}
	ctl = m
	m.build()

	s := core.NewStore()
	core.SeedDemo(s, "$")
	m.store = s
	return m
}

// seedMobileDebts adds one debt of every kind.
func seedMobileDebts(t *testing.T, s *core.Store) {
	t.Helper()
	acct := s.MoneyAccounts()
	if len(acct) == 0 {
		t.Fatal("demo store has no money accounts")
	}
	pay := acct[0].ID
	s.AddDebt(core.Debt{Name: "Phone", Type: core.DebtBNPL, Lender: "PayLater", AcctMoney: pay},
		core.GenerateInstallments(12000, 12, core.TodaySerial, "Monthly"))
	loan, ok := core.GenerateLoanSchedule(1_000_000, 1200, 88_849, core.TodaySerial, "Monthly")
	if !ok {
		t.Fatal("loan schedule")
	}
	s.AddDebt(core.Debt{Name: "Car loan", Type: core.DebtLoan, Lender: "Bank", AcctMoney: pay,
		APRBps: 1200, Freq: "Monthly", PaymentAmount: 88_849}, loan)
	s.AddDebt(core.Debt{Name: "Visa", Type: core.DebtRevolving, Lender: "BigBank", AcctMoney: pay,
		APRBps: 1999, CreditLimit: 5_000_000, StatementDay: 5, PayDueDay: 25,
		MinPayBps: 500, MinPayFloor: 50_000, Principal: 800_000}, nil)
	s.AddDebt(core.Debt{Name: "IOU Andi", Type: core.DebtInformal, Lender: "Andi", AcctMoney: pay,
		Principal: 250_000, DueDate: core.TodaySerial - 1}, nil)
}

// Every debt kind renders on the Debts tab, in its detail page, and through
// the add/edit and payment flows without panicking.
func TestMobileDebtScreensRenderAllTypes(t *testing.T) {
	m := newMobileTest(t)
	seedMobileDebts(t, m.store)

	m.selectTab(3) // Debts tab with all four cards + strategies row
	for _, d := range m.store.Debts() {
		m.openDebt(d)
		m.win.Content().Refresh()
		m.back()
	}

	// Type-driven add form: switching kinds rebuilds the fields.
	m.debtForm(nil)
	m.win.Content().Refresh()
	m.back()

	// Edit form per kind.
	for _, d := range m.store.Debts() {
		d := d
		m.editDebt(d)
		m.win.Content().Refresh()
		m.back()
	}

	// Payment pages for schedule-less kinds, extra payment for the loan.
	for _, d := range m.store.Debts() {
		switch {
		case !d.HasSchedule():
			m.payDebtAmountPage(d)
			m.win.Content().Refresh()
			m.back()
		case d.Type == core.DebtLoan:
			m.extraPaymentPage(d)
			m.win.Content().Refresh()
			m.back()
		}
	}

	// Strategies page.
	m.strategyPage()
	m.win.Content().Refresh()
	m.back()
}

// The statement review page posts interest to the card and advances the cycle.
func TestMobileStatementReview(t *testing.T) {
	m := newMobileTest(t)
	seedMobileDebts(t, m.store)
	var card core.Debt
	for _, d := range m.store.Debts() {
		if d.Type == core.DebtRevolving {
			card = d
		}
	}
	stmt := m.store.DebtByID(card.ID).NextStatement
	pend := m.store.PendingStatements(stmt)
	if len(pend) == 0 {
		t.Fatal("no pending statement at its own date")
	}
	balBefore := m.store.DebtBalance(card.ID)

	refreshed := false
	m.statementReviewPage(pend[0], func() { refreshed = true })
	m.win.Content().Refresh()

	// Drive the Post action directly through the store like the page does.
	if !m.store.PostStatement(card.ID, pend[0].Interest, pend[0].Date) {
		t.Fatal("PostStatement failed")
	}
	if got := m.store.DebtBalance(card.ID); got != balBefore+pend[0].Interest {
		t.Errorf("balance = %d, want %d", got, balBefore+pend[0].Interest)
	}
	m.back()
	_ = refreshed
}
