package core

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Guided CSV import: turn a single-account bank/card export (single-entry rows)
// into balanced double-entry transactions. Pure logic, no UI — the wizard
// supplies an ImportSpec (which columns mean what), and BuildImport produces
// reviewable rows. Nothing here writes to disk; committing is AddTransactions.

// AmountMode is how a row's amount is represented in the source file.
type AmountMode int

const (
	AmountSingle      AmountMode = iota // one signed column
	AmountDebitCredit                   // separate debit (out) + credit (in) columns
)

// ImportSpec is the user-confirmed mapping from the guided wizard. Column fields
// are 0-based indices into a record; -1 means "none".
type ImportSpec struct {
	AccountID string // money account to import into
	HasHeader bool

	DateCol  int
	PayeeCol int
	MemoCol  int

	AmountMode   AmountMode
	AmountCol    int // AmountSingle
	DebitCol     int // AmountDebitCredit (money out)
	CreditCol    int // AmountDebitCredit (money in)
	NegateAmount bool

	DateLayout string // Go time layout; empty = try known layouts

	IncomeCatID  string // contra category for money-in rows
	ExpenseCatID string // contra category for money-out rows
}

// ImportStatus classifies a parsed row.
type ImportStatus int

const (
	RowOK ImportStatus = iota
	RowError
	RowDuplicate
)

// ImportRow is one source row with its computed transaction and status.
type ImportRow struct {
	Line    int // 1-based data line (after header)
	Raw     []string
	Date    int
	Payee   string
	Memo    string
	Amount     int    // signed minor units; + = money in to the account
	CategoryID string // contra category (auto-matched or default); editable in preview
	Txn        Transaction
	Status     ImportStatus
	Reason     string // set when Status != RowOK
	Include    bool   // default selection for committing
}

// dateLayouts are tried in order when ImportSpec.DateLayout is empty.
var dateLayouts = []string{
	"2006-01-02", "2006/01/02", "01/02/2006", "02/01/2006", "1/2/2006",
	"2/1/2006", "01-02-2006", "02-01-2006", "02.01.2006", "2006.01.02",
	"01/02/06", "02/01/06", "Jan 2, 2006", "2 Jan 2006", "02 Jan 2006",
	"Jan-02-2006", "20060102",
}

// ParseCSV reads raw bytes into records, auto-detecting the delimiter and
// tolerating ragged rows / a UTF-8 BOM.
func ParseCSV(b []byte) ([][]string, error) {
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF}) // strip BOM
	r := csv.NewReader(bytes.NewReader(b))
	r.Comma = detectDelimiter(b)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	return r.ReadAll()
}

func detectDelimiter(b []byte) rune {
	line, _, _ := bytes.Cut(b, []byte{'\n'})
	counts := map[rune]int{',': 0, ';': 0, '\t': 0, '|': 0}
	for _, c := range string(line) {
		if _, ok := counts[c]; ok {
			counts[c]++
		}
	}
	best, bestN := ',', -1
	for _, d := range []rune{',', ';', '\t', '|'} {
		if counts[d] > bestN {
			best, bestN = d, counts[d]
		}
	}
	return best
}

// GuessSpec auto-detects a mapping from the header row and a few sample data
// rows. The wizard pre-fills its controls with this and lets the user adjust.
func GuessSpec(header []string, sample [][]string) ImportSpec {
	spec := ImportSpec{
		HasHeader: true, DateCol: -1, PayeeCol: -1, MemoCol: -1,
		AmountCol: -1, DebitCol: -1, CreditCol: -1, AmountMode: AmountSingle,
	}
	has := func(h, sub string) bool { return strings.Contains(h, sub) }
	for i, raw := range header {
		h := strings.ToLower(strings.TrimSpace(raw))
		switch {
		case has(h, "date") && spec.DateCol < 0:
			spec.DateCol = i
		case (h == "payee" || has(h, "description") || has(h, "merchant") ||
			has(h, "narrative") || has(h, "details") || has(h, "name")) && spec.PayeeCol < 0:
			spec.PayeeCol = i
		case has(h, "memo") || has(h, "note") || has(h, "reference"):
			spec.MemoCol = i
		case has(h, "debit") || has(h, "withdraw") || has(h, "paid out") || has(h, "money out"):
			spec.DebitCol = i
		case has(h, "credit") || has(h, "deposit") || has(h, "paid in") || has(h, "money in"):
			spec.CreditCol = i
		case has(h, "amount") && spec.AmountCol < 0:
			spec.AmountCol = i
		}
	}
	if spec.DebitCol >= 0 && spec.CreditCol >= 0 {
		spec.AmountMode = AmountDebitCredit
	}
	if spec.DateCol >= 0 {
		var samples []string
		for _, r := range sample {
			samples = append(samples, field(r, spec.DateCol))
		}
		spec.DateLayout = guessDateLayout(samples)
	}
	return spec
}

