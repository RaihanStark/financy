package core

import "sort"

// Debts: tracked liabilities. Four kinds share one model, differing mainly in
// whether they carry a payment schedule and how interest works:
//
//   - BNPL: a purchase split into a fixed number of equal, interest-free
//     installments. Paying one posts a real transaction (money → liability).
//   - Loan: an amortizing loan (mortgage, auto, student, personal). The schedule
//     is generated from principal + APR; each payment splits into principal
//     (drawing the liability down) and interest (a real expense, recognised as
//     paid — never pre-booked).
//   - Revolving: a credit card or line of credit. No schedule — the liability
//     account IS the debt; purchases and payments are ordinary transactions,
//     and interest lands via a monthly statement-review prompt.
//   - Informal: money owed to a person (an IOU). Just a balance with an
//     optional due date; pay arbitrary amounts anytime.

// Debt kinds.
const (
	DebtBNPL      = "BNPL"
	DebtLoan      = "Loan"
	DebtRevolving = "Revolving"
	DebtInformal  = "Informal"
)

// DebtTypes returns the selectable debt kinds, in order.
func DebtTypes() []string {
	return []string{DebtBNPL, DebtLoan, DebtRevolving, DebtInformal}
}

// HasSchedule reports whether this debt kind carries a fixed installment
// schedule. Unknown/legacy values fall back to BNPL behavior.
func (d Debt) HasSchedule() bool { return !d.IsRevolving() && d.Type != DebtInformal }

// HasEnvelope reports whether the debt gets its own budget envelope (backed by
// its off-budget liability account). Revolving debts don't: a card purchase
// already hits a spending envelope, so a payment envelope would double-count.
func (d Debt) HasEnvelope() bool { return !d.IsRevolving() }

// IsRevolving reports whether the debt is a revolving credit line.
func (d Debt) IsRevolving() bool { return d.Type == DebtRevolving }

// Debt is one tracked liability with an installment schedule. Each debt owns an
// off-budget Liability account (AcctLiability) that mirrors its outstanding
// balance, so it shows up in Accounts and Net Worth. When the debt is created we
// book the purchase as a balance-sheet-only financing event (OriginTxnID): the
// liability rises by the schedule total against an Equity contra — Net Worth
// drops by what you owe, but no expense category is charged. The debt is its own
// budget envelope; paying an installment moves money from AcctMoney into the
// liability (drawing it down) and counts as that envelope's spending.
type Debt struct {
	ID           string
	Name         string // what it's for, e.g. "iPhone 15"
	Type         string // one of DebtTypes()
	Lender       string // e.g. "Shopee PayLater"
	AcctMoney    string // money account installments are paid from
	PurchaseDate int    // when the debt was incurred — the origination date. The
	// whole point of "buy now, pay later" is this differs from the first payment's
	// due date. 0 means "today" at creation time.
	Note string

	AcctLiability string // Liability account mirroring the outstanding balance
	OriginTxnID   string // the purchase transaction that opened the liability

	// Interest-bearing kinds. APRBps is the annual rate in basis points
	// (1999 = 19.99%); 0 means interest-free.
	APRBps       int
	AcctInterest string // Expense category interest is charged to ("" → default)

	// Loan: the regular payment and its cadence; Principal is the remaining
	// principal at setup (as of PurchaseDate), which the schedule amortizes.
	Freq          string // one of Frequencies()
	PaymentAmount int    // minor units
	Principal     int    // minor units; Informal reuses it as the amount owed

	// Revolving: the card / credit line profile. The liability account balance
	// is the debt; NextStatement is the serial of the next statement to review.
	CreditLimit   int
	StatementDay  int // day of month the statement closes, 1..28
	PayDueDay     int // day of month payment is due, 1..28
	MinPayBps     int // minimum payment as bps of the balance
	MinPayFloor   int // minimum payment floor, minor units
	NextStatement int
	OwnsAccount   bool // AddDebt created AcctLiability (delete may remove it)

	// Informal: an optional single due date (0 = none). AcctOrigin optionally
	// names the asset account the borrowed money landed in ("" → equity contra).
	DueDate    int
	AcctOrigin string
}

