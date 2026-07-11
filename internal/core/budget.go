package core

import (
	"sort"
	"strings"
	"time"
)

// Budget — zero-based ("envelope") budgeting, modelled on YNAB.
//
// The journal stays the single source of truth for money; budgeting adds one
// extra fact per (month, category): how much money you ASSIGNED to that
// category that month. Everything else is derived:
//
//	Activity (month)  = net flow through the category that month
//	                    (outflow shown negative, like YNAB).
//	Available (month) = Σ over all months ≤ this one of (Assigned + Activity).
//	                    i.e. the running envelope balance that rolls forward.
//	Ready to Assign   = funds received so far − everything assigned so far.
//
// Categories are the Expense accounts. Assigned amounts live in the `budget`
// table keyed by "YYYY-MM" + category id.

// BudgetCategory is one category's envelope for a given month.
type BudgetCategory struct {
	ID        string
	Name      string
	Assigned  int  // assigned to this category THIS month
	Activity  int  // net flow this month; spending is negative
	Available int  // rolling envelope balance through this month
	IsDebt    bool // true for a debt's payment envelope (backed by its liability account)
}

// Overspent reports whether the envelope has gone negative.
func (b BudgetCategory) Overspent() bool { return b.Available < 0 }

// BudgetMonth is the full budget view for a single month.
type BudgetMonth struct {
	Month          string // "YYYY-MM"
	Funds          int    // on-budget assets net of card debt, as of this month
	ReadyToAssign  int
	TotalAssigned  int
	TotalActivity  int
	TotalAvailable int
	Categories     []BudgetCategory
}

// ---- month-key helpers ----

// CurrentMonthKey is the month containing today, as "YYYY-MM".
func CurrentMonthKey() string { return FmtSerialMonth(TodaySerial) }

// MonthEditable reports whether a budget month may still be changed. Only the
// current month and future months are editable; past months are locked, so a
// budget you've already lived through can't be rewritten.
func MonthEditable(month string) bool {
	return month >= CurrentMonthKey()
}

// ShiftMonth moves a "YYYY-MM" key by delta months (delta may be negative).
func ShiftMonth(month string, delta int) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		t = SerialToTime(TodaySerial)
	}
	return t.AddDate(0, delta, 0).Format("2006-01")
}

// FmtMonthLong renders a "YYYY-MM" key as e.g. "June 2026".
func FmtMonthLong(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.Format("January 2006")
}

func monthKey(month, catID string) string { return month + "|" + catID }

// deleteAssignmentsFor drops every budget assignment belonging to a category,
// in memory and on disk. Called when the category account is removed.
func (s *Store) deleteAssignmentsFor(catID string) {
	for k := range s.assignments {
		i := strings.IndexByte(k, '|')
		if i >= 0 && k[i+1:] == catID {
			delete(s.assignments, k)
		}
	}
	if s.db != nil {
		if _, err := s.db.Exec(`DELETE FROM budget WHERE category_id = ?`, catID); err != nil {
			s.reportError(err)
		}
	}
}

// ---- assignment access / mutation ----

// Assigned returns the amount assigned to a category for a month (0 if none).
func (s *Store) Assigned(month, catID string) int {
	if s.assignments == nil {
		return 0
	}
	return s.assignments[monthKey(month, catID)]
}

// SetAssigned records (or clears, when amount == 0) a category's assignment for
// a month and writes through to disk.
func (s *Store) SetAssigned(month, catID string, amount int) {
	if s.assignments == nil {
		s.assignments = map[string]int{}
	}
	key := monthKey(month, catID)
	if amount == 0 {
		if _, ok := s.assignments[key]; !ok {
			return
		}
		delete(s.assignments, key)
		if err := s.dbDeleteAssignment(month, catID); err != nil {
			s.reportError(err)
			return
		}
	} else {
		s.assignments[key] = amount
		if err := s.dbUpsertAssignment(month, catID, amount); err != nil {
			s.reportError(err)
			return
		}
	}
	s.notify()
}

// ---- derived flows ----

// categorySpentInMonth sums a category's postings dated within one month.
func (s *Store) categorySpentInMonth(catID, month string) int {
	sum := 0
	for _, t := range s.txns {
		if FmtSerialMonth(t.Date) != month {
			continue
		}
		for _, p := range t.Posts {
			if p.AccountID == catID {
				sum += p.Amount
			}
		}
	}
	return sum
}

