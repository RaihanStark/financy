package core

import (
	"testing"
	"time"
)

func debtStore() *Store {
	s := &Store{accounts: seedAccounts(), nextID: 1}
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	return s
}

func TestGenerateInstallmentsSplit(t *testing.T) {
	first := serialOf(2026, 1, 10)
	// 10000 over 3 months: remainder of 1 spread onto the earliest installments.
	insts := GenerateInstallments(10000, 3, first, "Monthly")
	if len(insts) != 3 {
		t.Fatalf("got %d installments, want 3", len(insts))
	}
	wantAmt := []int{3334, 3333, 3333}
	sum := 0
	for i, in := range insts {
		if in.Seq != i+1 {
			t.Errorf("installment %d seq = %d, want %d", i, in.Seq, i+1)
		}
		if in.Amount != wantAmt[i] {
			t.Errorf("installment %d amount = %d, want %d", i, in.Amount, wantAmt[i])
		}
		sum += in.Amount
	}
	if sum != 10000 {
		t.Errorf("installments sum = %d, want 10000", sum)
	}
	// Due dates advance monthly from the first.
	if insts[1].DueDate != serialOf(2026, 2, 10) || insts[2].DueDate != serialOf(2026, 3, 10) {
		t.Errorf("due dates didn't advance monthly: %s, %s",
			FmtSerialDate(insts[1].DueDate), FmtSerialDate(insts[2].DueDate))
	}
}

func TestPayInstallmentPostsTransaction(t *testing.T) {
	s := debtStore()
	first := serialOf(2026, 1, 1)
	d := Debt{Name: "Phone", Type: DebtBNPL, Lender: "PayLater", AcctMoney: "checking", AcctExpense: "shopping"}
	s.AddDebt(d, GenerateInstallments(12000, 12, first, "Monthly"))

	debts := s.Debts()
	if len(debts) != 1 {
		t.Fatalf("got %d debts, want 1", len(debts))
	}
	id := debts[0].ID
	liab := debts[0].AcctLiability
	insts := s.Installments(id)
	if len(insts) != 12 {
		t.Fatalf("got %d installments, want 12", len(insts))
	}

	// Creating the debt opens a Liability account and books the purchase: the
	// expense category is charged the full amount and the liability matches it.
	if a := s.AccountByID(liab); a == nil || a.Type != Liability {
		t.Fatalf("liability account not created: %v", a)
	}
	if got := len(s.Transactions()); got != 1 {
		t.Fatalf("got %d transactions after AddDebt, want 1 (the purchase)", got)
	}
	if bal := s.Balance("shopping"); bal != 12000 {
		t.Errorf("shopping balance after purchase = %d, want 12000", bal)
	}
	if out := s.DisplayBalance(*s.AccountByID(liab)); out != 12000 {
		t.Errorf("liability outstanding after purchase = %d, want 12000", out)
	}

	// Paying the first installment moves money into the liability (drawing it down).
	if !s.PayInstallment(insts[0].ID) {
		t.Fatal("PayInstallment returned false")
	}
	if got := len(s.Transactions()); got != 2 {
		t.Fatalf("got %d transactions after pay, want 2", got)
	}
	if bal := s.Balance("checking"); bal != -1000 {
		t.Errorf("checking balance = %d, want -1000", bal)
	}
	if out := s.DisplayBalance(*s.AccountByID(liab)); out != 11000 {
		t.Errorf("liability outstanding after one payment = %d, want 11000", out)
	}

	st := s.DebtStatus(id, first)
	if st.PaidCount != 1 || st.Paid != 1000 || st.Remaining != 11000 {
		t.Errorf("status after pay = %+v, want PaidCount=1 Paid=1000 Remaining=11000", st)
	}

	// Undo removes the payment (the purchase remains) and restores checking.
	if !s.UnpayInstallment(s.Installments(id)[0].ID) {
		t.Fatal("UnpayInstallment returned false")
	}
	if got := len(s.Transactions()); got != 1 {
		t.Fatalf("got %d transactions after undo, want 1 (the purchase)", got)
	}
	if bal := s.Balance("checking"); bal != 0 {
		t.Errorf("checking balance after undo = %d, want 0", bal)
	}
	if out := s.DisplayBalance(*s.AccountByID(liab)); out != 12000 {
		t.Errorf("liability outstanding after undo = %d, want 12000", out)
	}
}