// Installment is one scheduled payment of a Debt.
type Installment struct {
	ID      string
	DebtID  string
	Seq     int // 1-based position in the schedule
	DueDate int // Excel serial
	Amount  int // minor units, positive; always Principal + Interest
	// The split: Principal draws the liability down, Interest is expensed when
	// paid. BNPL installments are all principal (Interest 0).
	Principal int
	Interest  int
	Paid      bool
	TxnID     string // the posted (or linked) transaction, set when paid
	// Linked is true when the installment was satisfied by linking an existing
	// transaction the user already recorded — TxnID points at a transaction we did
	// NOT create. We re-point that transaction onto the liability so the books
	// reconcile; OrigPosts snapshots its prior postings so undo can restore them
	// instead of deleting a row that isn't ours.
	Linked    bool
	OrigPosts []Posting
}

// GenerateInstallments splits total into n near-equal installments starting at
// firstDue and advancing by freq. The remainder (when total isn't divisible) is
// spread one minor unit at a time over the earliest installments, so the amounts
// stay within one unit of each other and sum back to exactly total.
func GenerateInstallments(total, n, firstDue int, freq string) []Installment {
	if n <= 0 {
		return nil
	}
	base := total / n
	rem := total - base*n
	out := make([]Installment, 0, n)
	due := firstDue
	for i := 0; i < n; i++ {
		amt := base
		if i < rem {
			amt++
		}
		out = append(out, Installment{Seq: i + 1, DueDate: due, Amount: amt, Principal: amt})
		due = nextDate(due, freq)
	}
	return out
}

// ---- accessors / CRUD ----

func (s *Store) Debts() []Debt { return append([]Debt(nil), s.debts...) }

func (s *Store) DebtByID(id string) *Debt {
	for i := range s.debts {
		if s.debts[i].ID == id {
			return &s.debts[i]
		}
	}
	return nil
}