// spendIndex sums every category posting into a "categoryID|YYYY-MM" → amount map
// in a single pass, so the month-by-month simulation can run without rescanning
// the journal per category.
//
// Expense accounts count every posting. A debt's liability account is also a
// category (its payment envelope), but there only pay-downs count: an installment
// payment posts a POSITIVE amount into the liability (drawing it toward zero),
// whereas the origination posts a negative amount. Counting only positive postings
// keeps the envelope's activity equal to what you actually paid that month.
func (s *Store) spendIndex() map[string]int {
	expense := map[string]bool{}
	for _, c := range s.ExpenseAccounts() {
		expense[c.ID] = true
	}
	debt := map[string]bool{}
	for _, d := range s.debts {
		// Revolving debts carry no envelope: a card purchase already depletes a
		// spending envelope, so counting the payment too would double-budget it.
		if d.HasEnvelope() && d.AcctLiability != "" {
			debt[d.AcctLiability] = true
		}
	}
	idx := map[string]int{}
	for _, t := range s.txns {
		m := FmtSerialMonth(t.Date)
		for _, p := range t.Posts {
			switch {
			case expense[p.AccountID]:
				idx[p.AccountID+"|"+m] += p.Amount
			case debt[p.AccountID] && p.Amount > 0:
				idx[p.AccountID+"|"+m] += p.Amount
			}
		}
	}
	return idx
}

// budgetCategories is the ordered set of category accounts the envelope view is
// built from: the Expense accounts followed by each debt's liability account
// (its payment envelope).
func (s *Store) budgetCategories() []Account {
	cats := s.ExpenseAccounts()
	for _, d := range s.debts {
		if !d.HasEnvelope() {
			continue
		}
		if a := s.AccountByID(d.AcctLiability); a != nil {
			cats = append(cats, *a)
		}
	}
	return cats
}

// earliestMonth is the earliest "YYYY-MM" appearing in any transaction or
// assignment; "" when the journal and budget are both empty.
func (s *Store) earliestMonth() string {
	best := ""
	for _, t := range s.txns {
		if m := FmtSerialMonth(t.Date); best == "" || m < best {
			best = m
		}
	}
	for k := range s.assignments {
		if i := strings.IndexByte(k, '|'); i >= 0 {
			if m := k[:i]; best == "" || m < best {
				best = m
			}
		}
	}
	return best
}

// budgetFundsThrough is the net money available to the budget as of the end of
// a month: the summed balance of every on-budget Asset and Liability account,
// counting only transactions dated on or before that month.
//
// Including on-budget liabilities (credit cards, stored as negative debt) is
// what keeps the books honest: spending on a card depletes a category envelope
// AND grows card debt by the same amount, so the funds pool drops in step and
// Ready to Assign is left unchanged. Tracking accounts (OffBudget) — investments,
// a mortgage — are excluded so they don't masquerade as assignable cash.
func (s *Store) budgetFundsThrough(month string) int {
	onBudget := map[string]bool{}
	for _, a := range s.accounts {
		if (a.Type == Asset || a.Type == Liability) && !a.OffBudget {
			onBudget[a.ID] = true
		}
	}
	sum := 0
	for _, t := range s.txns {
		if FmtSerialMonth(t.Date) > month {
			continue
		}
		for _, p := range t.Posts {
			if onBudget[p.AccountID] {
				sum += p.Amount
			}
		}
	}
	return sum
}

// budgetCompute runs the month-by-month envelope simulation through `month` and
// returns each category's view plus the sum of all Available balances.
//
// YNAB's carry rule: only a POSITIVE Available rolls into the next month. A
// category that overspends cash shows red for that month, then resets to zero
// at the next rollover — the shortfall is absorbed by Ready to Assign rather
// than dragging the envelope negative forever.
func (s *Store) budgetCompute(month string) (cats []BudgetCategory, sumAvail int) {
	idx := s.spendIndex()
	start := s.earliestMonth()
	if start == "" || start > month {
		start = month
	}
	for _, c := range s.budgetCategories() {
		avail := 0
		for m := start; m <= month; m = ShiftMonth(m, 1) {
			if avail < 0 {
				avail = 0 // only positive balances carry forward
			}
			avail += s.Assigned(m, c.ID) - idx[c.ID+"|"+m]
		}
		spentMonth := idx[c.ID+"|"+month]
		bc := BudgetCategory{
			ID:        c.ID,
			Name:      c.Name,
			Assigned:  s.Assigned(month, c.ID),
			Activity:  -spentMonth,
			Available: avail,
			IsDebt:    c.Type == Liability,
		}
		cats = append(cats, bc)
		sumAvail += avail
	}
	return cats, sumAvail
}

// latestBudgetMonth is the furthest-out month that carries any budget meaning:
// the later of this calendar month and the newest month holding a transaction or
// an assignment (so money assigned to a future month is taken into account).
func (s *Store) latestBudgetMonth() string {
	best := CurrentMonthKey()
	for _, t := range s.txns {
		if m := FmtSerialMonth(t.Date); m > best {
			best = m
		}
	}
	for k := range s.assignments {
		if i := strings.IndexByte(k, '|'); i >= 0 {
			if m := k[:i]; m > best {
				best = m
			}
		}
	}
	return best
}

// committedFunds is the money tied up in category envelopes across the whole
// timeline: the sum of every category's running balance, counting only the
// POSITIVE ones. A negative envelope (this month's cash overspending) isn't
// money you can re-assign — that cash is already gone — so it contributes 0.
func (s *Store) committedFunds() int {
	cats, _ := s.budgetCompute(s.latestBudgetMonth())
	sum := 0
	for _, c := range cats {
		if c.Available > 0 {
			sum += c.Available
		}
	}
	return sum
}

