package view

import (
	"testing"

	"github.com/raihanstark/financy/internal/core"
)

// addDemoDebt seeds one BNPL debt paid from checking and returns it with its
// first installment.
func addDemoDebt(s *core.Store) (core.Debt, core.Installment) {
	s.AddDebt(core.Debt{Name: "Phone", Type: core.DebtBNPL, Lender: "PayLater", AcctMoney: "checking"},
		core.GenerateInstallments(12000, 12, todaySerial, "Monthly"))
	d := s.Debts()[len(s.Debts())-1]
	return d, s.Installments(d.ID)[0]
}

// Paying an installment opens the match-review dialog (post new vs link) rather
// than posting immediately.
func TestPayInstallmentDialogOpens(t *testing.T) {
	s, w := newViewTest(t)
	d, in := addDemoDebt(s)
	before := len(s.Transactions())

	payInstallmentDialog(d, in)
	dlg := dialogContent(w)
	if dlg == nil {
		t.Fatal("pay dialog did not open")
	}
	assertText(t, dlg, "Pay installment", "Record this payment as:", postNewTodayOption)
	// Opening the dialog must not post anything on its own.
	if got := len(s.Transactions()); got != before {
		t.Fatalf("transactions changed before confirm: got %d, want %d", got, before)
	}
}

// Confirming the default (post-new) choice posts a fresh payment and marks the
// installment paid.
func TestPayInstallmentDialogPostsNew(t *testing.T) {
	s, w := newViewTest(t)
	d, in := addDemoDebt(s)
	before := len(s.Transactions())

	payInstallmentDialog(d, in)
	dlg := dialogContent(w)
	if dlg == nil {
		t.Fatal("pay dialog did not open")
	}
	if !tapButton(dlg, "Confirm") {
		t.Fatal("no Confirm button in pay dialog")
	}
	if got := len(s.Transactions()); got != before+1 {
		t.Fatalf("transactions after confirm = %d, want %d", got, before+1)
	}
	if !s.Installments(d.ID)[0].Paid {
		t.Error("installment not marked paid after confirming post-new")
	}
}