// Installments returns a debt's schedule, ordered by sequence.
func (s *Store) Installments(debtID string) []Installment {
	var out []Installment
	for _, in := range s.installments {
		if in.DebtID == debtID {
			out = append(out, in)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}

func (s *Store) installmentByID(id string) *Installment {
	for i := range s.installments {
		if s.installments[i].ID == id {
			return &s.installments[i]
		}
	}
	return nil
}

// AddDebt persists a debt together with its schedule (schedule kinds only).
//
// BNPL / Loan / Informal each get their own off-budget Liability account, and
// the debt is opened by an origination transaction dated the purchase date
// (defaulting to today — NOT the first payment's due date), so it appears in
// Accounts and Net Worth from the moment you incur it. The origination books
// PRINCIPAL only: a loan's interest isn't owed until it accrues, so it is
// expensed per payment, never pre-booked.
//
// Revolving debts are different: the liability account IS the debt (attached
// or created on-budget), there is no origination and no schedule — purchases
// and payments are ordinary transactions against the account.
func (s *Store) AddDebt(d Debt, insts []Installment) {
	if d.ID == "" {
		d.ID = s.genDebtID()
	}
	if d.PurchaseDate == 0 {
		d.PurchaseDate = TodaySerial
	}

	if d.IsRevolving() {
		s.addRevolvingDebt(&d)
		insts = nil
	} else {
		// The Liability account that mirrors the outstanding balance. It's a tracking
		// (off-budget) account: the debt shows in Accounts and Net Worth, but the money
		// you still owe is NOT counted as assignable budget funds. Instead each debt is
		// its own budget envelope (see budget.go) that you fund as you pay it down.
		liab := Account{ID: "debt_" + d.ID, Name: d.Name, Type: Liability, Institution: d.Lender, Notes: d.Type, OffBudget: true}
		s.AddAccount(liab)
		d.AcctLiability = liab.ID
		d.OwnsAccount = true

		total := 0
		if d.HasSchedule() {
			for _, in := range insts {
				total += in.Principal
			}
			if d.Type == DebtLoan {
				d.Principal = total
				if d.AcctInterest == "" {
					d.AcctInterest = s.ensureLoanInterestCategory()
				}
			}
		} else {
			// Informal: no schedule — you owe the stated amount outright.
			total = d.Principal
			insts = nil
		}

		// Book the origination as a balance-sheet-only financing event: the liability
		// rises by the principal against an Equity contra. Net Worth drops by what
		// you now owe, but Ready to Assign is untouched — the cost is recognised in
		// the budget as you assign to the debt's envelope and pay it down. An informal
		// debt may instead name the asset account the borrowed money landed in.
		contra := s.ensureFinancedEquity()
		if d.Type == DebtInformal && d.AcctOrigin != "" {
			contra = d.AcctOrigin
		}
		d.OriginTxnID = s.genID()
		s.AddTransaction(Transaction{
			ID:    d.OriginTxnID,
			Date:  d.PurchaseDate,
			Payee: debtPayee(d),
			Memo:  d.Name + debtOriginMemo(d),
			Posts: PostingsFor(KindExpense, d.AcctLiability, contra, total),
		})
	}

	if err := s.dbUpsertDebt(d); err != nil {
		s.reportError(err)
		return
	}
	next := s.maxInstallmentNum() + 1
	for i := range insts {
		insts[i].DebtID = d.ID
		insts[i].ID = "i" + Itoa(next)
		next++
		if err := s.dbUpsertInstallment(insts[i]); err != nil {
			s.reportError(err)
			return
		}
	}
	s.debts = append(s.debts, d)
	s.installments = append(s.installments, insts...)
	s.notify()
}

// UpdateDebt edits a debt's metadata and terms. The Liability account's label
// follows the name / lender, and the origination booking is re-synced. On a
// loan, changing the APR, payment, or frequency regenerates the unpaid part of
// the schedule from the current balance (paid history is never rewritten). On
// a revolving debt, changing the statement day re-anchors the next statement.
func (s *Store) UpdateDebt(id string, d Debt) {
	existing := s.DebtByID(id)
	if existing == nil {
		return
	}
	d.ID = id
	d.Type = existing.Type // the kind is fixed at creation
	d.AcctLiability = existing.AcctLiability
	d.OriginTxnID = existing.OriginTxnID
	d.OwnsAccount = existing.OwnsAccount
	reamortize := false
	switch {
	case d.IsRevolving():
		d.NextStatement = existing.NextStatement
		if d.StatementDay != existing.StatementDay && d.StatementDay >= 1 {
			d.NextStatement = NextDayOfMonth(TodaySerial, d.StatementDay)
		}
	case d.Type == DebtLoan:
		d.Principal = existing.Principal // what was borrowed doesn't change
		reamortize = d.APRBps != existing.APRBps ||
			d.PaymentAmount != existing.PaymentAmount || d.Freq != existing.Freq
	}

	if a := s.AccountByID(d.AcctLiability); a != nil {
		upd := *a
		upd.Name, upd.Institution = d.Name, d.Lender
		s.UpdateAccount(a.ID, upd)
	}
	if err := s.dbUpsertDebt(d); err != nil {
		s.reportError(err)
		return
	}
	for i := range s.debts {
		if s.debts[i].ID == id {
			s.debts[i] = d
		}
	}
	s.syncDebtOrigination(&d)
	if reamortize {
		s.reamortizeUnpaid(s.DebtByID(id), true)
	}
	s.notify()
}

// PayDebtAmount records an arbitrary-amount payment against a debt without a
// schedule. On a revolving debt it's a budget-neutral transfer into the card
// (the spending already hit envelopes at purchase time); on an informal debt
// it's the envelope's payment, clamped to what's still owed. Returns whether
// it posted.
func (s *Store) PayDebtAmount(debtID string, amount, on int) bool {
	d := s.DebtByID(debtID)
	if d == nil || d.HasSchedule() || amount <= 0 || d.AcctMoney == "" {
		return false
	}
	kind := KindTransfer
	if !d.IsRevolving() {
		kind = KindExpense
		bal := s.DebtBalance(debtID)
		if bal <= 0 {
			return false
		}
		if amount > bal {
			amount = bal
		}
	}
	if on == 0 {
		on = TodaySerial
	}
	if !s.AddTransaction(Transaction{
		ID:    s.genID(),
		Date:  on,
		Payee: debtPayee(*d),
		Memo:  d.Name + " · payment",
		Posts: PostingsFor(kind, d.AcctMoney, d.AcctLiability, amount),
	}) {
		return false
	}
	s.notify()
	return true
}

// DeleteDebt removes a debt. For BNPL / Loan / Informal that's a full undo:
// linked installments first get their original postings restored (those
// transactions are the user's own — un-point them, don't destroy them), then
// the origination and every payment we posted are deleted along with the
// Liability account. A revolving debt instead DETACHES: card purchases and
// payments are the user's real history and are never touched; the account is
// removed only when we created it and nothing was ever posted to it.
func (s *Store) DeleteDebt(id string) {
	d := s.DebtByID(id)
	if d == nil {
		return
	}
	if d.IsRevolving() {
		if d.OwnsAccount && d.AcctLiability != "" {
			s.DeleteAccount(d.AcctLiability) // refuses if the account has postings
		}
	} else if liab := d.AcctLiability; liab != "" {
		for _, in := range s.Installments(id) {
			if in.Paid && in.Linked {
				s.UnpayInstallment(in.ID)
			}
		}
		var txnIDs []string
		for _, t := range s.txns {
			if _, touches := acctSum(t, liab); touches {
				txnIDs = append(txnIDs, t.ID)
			}
		}
		for _, tid := range txnIDs {
			s.DeleteTransaction(tid)
		}
		s.DeleteAccount(liab)
	}
	if err := s.dbDeleteDebt(id); err != nil { // cascades to installments on disk
		s.reportError(err)
		return
	}
	for i := range s.debts {
		if s.debts[i].ID == id {
			s.debts = append(s.debts[:i], s.debts[i+1:]...)
			break
		}
	}
	kept := s.installments[:0]
	for _, in := range s.installments {
		if in.DebtID != id {
			kept = append(kept, in)
		}
	}
	s.installments = kept
	s.notify()
}

// syncDebtOrigination rewrites the origination transaction so the liability
// always opens at what was actually borrowed, preserving its original date.
// For BNPL that's the schedule's (editable) total; for Loan and Informal it's
// the fixed principal — a loan's schedule can shrink below it after extra
// payments, but the amount originally borrowed doesn't change.
func (s *Store) syncDebtOrigination(d *Debt) {
	if d.OriginTxnID == "" {
		return
	}
	date := firstDueOf(s.Installments(d.ID))
	contra := s.ensureFinancedEquity()
	if d.Type == DebtInformal && d.AcctOrigin != "" {
		contra = d.AcctOrigin
	}
	if orig := s.TxnByID(d.OriginTxnID); orig != nil {
		date = orig.Date
	}
	total := d.Principal
	if d.Type == DebtBNPL {
		total = 0
		for _, in := range s.Installments(d.ID) {
			total += in.Principal
		}
	}
	s.UpdateTransaction(d.OriginTxnID, Transaction{
		ID:    d.OriginTxnID,
		Date:  date,
		Payee: debtPayee(*d),
		Memo:  d.Name + debtOriginMemo(*d),
		Posts: PostingsFor(KindExpense, d.AcctLiability, contra, total),
	})
}

func firstDueOf(insts []Installment) int {
	if len(insts) > 0 {
		return insts[0].DueDate
	}
	return TodaySerial
}

// UpdateInstallment changes one installment's due date and amount — schedules
// aren't always equal (a final balloon payment, a skipped month, a fee). The
// scheduled interest stays fixed; an amount edit moves the principal portion,
// so on a loan the new amount must still cover the interest. If the installment
// is already paid, its linked transaction is rewritten to match so the books
// stay in sync. Returns whether it updated.
func (s *Store) UpdateInstallment(instID string, dueDate, amount int) bool {
	in := s.installmentByID(instID)
	if in == nil || dueDate == 0 || amount <= in.Interest || amount <= 0 {
		return false
	}
	in.DueDate = dueDate
	in.Amount = amount
	in.Principal = amount - in.Interest
	d := s.DebtByID(in.DebtID)
	if in.Paid && in.TxnID != "" && d != nil {
		// Keep the actual payment date — editing the schedule shouldn't move the
		// date the money already left your account.
		paidOn := dueDate
		if old := s.TxnByID(in.TxnID); old != nil {
			paidOn = old.Date
		}
		s.UpdateTransaction(in.TxnID, Transaction{
			ID:    in.TxnID,
			Date:  paidOn,
			Payee: debtPayee(*d),
			Memo:  d.Name + " · installment " + Itoa(in.Seq),
			Posts: s.installmentPosts(d, in.Principal, in.Interest, amount),
		})
	}
	s.reportError(s.dbUpsertInstallment(*in))
	if d != nil {
		s.syncDebtOrigination(d) // total may have changed → keep the liability opening in step
	}
	s.notify()
	return true
}

// PayInstallment posts an installment as an Expense transaction dated `on` — the
// actual payment date, which is when the cash really leaves your account. That's
// "today" when you pay from the UI (paying early, or clearing an overdue one, both
// happen now); the due date stays the schedule, used only for "overdue" detection.
// Pass 0 to default to today. Marks the installment paid and links it to the
// transaction. Returns whether it posted.
func (s *Store) PayInstallment(instID string, on int) bool {
	in := s.installmentByID(instID)
	if in == nil || in.Paid {
		return false
	}
	d := s.DebtByID(in.DebtID)
	if d == nil {
		return false
	}
	if on == 0 {
		on = TodaySerial
	}
	txnID := s.genID()
	txn := Transaction{
		ID:    txnID,
		Date:  on,
		Payee: debtPayee(*d),
		Memo:  d.Name + " · installment " + Itoa(in.Seq),
		Posts: s.installmentPosts(d, in.Principal, in.Interest, in.Amount),
	}
	if !s.AddTransaction(txn) {
		return false
	}
	in.Paid = true
	in.TxnID = txnID
	s.reportError(s.dbUpsertInstallment(*in))
	s.notify()
	return true
}

// LinkInstallment marks an installment paid by linking it to a transaction the
// user already recorded for this payment (e.g. entered manually or imported),
// instead of posting a fresh one. A normal payment draws the liability down by
// moving money into it; an existing transaction was almost always booked against
// a spending category instead, so it neither draws the liability down nor avoids
// double-counting the cost (already recognised at origination). We therefore
// re-point the chosen transaction onto the liability — preserving the amount that
// actually left the money account — and snapshot its prior postings so undo can
// restore them. Returns whether it linked.
func (s *Store) LinkInstallment(instID, txnID string) bool {
	in := s.installmentByID(instID)
	if in == nil || in.Paid {
		return false
	}
	d := s.DebtByID(in.DebtID)
	if d == nil {
		return false
	}
	t := s.TxnByID(txnID)
	if t == nil {
		return false
	}
	// The real amount paid is whatever left the money account on this transaction;
	// fall back to the scheduled amount if it doesn't touch it.
	paid := absInt(func() int { sum, _ := acctSum(*t, d.AcctMoney); return sum }())
	if paid == 0 {
		paid = in.Amount
	}
	// On a loan the payment covers the scheduled interest first; whatever is left
	// draws the principal down. A short payment is all interest.
	interest := in.Interest
	if interest > paid {
		interest = paid
	}
	orig := append([]Posting(nil), t.Posts...)
	upd := *t
	upd.Posts = s.installmentPosts(d, paid-interest, interest, paid)
	if !s.UpdateTransaction(txnID, upd) {
		return false
	}
	in.Paid = true
	in.TxnID = txnID
	in.Linked = true
	in.OrigPosts = orig
	s.reportError(s.dbUpsertInstallment(*in))
	s.notify()
	return true
}

// UnpayInstallment reverses a payment and marks the installment unpaid again. A
// payment we posted is deleted outright; a linked transaction (one the user
// already had) is left in place with its original postings restored, so undo
// never destroys a row we didn't create. Returns whether it reversed.
func (s *Store) UnpayInstallment(instID string) bool {
	in := s.installmentByID(instID)
	if in == nil || !in.Paid {
		return false
	}
	switch {
	case in.Linked:
		if in.TxnID != "" && len(in.OrigPosts) > 0 {
			if t := s.TxnByID(in.TxnID); t != nil {
				upd := *t
				upd.Posts = append([]Posting(nil), in.OrigPosts...)
				s.UpdateTransaction(in.TxnID, upd)
			}
		}
	case in.TxnID != "":
		s.DeleteTransaction(in.TxnID)
	}
	in.Paid = false
	in.TxnID = ""
	in.Linked = false
	in.OrigPosts = nil
	s.reportError(s.dbUpsertInstallment(*in))
	s.notify()
	return true
}

// clearInstallmentForTxn resets any installment satisfied by txnID back to unpaid.
// Called from DeleteTransaction so deleting a debt payment from the Transactions
// page also rolls back the installment on the Debt page. It only mutates the
// installment (never deletes the txn), so it's safe to call mid-delete without
// recursing back into DeleteTransaction. For a linked installment the transaction
// was the user's own; it's being deleted, so we can't restore OrigPosts — reverting
// to unpaid is the right outcome (the user can re-link later).
func (s *Store) clearInstallmentForTxn(txnID string) {
	if txnID == "" {
		return
	}
	for i := range s.installments {
		in := &s.installments[i]
		if in.TxnID != txnID {
			continue
		}
		in.Paid = false
		in.TxnID = ""
		in.Linked = false
		in.OrigPosts = nil
		s.reportError(s.dbUpsertInstallment(*in))
	}
}

// ---- progress ----

// DebtStatus is a debt's rolled-up progress as of a date.
type DebtStatus struct {
	Total         int // sum of all installments
	Paid          int // amount paid so far
	Remaining     int // amount still owed
	Count         int // number of installments
	PaidCount     int // installments paid
	NextDue       int // serial of the next unpaid installment, 0 if all paid
	OverdueCount  int // unpaid installments due on/before asOf
	OverdueAmount int
}

// DebtStatus rolls up a debt's installments into progress figures.
func (s *Store) DebtStatus(debtID string, asOf int) DebtStatus {
	var st DebtStatus
	for _, in := range s.Installments(debtID) {
		st.Count++
		st.Total += in.Amount
		if in.Paid {
			st.PaidCount++
			st.Paid += in.Amount
			continue
		}
		st.Remaining += in.Amount
		if st.NextDue == 0 || in.DueDate < st.NextDue {
			st.NextDue = in.DueDate
		}
		if in.DueDate <= asOf {
			st.OverdueCount++
			st.OverdueAmount += in.Amount
		}
	}
	return st
}

// DebtBuckets summarizes unpaid installments by timing relative to asOf: what's
// already due, what's still coming later this month, and what falls in the next
// calendar month. The buckets are disjoint; Outstanding is the total of all unpaid.
type DebtBuckets struct {
	Outstanding int // every unpaid installment
	Due         int // unpaid and due on or before asOf — already owed
	ThisMonth   int // unpaid, due later this month (after asOf)
	NextMonth   int // unpaid, due in the next calendar month
}

// DebtBuckets rolls what's owed across all debts into timing buckets: unpaid
// installments by due date, an informal debt's balance by its (optional) due
// date, and a revolving debt's minimum payment by its pending statement date.
func (s *Store) DebtBuckets(asOf int) DebtBuckets {
	var b DebtBuckets
	thisM := FmtSerialMonth(asOf)
	nextM := ShiftMonth(thisM, 1)
	bucket := func(amount, due int) {
		switch {
		case due <= asOf:
			b.Due += amount
		case FmtSerialMonth(due) == thisM:
			b.ThisMonth += amount
		case FmtSerialMonth(due) == nextM:
			b.NextMonth += amount
		}
	}
	for _, in := range s.installments {
		if in.Paid {
			continue
		}
		b.Outstanding += in.Amount
		bucket(in.Amount, in.DueDate)
	}
	for _, d := range s.debts {
		if d.HasSchedule() {
			continue
		}
		bal := s.DebtBalance(d.ID)
		if bal <= 0 {
			continue
		}
		b.Outstanding += bal
		switch {
		case d.IsRevolving():
			if d.NextStatement > 0 {
				bucket(s.MinPaymentDue(d), d.NextStatement)
			}
		case d.DueDate > 0:
			bucket(bal, d.DueDate)
		}
	}
	return b
}

// DueDebtCount counts what needs attention as of a date: unpaid installments
// past due, informal debts past their due date, and revolving statements
// awaiting review.
func (s *Store) DueDebtCount(asOf int) int {
	n := 0
	for _, in := range s.installments {
		if !in.Paid && in.DueDate <= asOf {
			n++
		}
	}
	for _, d := range s.debts {
		if d.HasSchedule() {
			continue
		}
		switch {
		case d.IsRevolving():
			if d.NextStatement > 0 && d.NextStatement <= asOf {
				n++
			}
		case d.DueDate > 0 && d.DueDate <= asOf && s.DebtBalance(d.ID) > 0:
			n++
		}
	}
	return n
}

// TotalDebtOutstanding is the money still owed across all tracked debts — the
// sum of every debt's liability balance.
func (s *Store) TotalDebtOutstanding() int {
	sum := 0
	for _, d := range s.debts {
		if bal := s.DebtBalance(d.ID); bal > 0 {
			sum += bal
		}
	}
	return sum
}

// installmentPosts builds the balanced postings for an installment payment of
// `paid` (= principal + interest): principal draws the liability down, and any
// interest is charged to the debt's interest category as a real expense — a
// 3-leg split on loans, the classic 2-leg posting when there's no interest.
func (s *Store) installmentPosts(d *Debt, principal, interest, paid int) []Posting {
	if interest <= 0 {
		return PostingsFor(KindExpense, d.AcctMoney, d.AcctLiability, paid)
	}
	posts := make([]Posting, 0, 3)
	if principal > 0 {
		posts = append(posts, P(d.AcctLiability, principal))
	}
	posts = append(posts, P(s.debtInterestAcct(d), interest), P(d.AcctMoney, -paid))
	return posts
}

// debtInterestAcct is the Expense category a debt's interest is charged to,
// falling back to the shared Loan Fees & Interest category.
func (s *Store) debtInterestAcct(d *Debt) string {
	if d.AcctInterest != "" && s.AccountByID(d.AcctInterest) != nil {
		return d.AcctInterest
	}
	return s.ensureLoanInterestCategory()
}

// DebtBalance is what's still owed on a debt right now — the (displayed)
// balance of its liability account. This is the single source of truth for all
// kinds: remaining principal on BNPL/Loan, the running card balance on
// Revolving, the amount still owed on Informal.
func (s *Store) DebtBalance(debtID string) int {
	d := s.DebtByID(debtID)
	if d == nil || d.AcctLiability == "" {
		return 0
	}
	return -s.Balance(d.AcctLiability)
}

// ApplyExtraPayment posts an extra, principal-only payment against a loan and
// regenerates the unpaid schedule from the new balance. keepPayment=true keeps
// the regular payment and shortens the term (the lender default, and what saves
// the most interest); false keeps the remaining number of payments and lowers
// the payment (re-amortize). Returns whether it posted.
func (s *Store) ApplyExtraPayment(debtID string, amount, on int, keepPayment bool) bool {
	d := s.DebtByID(debtID)
	if d == nil || d.Type != DebtLoan || amount <= 0 {
		return false
	}
	bal := s.DebtBalance(debtID)
	if bal <= 0 {
		return false
	}
	if amount > bal {
		amount = bal
	}
	if on == 0 {
		on = TodaySerial
	}
	if !s.AddTransaction(Transaction{
		ID:    s.genID(),
		Date:  on,
		Payee: debtPayee(*d),
		Memo:  d.Name + " · extra payment",
		Posts: PostingsFor(KindExpense, d.AcctMoney, d.AcctLiability, amount),
	}) {
		return false
	}
	s.reamortizeUnpaid(d, keepPayment)
	s.notify()
	return true
}

// reamortizeUnpaid replaces a loan's unpaid installments with a fresh schedule
// walked from the current balance — after an extra payment, or after the APR /
// payment / frequency changed. Paid rows are untouched; new rows are numbered
// after them. keepPayment keeps d.PaymentAmount (term shortens); otherwise the
// payment is re-solved over the remaining count of installments (term keeps,
// payment drops) and stored back on the debt.
func (s *Store) reamortizeUnpaid(d *Debt, keepPayment bool) {
	insts := s.Installments(d.ID)
	maxPaidSeq, unpaidCount := 0, 0
	firstDue, lastPaidDue := 0, 0
	for _, in := range insts {
		if in.Paid {
			if in.Seq > maxPaidSeq {
				maxPaidSeq = in.Seq
			}
			if in.DueDate > lastPaidDue {
				lastPaidDue = in.DueDate
			}
			continue
		}
		unpaidCount++
		if firstDue == 0 || in.DueDate < firstDue {
			firstDue = in.DueDate
		}
	}
	if firstDue == 0 {
		// Nothing unpaid to anchor on: resume one period after the last payment.
		firstDue = TodaySerial
		if lastPaidDue > 0 {
			firstDue = nextDate(lastPaidDue, d.Freq)
		}
	}

	// Drop the unpaid rows; the paid history stays.
	kept := s.installments[:0]
	for _, in := range s.installments {
		if in.DebtID == d.ID && !in.Paid {
			s.reportError(s.dbDeleteInstallment(in.ID))
			continue
		}
		kept = append(kept, in)
	}
	s.installments = kept

	bal := s.DebtBalance(d.ID)
	if bal <= 0 {
		return // paid off — no schedule left to generate
	}
	payment := d.PaymentAmount
	if !keepPayment && unpaidCount > 0 {
		payment = AmortizedPayment(bal, d.APRBps, unpaidCount, d.Freq)
	}
	newInsts, ok := GenerateLoanSchedule(bal, d.APRBps, payment, firstDue, d.Freq)
	if !ok {
		// The payment no longer amortizes (shouldn't happen — the balance only
		// shrinks); fall back to a single balloon installment so nothing is lost.
		newInsts = []Installment{{Seq: 1, DueDate: firstDue, Amount: bal, Principal: bal}}
	}
	if payment != d.PaymentAmount {
		d.PaymentAmount = payment
		s.reportError(s.dbUpsertDebt(*d))
		for i := range s.debts {
			if s.debts[i].ID == d.ID {
				s.debts[i] = *d
			}
		}
	}
	next := s.maxInstallmentNum() + 1
	for i := range newInsts {
		newInsts[i].DebtID = d.ID
		newInsts[i].ID = "i" + Itoa(next)
		newInsts[i].Seq = maxPaidSeq + i + 1
		next++
		s.reportError(s.dbUpsertInstallment(newInsts[i]))
	}
	s.installments = append(s.installments, newInsts...)
}

// debtOriginMemo labels a debt's origination transaction per kind.
func debtOriginMemo(d Debt) string {
	switch d.Type {
	case DebtLoan:
		return " · loan principal"
	case DebtInformal:
		return " · borrowed"
	default:
		return " · purchase"
	}
}

// ensureLoanInterestCategory returns the default interest Expense category,
// creating it when an older document doesn't have (or deleted) the seed.
func (s *Store) ensureLoanInterestCategory() string {
	if s.AccountByID("loanfees") == nil {
		s.AddAccount(Account{
			ID:    "loanfees",
			Name:  "Loan Fees & Interest",
			Type:  Expense,
			Notes: "Financing cost",
		})
	}
	return "loanfees"
}

// acctFinancedEquity is the shared Equity contra account that backs debt
// originations. Equity accounts are outside both the budget funds pool and the
// spending categories, so booking a purchase against it raises the liability
// (lowering Net Worth) without touching Ready to Assign or any expense category.
const acctFinancedEquity = "equity_financed"

// ensureFinancedEquity returns the financing contra account, creating it once.
func (s *Store) ensureFinancedEquity() string {
	if s.AccountByID(acctFinancedEquity) == nil {
		s.AddAccount(Account{
			ID:    acctFinancedEquity,
			Name:  "Financed Purchases",
			Type:  Equity,
			Notes: "Contra account for debts bought on a payment plan",
		})
	}
	return acctFinancedEquity
}

func debtPayee(d Debt) string {
	if d.Lender != "" {
		return d.Lender
	}
	return d.Name
}

func (s *Store) genDebtID() string {
	max := 0
	for _, d := range s.debts {
		if n := numSuffix(d.ID); n > max {
			max = n
		}
	}
	return "d" + Itoa(max+1)
}

func (s *Store) maxInstallmentNum() int {
	max := 0
	for _, in := range s.installments {
		if n := numSuffix(in.ID); n > max {
			max = n
		}
	}
	return max
}
