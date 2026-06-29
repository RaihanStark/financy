package core

import "sort"

// Debts: tracked liabilities you pay off on a fixed installment schedule. The
// first supported kind is BNPL (Buy Now, Pay Later) — a purchase split into a
// fixed number of monthly installments. A Debt owns a schedule of Installments;
// paying one posts a real Expense transaction (money out → category) so the debt
// shows up in your books and analytics, and the installment is marked paid and
// linked to that transaction.

// Debt kinds. Only BNPL today; the field keeps room for loans/credit later.
const DebtBNPL = "BNPL"

// DebtTypes returns the selectable debt kinds, in order.
func DebtTypes() []string { return []string{DebtBNPL} }

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
	Type         string // DebtBNPL
	Lender       string // e.g. "Shopee PayLater"
	AcctMoney    string // money account installments are paid from
	PurchaseDate int    // when the debt was incurred — the origination date. The
	// whole point of "buy now, pay later" is this differs from the first payment's
	// due date. 0 means "today" at creation time.
	Note string

	AcctLiability string // Liability account mirroring the outstanding balance
	OriginTxnID   string // the purchase transaction that opened the liability
}

// Installment is one scheduled payment of a Debt.
type Installment struct {
	ID      string
	DebtID  string
	Seq     int // 1-based position in the schedule
	DueDate int // Excel serial
	Amount  int // minor units, positive
	Paid    bool
	TxnID   string // the posted (or linked) transaction, set when paid
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
		out = append(out, Installment{Seq: i + 1, DueDate: due, Amount: amt})
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

// AddDebt persists a debt together with its generated schedule. It also creates
// the debt's off-budget Liability account and books the purchase against an Equity
// contra (dated the purchase date, defaulting to today — NOT the first payment's
// due date), so the debt appears in Accounts and Net Worth from the moment you
// incur it, even when the first installment is weeks away.
func (s *Store) AddDebt(d Debt, insts []Installment) {
	if d.ID == "" {
		d.ID = s.genDebtID()
	}
	if d.PurchaseDate == 0 {
		d.PurchaseDate = TodaySerial
	}

	// The Liability account that mirrors the outstanding balance. It's a tracking
	// (off-budget) account: the debt shows in Accounts and Net Worth, but the money
	// you still owe is NOT counted as assignable budget funds. Instead each debt is
	// its own budget envelope (see budget.go) that you fund as you pay it down.
	liab := Account{ID: "debt_" + d.ID, Name: d.Name, Type: Liability, Institution: d.Lender, Notes: d.Type, OffBudget: true}
	s.AddAccount(liab)
	d.AcctLiability = liab.ID

	// Book the purchase as a balance-sheet-only financing event: the liability
	// rises by the schedule total against an Equity contra. Net Worth drops by what
	// you now owe, but Ready to Assign is untouched — the cost is recognised in the
	// budget as you assign to the debt's envelope and pay each installment.
	total := 0
	for _, in := range insts {
		total += in.Amount
	}
	d.OriginTxnID = s.genID()
	s.AddTransaction(Transaction{
		ID:    d.OriginTxnID,
		Date:  d.PurchaseDate,
		Payee: debtPayee(d),
		Memo:  d.Name + " · purchase",
		Posts: PostingsFor(KindExpense, d.AcctLiability, s.ensureFinancedEquity(), total),
	})

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

// UpdateDebt edits a debt's metadata (name, lender, accounts, note). The
// schedule is left untouched. The Liability account's label follows the name /
// lender, and the purchase booking is re-synced in case the category changed.
func (s *Store) UpdateDebt(id string, d Debt) {
	existing := s.DebtByID(id)
	if existing == nil {
		return
	}
	d.ID = id
	d.AcctLiability = existing.AcctLiability
	d.OriginTxnID = existing.OriginTxnID

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
	s.notify()
}

// DeleteDebt removes a debt, its schedule, its Liability account and every
// transaction booked against it (the purchase and any payments) — undoing the
// debt entirely.
func (s *Store) DeleteDebt(id string) {
	d := s.DebtByID(id)
	if d == nil {
		return
	}
	liab := d.AcctLiability
	if liab != "" {
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

// syncDebtOrigination rewrites the purchase transaction so the liability always
// opens at the current total of the schedule (installment amounts can be edited),
// preserving its original date.
func (s *Store) syncDebtOrigination(d *Debt) {
	if d.OriginTxnID == "" {
		return
	}
	date := firstDueOf(s.Installments(d.ID))
	if orig := s.TxnByID(d.OriginTxnID); orig != nil {
		date = orig.Date
	}
	total := 0
	for _, in := range s.Installments(d.ID) {
		total += in.Amount
	}
	s.UpdateTransaction(d.OriginTxnID, Transaction{
		ID:    d.OriginTxnID,
		Date:  date,
		Payee: debtPayee(*d),
		Memo:  d.Name + " · purchase",
		Posts: PostingsFor(KindExpense, d.AcctLiability, s.ensureFinancedEquity(), total),
	})
}

func firstDueOf(insts []Installment) int {
	if len(insts) > 0 {
		return insts[0].DueDate
	}
	return TodaySerial
}

// UpdateInstallment changes one installment's due date and amount — schedules
// aren't always equal (a final balloon payment, a skipped month, a fee). If the
// installment is already paid, its linked transaction is rewritten to match so
// the books stay in sync. Returns whether it updated.
func (s *Store) UpdateInstallment(instID string, dueDate, amount int) bool {
	in := s.installmentByID(instID)
	if in == nil || dueDate == 0 || amount <= 0 {
		return false
	}
	in.DueDate = dueDate
	in.Amount = amount
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
			Posts: PostingsFor(KindExpense, d.AcctMoney, d.AcctLiability, amount),
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
		Posts: PostingsFor(KindExpense, d.AcctMoney, d.AcctLiability, in.Amount),
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
	orig := append([]Posting(nil), t.Posts...)
	upd := *t
	upd.Posts = PostingsFor(KindExpense, d.AcctMoney, d.AcctLiability, paid)
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

// DebtBuckets rolls every unpaid installment across all debts into timing buckets.
func (s *Store) DebtBuckets(asOf int) DebtBuckets {
	var b DebtBuckets
	thisM := FmtSerialMonth(asOf)
	nextM := ShiftMonth(thisM, 1)
	for _, in := range s.installments {
		if in.Paid {
			continue
		}
		b.Outstanding += in.Amount
		switch {
		case in.DueDate <= asOf:
			b.Due += in.Amount
		case FmtSerialMonth(in.DueDate) == thisM:
			b.ThisMonth += in.Amount
		case FmtSerialMonth(in.DueDate) == nextM:
			b.NextMonth += in.Amount
		}
	}
	return b
}

// DueDebtCount counts unpaid installments due on or before asOf across all debts.
func (s *Store) DueDebtCount(asOf int) int {
	n := 0
	for _, in := range s.installments {
		if !in.Paid && in.DueDate <= asOf {
			n++
		}
	}
	return n
}

// TotalDebtOutstanding is the sum of every unpaid installment — the money still
// owed across all tracked debts.
func (s *Store) TotalDebtOutstanding() int {
	sum := 0
	for _, in := range s.installments {
		if !in.Paid {
			sum += in.Amount
		}
	}
	return sum
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
