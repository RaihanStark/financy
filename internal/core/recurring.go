package core

import (
	"sort"
	"time"
)

// Recurring transactions: templates that describe a transaction plus a schedule.
// They never post by themselves — when due, the UI prompts and posts the real
// transaction(s) on confirmation. This keeps the user in control and lets us flag
// entries that already exist (e.g. paid manually or imported) as duplicates.

// Transaction kinds, matching the entry form's options.
const (
	KindExpense  = "Expense"
	KindIncome   = "Income"
	KindTransfer = "Transfer"
)

// Frequencies returns the selectable recurrence frequencies, in order.
func Frequencies() []string {
	return []string{"Weekly", "Biweekly", "Monthly", "Quarterly", "Yearly"}
}

// Recurring is one scheduled-transaction template.
type Recurring struct {
	ID      string
	Kind    string // KindExpense | KindIncome | KindTransfer
	AcctA   string // money account (paid from / deposited to / transfer from)
	AcctB   string // category, or transfer-to account
	Amount  int    // minor units, positive
	Payee   string
	Memo    string
	Freq    string // one of Frequencies()
	NextDue int    // Excel serial of the next occurrence
	Enabled bool
}

// PostingsFor builds the balanced postings for a kind, shared by the entry form
// and recurring so the two can't drift. amt is a positive minor-unit amount.
func PostingsFor(kind, acctA, acctB string, amt int) []Posting {
	if kind == KindIncome {
		return []Posting{P(acctA, amt), P(acctB, -amt)}
	}
	// Expense and Transfer share the same shape.
	return []Posting{P(acctB, amt), P(acctA, -amt)}
}

// ---- accessors / CRUD ----

func (s *Store) Recurrings() []Recurring { return append([]Recurring(nil), s.recurring...) }

func (s *Store) RecurringByID(id string) *Recurring {
	for i := range s.recurring {
		if s.recurring[i].ID == id {
			return &s.recurring[i]
		}
	}
	return nil
}

func (s *Store) AddRecurring(r Recurring) {
	if r.ID == "" {
		r.ID = s.genRecurringID()
	}
	if err := s.dbUpsertRecurring(r); err != nil {
		s.reportError(err)
		return
	}
	s.recurring = append(s.recurring, r)
	s.notify()
}

func (s *Store) UpdateRecurring(id string, r Recurring) {
	for i := range s.recurring {
		if s.recurring[i].ID == id {
			r.ID = id
			if err := s.dbUpsertRecurring(r); err != nil {
				s.reportError(err)
				return
			}
			s.recurring[i] = r
			s.notify()
			return
		}
	}
}

func (s *Store) DeleteRecurring(id string) {
	for i := range s.recurring {
		if s.recurring[i].ID == id {
			if err := s.dbDeleteRecurring(id); err != nil {
				s.reportError(err)
				return
			}
			s.recurring = append(s.recurring[:i], s.recurring[i+1:]...)
			s.notify()
			return
		}
	}
}

func (s *Store) genRecurringID() string {
	max := 0
	for _, r := range s.recurring {
		if n := numSuffix(r.ID); n > max {
			max = n
		}
	}
	return "r" + Itoa(max+1)
}

func numSuffix(id string) int {
	i := len(id)
	for i > 0 && id[i-1] >= '0' && id[i-1] <= '9' {
		i--
	}
	return atoiSafe(id[i:])
}

// ---- scheduling ----

// nextDate advances a serial date by one period of freq, keeping the day of
// month for month-based frequencies (clamped to the month's length, so monthly
// from Jan 31 lands on Feb 28/29 rather than overflowing into March).
func nextDate(serial int, freq string) int {
	t := SerialToTime(serial)
	switch freq {
	case "Weekly":
		t = t.AddDate(0, 0, 7)
	case "Biweekly":
		t = t.AddDate(0, 0, 14)
	case "Quarterly":
		t = addMonths(t, 3)
	case "Yearly":
		t = addMonths(t, 12)
	default: // Monthly
		t = addMonths(t, 1)
	}
	return TimeToSerial(t)
}

func addMonths(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	if last := first.AddDate(0, 1, -1).Day(); d > last {
		d = last
	}
	return time.Date(first.Year(), first.Month(), d, 0, 0, 0, 0, time.UTC)
}

// ---- due / post ----

// DueItem is one occurrence of a recurring template that's due. The user either
// posts Txn (a new transaction) or links it to one of Candidates — an existing
// transaction that already paid it.
type DueItem struct {
	RecurringID string
	Kind        string
	Date        int
	Payee       string
	Amount      int
	AcctA       string
	Txn         Transaction   // the new transaction to post
	Candidates  []Transaction // relevant existing transactions to link to (best first)
	AutoMatch   bool          // Candidates[0] closely matches the amount → pre-select it
}