// guessDateLayout returns the first known layout that parses every non-empty
// sample, or "" if none fits all.
func guessDateLayout(samples []string) string {
	for _, layout := range dateLayouts {
		ok, any := true, false
		for _, s := range samples {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			any = true
			if _, err := time.Parse(layout, s); err != nil {
				ok = false
				break
			}
		}
		if ok && any {
			return layout
		}
	}
	return ""
}

// BuildImport turns records into reviewable rows per the spec, flagging rows
// that fail to parse or look like duplicates of existing transactions.
func (s *Store) BuildImport(records [][]string, spec ImportSpec) []ImportRow {
	dup := s.importDupKeys(spec.AccountID)
	learned := s.learnedPayees()
	start := 0
	if spec.HasHeader {
		start = 1
	}
	out := make([]ImportRow, 0, len(records))
	for idx := start; idx < len(records); idx++ {
		rec := records[idx]
		row := ImportRow{Line: idx - start + 1, Raw: rec}

		date, ok := parseDateWith(field(rec, spec.DateCol), spec.DateLayout)
		if !ok {
			row.Status, row.Reason = RowError, "unrecognized date"
			out = append(out, row)
			continue
		}
		amt, ok := rowAmount(rec, spec)
		if !ok {
			row.Status, row.Reason = RowError, "unrecognized amount"
			out = append(out, row)
			continue
		}
		if spec.NegateAmount {
			amt = -amt
		}
		if amt == 0 {
			row.Status, row.Reason = RowError, "zero amount"
			out = append(out, row)
			continue
		}

		row.Date = date
		row.Payee = strings.TrimSpace(field(rec, spec.PayeeCol))
		row.Memo = strings.TrimSpace(field(rec, spec.MemoCol))
		row.Amount = amt

		// Default contra by direction; upgrade to a previously-used category if we
		// recognize the payee from already-categorized history.
		catID := spec.ExpenseCatID
		if amt >= 0 {
			catID = spec.IncomeCatID
		}
		if m := s.matchCategory(row.Payee, amt, learned); m != "" {
			catID = m
		}
		row.CategoryID = catID
		row.Txn = BuildImportTxn(spec.AccountID, catID, date, row.Payee, row.Memo, amt)

		if dup[dupKey(date, row.Payee, amt)] {
			row.Status, row.Reason, row.Include = RowDuplicate, "already in this account", false
		} else {
			row.Status, row.Include = RowOK, true
		}
		out = append(out, row)
	}
	return out
}

// BuildImportTxn produces a balanced 2-posting transaction: the money account
// plus an explicit contra category, following the app's sign conventions. The
// preview uses this to rebuild a row when the user changes its category.
func BuildImportTxn(accountID, categoryID string, date int, payee, memo string, amt int) Transaction {
	var posts []Posting
	if amt >= 0 { // money in → income
		posts = []Posting{P(accountID, amt), P(categoryID, -amt)}
	} else { // money out → expense
		posts = []Posting{P(categoryID, -amt), P(accountID, amt)}
	}
	return Transaction{Date: date, Payee: payee, Memo: memo, Posts: posts}
}

// payeeCategoryMap learns, from already-categorized transactions, which contra
// category each payee was last assigned — so import can auto-categorize
// recurring payees. Only clean single-category entries are learned, and the
// catch-all Uncategorized categories are ignored so real choices always win.
func (s *Store) payeeCategoryMap() map[string]string {
	m := make(map[string]string)
	for _, t := range s.txns {
		payee := normPayee(t.Payee)
		if payee == "" {
			continue
		}
		cat, n := "", 0
		for _, p := range t.Posts {
			if a := s.AccountByID(p.AccountID); a != nil && (a.Type == Income || a.Type == Expense) {
				cat, n = p.AccountID, n+1
			}
		}
		if n == 1 && cat != "uncat_income" && cat != "uncat_expense" {
			m[payee] = cat // later (more recent) entries win
		}
	}
	return m
}

func normPayee(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// directionMatches reports whether a category's type fits the money direction:
// income for money in, expense for money out.
func directionMatches(typ AcctType, amt int) bool {
	if amt >= 0 {
		return typ == Income
	}
	return typ == Expense
}

// learnedPayee is one known payee→category association, pre-tokenized for fuzzy
// matching.
type learnedPayee struct {
	norm   string
	tokens map[string]bool
	catID  string
}

// learnedPayees returns the known payee→category associations (sorted, so
// matching is deterministic on ties).
func (s *Store) learnedPayees() []learnedPayee {
	m := s.payeeCategoryMap()
	out := make([]learnedPayee, 0, len(m))
	for norm, cat := range m {
		out = append(out, learnedPayee{norm: norm, tokens: tokenSet(norm), catID: cat})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].norm < out[j].norm })
	return out
}