// ReadyToAssign is a single, global figure — the same whatever month you view:
// your on-budget funds (assets net of card debt) minus everything you've already
// committed to a category in ANY month. Assign money to a future month and this
// drops today, because that money now has a job; the budgeting goal is to assign
// until it reaches zero.
func (s *Store) ReadyToAssign(string) int {
	return s.budgetFundsThrough(s.latestBudgetMonth()) - s.committedFunds()
}

// BudgetFor builds the complete budget view for a month. The category rows are
// that month's snapshot, while Funds and Ready to Assign are the global position
// (identical on every month).
func (s *Store) BudgetFor(month string) BudgetMonth {
	cats, sumAvail := s.budgetCompute(month)
	bm := BudgetMonth{Month: month, Categories: cats, TotalAvailable: sumAvail}
	for _, c := range cats {
		bm.TotalAssigned += c.Assigned
		bm.TotalActivity += c.Activity
	}
	bm.Funds = s.budgetFundsThrough(s.latestBudgetMonth())
	bm.ReadyToAssign = bm.Funds - s.committedFunds()
	return bm
}

// ---- quick-budget suggestions (YNAB "auto-assign") ----

// AssignedLastMonth is what was assigned to a category the previous month.
func (s *Store) AssignedLastMonth(month, catID string) int {
	return s.Assigned(ShiftMonth(month, -1), catID)
}

// SpentLastMonth is the (positive) amount spent in a category last month.
func (s *Store) SpentLastMonth(month, catID string) int {
	return s.categorySpentInMonth(catID, ShiftMonth(month, -1))
}

// AverageSpent is the average monthly spend (positive) over the n months
// preceding `month` (rounded to the nearest minor unit).
func (s *Store) AverageSpent(month, catID string, n int) int {
	if n <= 0 {
		return 0
	}
	total := 0
	for i := 1; i <= n; i++ {
		total += s.categorySpentInMonth(catID, ShiftMonth(month, -i))
	}
	// round-half-up division for non-negative totals
	return (total + n/2) / n
}

// AutoAssignChange is one category's proposed adjustment from Auto-Assign
// (copying last month's assignment into `month`). From is what is currently
// assigned this month; To is what it would become.
type AutoAssignChange struct {
	CatID string
	Name  string
	From  int
	To    int
}

// PreviewAutoAssign reports the changes AssignLastMonthsAmounts would make,
// without applying them. Only categories whose assignment would actually
// change are included; order matches ExpenseAccounts. This lets the UI show
// the user exactly what will be assigned/adjusted before they confirm.
func (s *Store) PreviewAutoAssign(month string) []AutoAssignChange {
	prev := ShiftMonth(month, -1)
	var out []AutoAssignChange
	for _, c := range s.ExpenseAccounts() {
		to := s.Assigned(prev, c.ID)
		if to == 0 {
			continue
		}
		from := s.Assigned(month, c.ID)
		if from == to {
			continue
		}
		out = append(out, AutoAssignChange{CatID: c.ID, Name: c.Name, From: from, To: to})
	}
	return out
}

// AssignLastMonthsAmounts copies every category's previous-month assignment into
// `month`, in a single notification. Returns how many categories were set. The
// change set matches PreviewAutoAssign; callers should preview and confirm with
// the user before calling this, since it overwrites existing assignments.
func (s *Store) AssignLastMonthsAmounts(month string) int {
	prev := ShiftMonth(month, -1)
	n := 0
	if s.assignments == nil {
		s.assignments = map[string]int{}
	}
	for _, c := range s.ExpenseAccounts() {
		amt := s.Assigned(prev, c.ID)
		if amt == 0 {
			continue
		}
		key := monthKey(month, c.ID)
		if s.assignments[key] == amt {
			continue
		}
		s.assignments[key] = amt
		if err := s.dbUpsertAssignment(month, c.ID, amt); err != nil {
			s.reportError(err)
			continue
		}
		n++
	}
	if n > 0 {
		s.notify()
	}
	return n
}

// BudgetMonthsWithData lists every month that has either an assignment or a
// categorized transaction, plus the current month, newest first.
func (s *Store) BudgetMonthsWithData() []string {
	set := map[string]bool{CurrentMonthKey(): true}
	for k := range s.assignments {
		if i := strings.IndexByte(k, '|'); i >= 0 {
			set[k[:i]] = true
		}
	}
	expense := map[string]bool{}
	for _, c := range s.ExpenseAccounts() {
		expense[c.ID] = true
	}
	for _, t := range s.txns {
		for _, p := range t.Posts {
			if expense[p.AccountID] {
				set[FmtSerialMonth(t.Date)] = true
				break
			}
		}
	}
	out := make([]string, 0, len(set))
	for m := range set {
		out = append(out, m)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out
}