// PendingRecurring expands every enabled, due template into its occurrences up to
// asOf (catching up missed periods), each as a ready transaction and flagged if a
// matching entry already exists.
func (s *Store) PendingRecurring(asOf int) []DueItem {
	var out []DueItem
	for _, r := range s.recurring {
		if !r.Enabled {
			continue
		}
		for d := r.NextDue; d <= asOf; d = nextDate(d, r.Freq) {
			cands, auto := s.recurringCandidates(r.AcctA, r.Amount, d, freqWindow(r.Freq))
			out = append(out, DueItem{
				RecurringID: r.ID,
				Kind:        r.Kind,
				Date:        d,
				Payee:       r.Payee,
				Amount:      r.Amount,
				AcctA:       r.AcctA,
				Txn:         Transaction{Date: d, Payee: r.Payee, Memo: r.Memo, Posts: PostingsFor(r.Kind, r.AcctA, r.AcctB, r.Amount)},
				Candidates:  cands,
				AutoMatch:   auto,
			})
		}
	}
	return out
}

// PostRecurring posts the chosen occurrences as real transactions, then advances
// every due template's schedule past asOf so skipped/duplicate occurrences don't
// keep prompting. Returns how many transactions were posted.
func (s *Store) PostRecurring(asOf int, post []DueItem) (int, error) {
	txns := make([]Transaction, 0, len(post))
	for _, it := range post {
		txns = append(txns, it.Txn)
	}
	n, err := s.AddTransactions(txns)
	if err != nil {
		return 0, err
	}
	for i := range s.recurring {
		r := &s.recurring[i]
		if !r.Enabled || r.NextDue > asOf {
			continue
		}
		for r.NextDue <= asOf {
			r.NextDue = nextDate(r.NextDue, r.Freq)
		}
		s.reportError(s.dbUpsertRecurring(*r))
	}
	s.notify()
	return n, nil
}

// PostRecurringNow posts a single occurrence of a template immediately (dated
// `date` — e.g. paying a bill early before its due date) and advances the
// template's schedule by one period. Returns whether it posted.
func (s *Store) PostRecurringNow(id string, date int) bool {
	r := s.RecurringByID(id)
	if r == nil {
		return false
	}
	txn := Transaction{Date: date, Payee: r.Payee, Memo: r.Memo,
		Posts: PostingsFor(r.Kind, r.AcctA, r.AcctB, r.Amount)}
	if _, err := s.AddTransactions([]Transaction{txn}); err != nil {
		return false
	}
	r.NextDue = nextDate(r.NextDue, r.Freq)
	s.reportError(s.dbUpsertRecurring(*r))
	s.notify()
	return true
}

// NextDate reports the occurrence after serial for a frequency (exported for the
// UI to preview where the schedule will move).
func NextDate(serial int, freq string) int { return nextDate(serial, freq) }

// recurringCandidates returns existing transactions on acctA within window days
// of date — relevant entries this occurrence might already be recorded as. The
// amount need not match exactly (bills vary); results are ranked by amount
// closeness then date closeness. autoMatch is true when the best candidate's
// amount is within tolerance — confident enough to pre-select.
func (s *Store) recurringCandidates(acctA string, amount, date, window int) (cands []Transaction, autoMatch bool) {
	type scored struct {
		t        Transaction
		amtDiff  int
		dateDiff int
	}
	var list []scored
	for _, t := range s.txns {
		sum, touches := acctSum(t, acctA)
		if !touches || absInt(t.Date-date) > window {
			continue
		}
		list = append(list, scored{t, absInt(absInt(sum) - absInt(amount)), absInt(t.Date - date)})
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].amtDiff != list[j].amtDiff {
			return list[i].amtDiff < list[j].amtDiff
		}
		return list[i].dateDiff < list[j].dateDiff
	})
	if len(list) > 8 { // keep the dropdown manageable
		list = list[:8]
	}
	out := make([]Transaction, len(list))
	for i := range list {
		out[i] = list[i].t
	}
	return out, len(list) > 0 && list[0].amtDiff <= amountTolerance(amount)
}

// RecurringCandidates returns transactions near `date` on accountID that could
// correspond to a recurring occurrence (for manual linking in the early-post
// dialog), and whether the closest is a confident amount match.
func (s *Store) RecurringCandidates(accountID string, amount, date int) ([]Transaction, bool) {
	return s.recurringCandidates(accountID, amount, date, 31)
}

func acctSum(t Transaction, accountID string) (sum int, touches bool) {
	for _, p := range t.Posts {
		if p.AccountID == accountID {
			sum += p.Amount
			touches = true
		}
	}
	return sum, touches
}

// amountTolerance is the amount window (5%, with a small floor) within which a
// candidate is considered a confident match.
func amountTolerance(amount int) int {
	if tol := absInt(amount) / 20; tol > 100 {
		return tol
	}
	return 100
}

// AdvanceRecurring moves a template's schedule one period forward without posting
// — used when an occurrence is satisfied by an existing transaction.
func (s *Store) AdvanceRecurring(id string) bool {
	r := s.RecurringByID(id)
	if r == nil {
		return false
	}
	r.NextDue = nextDate(r.NextDue, r.Freq)
	s.reportError(s.dbUpsertRecurring(*r))
	s.notify()
	return true
}

// freqWindow is the duplicate-match tolerance (days) for a frequency — roughly
// half a period, so a bill paid a few days early/late still matches.
func freqWindow(freq string) int {
	switch freq {
	case "Weekly":
		return 3
	case "Biweekly":
		return 7
	case "Quarterly":
		return 45
	case "Yearly":
		return 60
	default: // Monthly
		return 15
	}
}