// matchCategory finds the best learned category for an import payee, trying an
// exact match first, then token-subset / substring matches. Only categories
// whose type fits the money direction are considered. Returns "" if nothing
// matches confidently.
func (s *Store) matchCategory(payee string, amt int, learned []learnedPayee) string {
	np := normPayee(payee)
	if np == "" {
		return ""
	}
	pt := tokenSet(np)
	best, bestScore := "", 0
	for _, lp := range learned {
		if a := s.AccountByID(lp.catID); a == nil || !directionMatches(a.Type, amt) {
			continue
		}
		score := matchScore(np, pt, lp.norm, lp.tokens)
		if lp.norm == np {
			score = 1000 // exact match wins decisively
		}
		if score > bestScore {
			best, bestScore = lp.catID, score
		}
	}
	return best
}

// matchScore rates how well a known payee matches an import payee. A match
// requires either one token set to be a subset of the other (with ≥4 shared
// characters, so generic single-token overlaps don't trigger) or one normalized
// name to contain the other. Higher = more specific.
func matchScore(np string, pt map[string]bool, kNorm string, kt map[string]bool) int {
	if len(kt) == 0 {
		return 0
	}
	shared := 0
	for tok := range kt {
		if pt[tok] {
			shared += len(tok)
		}
	}
	if (allIn(kt, pt) || allIn(pt, kt)) && shared >= 4 {
		return shared
	}
	if len(kNorm) >= 4 && (strings.Contains(np, kNorm) || strings.Contains(kNorm, np)) {
		return 4
	}
	return 0
}

// allIn reports whether every key in a is present in b (a ⊆ b); false if a is empty.
func allIn(a, b map[string]bool) bool {
	if len(a) == 0 {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// tokenSet splits a normalized payee into significant tokens: alphanumeric runs
// of length ≥3 that aren't purely numeric (drops store numbers, "sq", "co", …).
func tokenSet(norm string) map[string]bool {
	set := make(map[string]bool)
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		t := b.String()
		b.Reset()
		if len(t) >= 3 && !isAllDigits(t) {
			set[t] = true
		}
	}
	for _, r := range norm {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return set
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func rowAmount(rec []string, spec ImportSpec) (int, bool) {
	if spec.AmountMode == AmountDebitCredit {
		d, dok := parseSignedAmount(field(rec, spec.DebitCol))
		c, cok := parseSignedAmount(field(rec, spec.CreditCol))
		if !dok && !cok {
			return 0, false
		}
		return absInt(c) - absInt(d), true // credit in, debit out
	}
	return parseSignedAmount(field(rec, spec.AmountCol))
}

// parseSignedAmount parses a money cell into signed minor units, handling
// accounting-style parentheses negatives in addition to a leading minus.
func parseSignedAmount(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if !strings.ContainsAny(s, "0123456789") {
		return 0, false
	}
	paren := strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")")
	v := ParseAmount(s) // separator/decimal/minus aware
	if paren && v > 0 {
		v = -v
	}
	return v, true
}

func parseDateWith(s, layout string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	toSerial := func(t time.Time) int {
		return TimeToSerial(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC))
	}
	if layout != "" {
		if t, err := time.Parse(layout, s); err == nil {
			return toSerial(t), true
		}
		return 0, false
	}
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return toSerial(t), true
		}
	}
	return 0, false
}

// importDupKeys keys existing transactions by (date, payee, |amount to account|)
// so a re-imported file doesn't silently double-add.
func (s *Store) importDupKeys(accountID string) map[string]bool {
	keys := make(map[string]bool)
	for _, t := range s.txns {
		acctAmt, found := 0, false
		for _, p := range t.Posts {
			if p.AccountID == accountID {
				acctAmt += p.Amount
				found = true
			}
		}
		if found {
			keys[dupKey(t.Date, t.Payee, acctAmt)] = true
		}
	}
	return keys
}

func dupKey(date int, payee string, amt int) string {
	return fmt.Sprintf("%d|%s|%d", date, strings.ToLower(strings.TrimSpace(payee)), absInt(amt))
}

// EnsureImportCategories returns the IDs of the catch-all income/expense
// categories used as the default contra side, creating them if missing.
func (s *Store) EnsureImportCategories() (incomeID, expenseID string) {
	incomeID, expenseID = "uncat_income", "uncat_expense"
	if s.AccountByID(incomeID) == nil {
		s.AddAccount(Account{ID: incomeID, Name: "Uncategorized Income", Type: Income, Notes: "Imported income, not yet categorized"})
	}
	if s.AccountByID(expenseID) == nil {
		s.AddAccount(Account{ID: expenseID, Name: "Uncategorized", Type: Expense, Notes: "Imported spending, not yet categorized"})
	}
	return incomeID, expenseID
}

func field(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return rec[i]
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
