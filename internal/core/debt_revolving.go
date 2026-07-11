package core

import (
	"sort"
	"time"
)

// Revolving debts (credit cards, lines of credit). There is no schedule and no
// origination: the liability account IS the debt. Card purchases are ordinary
// expense transactions against the account (each already hits a spending
// envelope), a payment is a budget-neutral Transfer into it, and interest is an
// Expense paid *from* the card. Interest is never auto-posted: once a month the
// statement-review prompt proposes an editable estimated charge — the real
// statement is ground truth, the user confirms or corrects it.

// addRevolvingDebt wires a revolving debt to its liability account: either the
// existing account the user already tracks the card with, or a fresh on-budget
// one (on-budget is what keeps card spending honest in the budget — see
// budgetFundsThrough). A starting balance is booked as an opening-balance
// event, exactly like creating an account by hand.
func (s *Store) addRevolvingDebt(d *Debt) {
	if d.AcctLiability == "" {
		liab := Account{ID: "debt_" + d.ID, Name: d.Name, Type: Liability, Institution: d.Lender, Notes: d.Type}
		s.AddAccount(liab)
		d.AcctLiability = liab.ID
		d.OwnsAccount = true
		if d.Principal > 0 {
			s.AddTransaction(Transaction{
				ID:    s.genID(),
				Date:  d.PurchaseDate,
				Payee: debtPayee(*d),
				Memo:  d.Name + " · opening balance",
				Posts: []Posting{P("opening", d.Principal), P(d.AcctLiability, -d.Principal)},
			})
		}
	}
	if d.NextStatement == 0 && d.StatementDay >= 1 {
		d.NextStatement = NextDayOfMonth(TodaySerial, d.StatementDay)
	}
}

// MinPaymentDue is a revolving debt's minimum payment on its current balance:
// the larger of the percentage rule and the floor, never more than the balance.
func (s *Store) MinPaymentDue(d Debt) int {
	bal := s.DebtBalance(d.ID)
	if bal <= 0 {
		return 0
	}
	min := int((int64(bal)*int64(d.MinPayBps) + 5000) / 10000)
	if min < d.MinPayFloor {
		min = d.MinPayFloor
	}
	if min > bal {
		min = bal
	}
	return min
}

// StatementDue is one revolving statement awaiting review: a proposed,
// editable interest charge for the user to confirm, correct, or skip.
type StatementDue struct {
	DebtID     string
	Name       string
	Date       int // statement close serial
	Balance    int // balance right now
	Interest   int // proposed estimate — the real statement is ground truth
	MinPayment int
}

// PendingStatements lists every revolving statement due on or before asOf,
// oldest first — one entry per missed month, like PendingRecurring, so months
// the app was closed still get reviewed.
func (s *Store) PendingStatements(asOf int) []StatementDue {
	var out []StatementDue
	for _, d := range s.debts {
		if !d.IsRevolving() || d.NextStatement == 0 {
			continue
		}
		bal := s.DebtBalance(d.ID)
		for due := d.NextStatement; due <= asOf; due = NextDayOfMonth(due, d.StatementDay) {
			out = append(out, StatementDue{
				DebtID:     d.ID,
				Name:       d.Name,
				Date:       due,
				Balance:    bal,
				Interest:   PeriodInterest(bal, d.APRBps, 12),
				MinPayment: s.MinPaymentDue(d),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

// PostStatement records a statement's interest charge — an Expense paid *from*
// the card, growing the balance — and advances the debt to the next statement.
// A zero amount just advances (nothing was charged this month). Returns whether
// the statement was processed.
func (s *Store) PostStatement(debtID string, amount, on int) bool {
	d := s.DebtByID(debtID)
	if d == nil || !d.IsRevolving() || d.NextStatement == 0 || amount < 0 {
		return false
	}
	if on == 0 {
		on = d.NextStatement
	}
	if amount > 0 {
		if !s.AddTransaction(Transaction{
			ID:    s.genID(),
			Date:  on,
			Payee: debtPayee(*d),
			Memo:  d.Name + " · interest",
			Posts: PostingsFor(KindExpense, d.AcctLiability, s.debtInterestAcct(d), amount),
		}) {
			return false
		}
	}
	s.advanceStatement(d)
	return true
}

// SkipStatement advances past the next statement without posting anything.
func (s *Store) SkipStatement(debtID string) bool {
	d := s.DebtByID(debtID)
	if d == nil || !d.IsRevolving() || d.NextStatement == 0 {
		return false
	}
	s.advanceStatement(d)
	return true
}

func (s *Store) advanceStatement(d *Debt) {
	d.NextStatement = NextDayOfMonth(d.NextStatement, d.StatementDay)
	s.reportError(s.dbUpsertDebt(*d))
	s.notify()
}

// NextDayOfMonth returns the serial of the first occurrence of a day-of-month
// strictly after `after`. Days are clamped to 1..28 so every month has one.
func NextDayOfMonth(after, day int) int {
	if day < 1 {
		day = 1
	}
	if day > 28 {
		day = 28
	}
	t := SerialToTime(after)
	cand := time.Date(t.Year(), t.Month(), day, 0, 0, 0, 0, time.UTC)
	if TimeToSerial(cand) <= after {
		cand = time.Date(t.Year(), t.Month()+1, day, 0, 0, 0, 0, time.UTC)
	}
	return TimeToSerial(cand)
}