func TestUpdateInstallment(t *testing.T) {
	s := debtStore()
	first := serialOf(2026, 1, 1)
	s.AddDebt(Debt{Name: "Sofa", Type: DebtBNPL, AcctMoney: "checking", AcctExpense: "shopping"},
		GenerateInstallments(6000, 6, first, "Monthly"))
	id := s.Debts()[0].ID
	liab := s.Debts()[0].AcctLiability
	insts := s.Installments(id)

	// Edit an unpaid installment's due date and amount (e.g. a bigger final month).
	// The liability opening re-syncs to the new schedule total (6000 → 6500).
	newDue := serialOf(2026, 2, 15)
	if !s.UpdateInstallment(insts[1].ID, newDue, 1500) {
		t.Fatal("UpdateInstallment returned false")
	}
	got := s.Installments(id)[1]
	if got.DueDate != newDue || got.Amount != 1500 {
		t.Errorf("edited installment = %+v, want due %s amount 1500", got, FmtSerialDate(newDue))
	}
	if out := s.DisplayBalance(*s.AccountByID(liab)); out != 6500 {
		t.Errorf("liability outstanding after editing schedule = %d, want 6500", out)
	}

	// Editing a PAID installment rewrites its payment so the books match: checking
	// reflects the new payment amount, and the liability draws down by it.
	if !s.PayInstallment(insts[0].ID) {
		t.Fatal("pay failed")
	}
	if !s.UpdateInstallment(insts[0].ID, first, 1200) {
		t.Fatal("update paid failed")
	}
	if bal := s.Balance("checking"); bal != -1200 {
		t.Errorf("checking balance after editing paid installment = %d, want -1200", bal)
	}
	// Schedule total is now 1200 + 1500 + 1000*4 = 6700; one payment of 1200 made.
	if out := s.DisplayBalance(*s.AccountByID(liab)); out != 5500 {
		t.Errorf("liability outstanding = %d, want 5500", out)
	}

	// Invalid edits are rejected.
	if s.UpdateInstallment(insts[2].ID, 0, 1000) || s.UpdateInstallment(insts[2].ID, first, 0) {
		t.Error("UpdateInstallment accepted an invalid value")
	}
}

func TestDebtStatusDueAndDelete(t *testing.T) {
	s := debtStore()
	first := serialOf(2026, 1, 1)
	s.AddDebt(Debt{Name: "Laptop", Type: DebtBNPL, AcctMoney: "checking", AcctExpense: "shopping"},
		GenerateInstallments(6000, 6, first, "Monthly"))
	id := s.Debts()[0].ID

	// As of the 3rd month, three installments (Jan/Feb/Mar) are due.
	asOf := serialOf(2026, 3, 15)
	if n := s.DueDebtCount(asOf); n != 3 {
		t.Errorf("DueDebtCount = %d, want 3", n)
	}
	if out := s.TotalDebtOutstanding(); out != 6000 {
		t.Errorf("TotalDebtOutstanding = %d, want 6000", out)
	}

	// Deleting the debt drops its installments, its Liability account, and the
	// purchase transaction booked against it.
	liab := s.Debts()[0].AcctLiability
	s.DeleteDebt(id)
	if len(s.Debts()) != 0 {
		t.Error("debt not deleted")
	}
	if len(s.Installments(id)) != 0 {
		t.Error("installments not deleted")
	}
	if s.AccountByID(liab) != nil {
		t.Error("liability account not deleted")
	}
	if len(s.Transactions()) != 0 {
		t.Errorf("got %d transactions after delete, want 0", len(s.Transactions()))
	}
}

func TestDebtPersistRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/debt.financy"
	s, err := NewDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	first := TimeToSerial(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	s.AddDebt(Debt{Name: "TV", Type: DebtBNPL, Lender: "Kredivo", AcctMoney: "checking", AcctExpense: "shopping"},
		GenerateInstallments(9000, 9, first, "Monthly"))
	id := s.Debts()[0].ID
	s.PayInstallment(s.Installments(id)[0].ID)
	s.Close()

	// Reopen and confirm the debt, its schedule, and the paid flag survived.
	s2, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if len(s2.Debts()) != 1 {
		t.Fatalf("got %d debts after reopen, want 1", len(s2.Debts()))
	}
	insts := s2.Installments(s2.Debts()[0].ID)
	if len(insts) != 9 {
		t.Fatalf("got %d installments after reopen, want 9", len(insts))
	}
	if !insts[0].Paid || insts[0].TxnID == "" {
		t.Errorf("first installment didn't persist as paid: %+v", insts[0])
	}
	if st := s2.DebtStatus(s2.Debts()[0].ID, first); st.PaidCount != 1 {
		t.Errorf("PaidCount after reopen = %d, want 1", st.PaidCount)
	}
	// The Liability account and purchase link (v4 columns) round-trip too.
	rd := s2.Debts()[0]
	if rd.AcctLiability == "" || rd.OriginTxnID == "" {
		t.Errorf("debt liability/origin links didn't persist: %+v", rd)
	}
	if a := s2.AccountByID(rd.AcctLiability); a == nil || a.Type != Liability {
		t.Errorf("liability account missing after reopen: %v", a)
	}
	// Outstanding = 9000 total − one paid installment of 1000.
	if out := s2.DisplayBalance(*s2.AccountByID(rd.AcctLiability)); out != 8000 {
		t.Errorf("liability outstanding after reopen = %d, want 8000", out)
	}
}
