package core

import (
	"database/sql"
	"sort"
	"time"
)

// Store is the runtime source of truth. It holds the chart of accounts and the
// journal of balanced transactions; all balances and totals are derived. When
// backed by a database (db != nil) every mutation writes through to disk.
type Store struct {
	db           *sql.DB
	accounts     []Account
	txns         []Transaction
	recurring    []Recurring
	debts        []Debt
	installments []Installment
	nextID       int
	owner        string
	currency     string
	year         int
	numberFormat string // separator style override; "" = currency default
	listeners    []func()
	onError   func(error)
}

// SetErrorHandler registers a callback invoked when a persistence write fails,
// so the UI can surface it instead of failing silently.
func (s *Store) SetErrorHandler(fn func(error)) { s.onError = fn }

func (s *Store) reportError(err error) {
	if err != nil && s.onError != nil {
		s.onError(err)
	}
}

// TodaySerial is today's date (local), used as the default for new entries.
var TodaySerial = todaySerial()

func todaySerial() int {
	n := time.Now()
	return TimeToSerial(time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC))
}

// NewStore creates an in-memory store seeded with default categories — used for
// tests and ephemeral sessions. It is not persisted (db is nil).
func NewStore() *Store {
	s := &Store{
		accounts: seedAccounts(),
		txns:     seedTransactions(),
		owner:    settingsOwner,
		currency: settingsCurrency,
		year:     settingsYear,
	}
	s.nextID = len(s.txns) + 1
	SetCurrencySymbol(s.currency)
	return s
}

// OpenStore opens an existing document file and loads it into memory.
func OpenStore(path string) (*Store, error) {
	db, err := openDB(path)
	if err != nil {
		return nil, err
	}
	return loadStore(db)
}

// NewDocument creates (or opens) a document file, seeding a fresh one with the
// default chart of accounts. Existing files are simply loaded.
func NewDocument(path string) (*Store, error) {
	db, err := openDB(path)
	if err != nil {
		return nil, err
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&n); err != nil {
		db.Close()
		return nil, err
	}
	if n == 0 {
		seed := &Store{db: db, currency: "Rp", year: settingsYear}
		for _, a := range seedAccounts() {
			if err := seed.dbUpsertAccount(a); err != nil {
				db.Close()
				return nil, err
			}
		}
		if err := seed.dbSetSettings(); err != nil {
			db.Close()
			return nil, err
		}
	}
	return loadStore(db)
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveCopy writes a clean snapshot of the document to path (which must not yet
// exist). Used for "Save a Copy".
func (s *Store) SaveCopy(path string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`VACUUM INTO ?`, path)
	return err
}

// FileVersion / LatestVersion let callers detect files needing migration.
func FileVersion(path string) (int, error) { return dbVersion(path) }
func LatestVersion() int                   { return schemaVersion() }

// ---- change notification ----

func (s *Store) Subscribe(fn func()) { s.listeners = append(s.listeners, fn) }
func (s *Store) notify() {
	for _, fn := range s.listeners {
		fn()
	}
}

// ---- account access ----

func (s *Store) AccountByID(id string) *Account {
	for i := range s.accounts {
		if s.accounts[i].ID == id {
			return &s.accounts[i]
		}
	}
	return nil
}

func (s *Store) AccountByName(name string) *Account {
	for i := range s.accounts {
		if s.accounts[i].Name == name {
			return &s.accounts[i]
		}
	}
	return nil
}

func NamesOf(accts []Account) []string {
	out := make([]string, len(accts))
	for i, a := range accts {
		out[i] = a.Name
	}
	return out
}

func (s *Store) accountsOfType(t AcctType) []Account {
	var out []Account
	for _, a := range s.accounts {
		if a.Type == t {
			out = append(out, a)
		}
	}
	return out
}

func (s *Store) AssetAccounts() []Account     { return s.accountsOfType(Asset) }
func (s *Store) LiabilityAccounts() []Account { return s.accountsOfType(Liability) }
func (s *Store) IncomeAccounts() []Account    { return s.accountsOfType(Income) }
func (s *Store) ExpenseAccounts() []Account   { return s.accountsOfType(Expense) }

// moneyAccounts returns asset + liability accounts (real-money accounts you can
// pay from / into).
func (s *Store) MoneyAccounts() []Account {
	var out []Account
	for _, a := range s.accounts {
		if a.Type == Asset || a.Type == Liability {
			out = append(out, a)
		}
	}
	return out
}

// ---- balances ----

// Balance returns the raw signed balance (sum of postings) for an account.
func (s *Store) Balance(id string) int {
	sum := 0
	for _, t := range s.txns {
		for _, p := range t.Posts {
			if p.AccountID == id {
				sum += p.Amount
			}
		}
	}
	return sum
}

// DisplayBalance returns the natural, human-facing balance: liabilities and
// income accounts are shown as positive magnitudes.
func (s *Store) DisplayBalance(a Account) int {
	b := s.Balance(a.ID)
	switch a.Type {
	case Liability, Income, Equity:
		return -b
	default:
		return b
	}
}

func (s *Store) TotalAssets() int {
	sum := 0
	for _, a := range s.AssetAccounts() {
		sum += s.Balance(a.ID)
	}
	return sum
}

func (s *Store) TotalLiabilities() int {
	sum := 0
	for _, a := range s.LiabilityAccounts() {
		sum += -s.Balance(a.ID) // stored negative
	}
	return sum
}

func (s *Store) NetWorth() int { return s.TotalAssets() - s.TotalLiabilities() }

// ---- period flows ----

func inYear(date, year int) bool { return SerialToTime(date).Year() == year }

// IncomeTotal sums real income for the store's year.
func (s *Store) IncomeTotal() int {
	sum := 0
	for _, a := range s.IncomeAccounts() {
		for _, t := range s.txns {
			if !inYear(t.Date, s.year) {
				continue
			}
			for _, p := range t.Posts {
				if p.AccountID == a.ID {
					sum += -p.Amount
				}
			}
		}
	}
	return sum
}

// ExpenseTotal sums real expenses for the store's year.
func (s *Store) ExpenseTotal() int {
	sum := 0
	for _, a := range s.ExpenseAccounts() {
		sum += s.expenseAccountTotal(a.ID)
	}
	return sum
}

func (s *Store) expenseAccountTotal(id string) int {
	sum := 0
	for _, t := range s.txns {
		if !inYear(t.Date, s.year) {
			continue
		}
		for _, p := range t.Posts {
			if p.AccountID == id {
				sum += p.Amount
			}
		}
	}
	return sum
}

func (s *Store) incomeAccountTotal(id string) int {
	sum := 0
	for _, t := range s.txns {
		if !inYear(t.Date, s.year) {
			continue
		}
		for _, p := range t.Posts {
			if p.AccountID == id {
				sum += -p.Amount
			}
		}
	}
	return sum
}

// CategoryTotal returns the period total for an income or expense account.
func (s *Store) CategoryTotal(a Account) int {
	if a.Type == Income {
		return s.incomeAccountTotal(a.ID)
	}
	return s.expenseAccountTotal(a.ID)
}

// ---- transactions ----

func (s *Store) Transactions() []Transaction {
	out := append([]Transaction(nil), s.txns...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date > out[j].Date
		}
		return out[i].ID > out[j].ID
	})
	return out
}

func (s *Store) TxnByID(id string) *Transaction {
	for i := range s.txns {
		if s.txns[i].ID == id {
			return &s.txns[i]
		}
	}
	return nil
}

// RegisterRow is one line of an account register, with running balance.
type RegisterRow struct {
	Txn     Transaction
	Amount  int // signed change to THIS account
	Balance int // running balance after this row (display sense)
}

// Register returns an account's postings oldest-first with a running balance.
func (s *Store) Register(id string) []RegisterRow {
	a := s.AccountByID(id)
	if a == nil {
		return nil
	}
	var rows []RegisterRow
	src := append([]Transaction(nil), s.txns...)
	sort.SliceStable(src, func(i, j int) bool {
		if src[i].Date != src[j].Date {
			return src[i].Date < src[j].Date
		}
		return src[i].ID < src[j].ID
	})
	running := 0
	for _, t := range src {
		for _, p := range t.Posts {
			if p.AccountID != id {
				continue
			}
			running += p.Amount
			disp := running
			if a.Type == Liability || a.Type == Income || a.Type == Equity {
				disp = -running
			}
			rows = append(rows, RegisterRow{Txn: t, Amount: p.Amount, Balance: disp})
		}
	}
	// newest first for display
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows
}

func (s *Store) genID() string {
	id := "t" + Itoa(s.nextID)
	s.nextID++
	return id
}

// AddTransaction appends a balanced transaction (no-op if unbalanced or if the
// write-through to disk fails).
func (s *Store) AddTransaction(t Transaction) bool {
	if !t.balanced() || len(t.Posts) < 2 {
		return false
	}
	if t.ID == "" {
		t.ID = s.genID()
	}
	if err := s.dbUpsertTxn(t); err != nil {
			s.reportError(err)
		return false
	}
	s.txns = append(s.txns, t)
	s.notify()
	return true
}

// AddTransactions commits many balanced transactions at once — one DB
// transaction and a single change notification (used by CSV import). Unbalanced
// entries are skipped. Returns the number added.
func (s *Store) AddTransactions(txns []Transaction) (int, error) {
	valid := make([]Transaction, 0, len(txns))
	for _, t := range txns {
		if !t.balanced() || len(t.Posts) < 2 {
			continue
		}
		if t.ID == "" {
			t.ID = s.genID()
		}
		valid = append(valid, t)
	}
	if len(valid) == 0 {
		return 0, nil
	}
	if err := s.dbInsertTxnBatch(valid); err != nil {
		s.reportError(err)
		return 0, err
	}
	s.txns = append(s.txns, valid...)
	s.notify()
	return len(valid), nil
}

func (s *Store) UpdateTransaction(id string, t Transaction) bool {
	if !t.balanced() {
		return false
	}
	for i := range s.txns {
		if s.txns[i].ID == id {
			t.ID = id
			if err := s.dbUpsertTxn(t); err != nil {
					s.reportError(err)
				return false
			}
			s.txns[i] = t
			s.notify()
			return true
		}
	}
	return false
}

func (s *Store) DeleteTransaction(id string) {
	for i := range s.txns {
		if s.txns[i].ID == id {
			if err := s.dbDeleteTxn(id); err != nil {
					s.reportError(err)
				return
			}
			s.txns = append(s.txns[:i], s.txns[i+1:]...)
			s.notify()
			return
		}
	}
}

func (s *Store) TxnCountForAccount(id string) int {
	n := 0
	for _, t := range s.txns {
		for _, p := range t.Posts {
			if p.AccountID == id {
				n++
			}
		}
	}
	return n
}

// ---- account mutations ----

func (s *Store) AddAccount(a Account) {
	if a.ID == "" {
		a.ID = Slugify(a.Name)
	}
	if err := s.dbUpsertAccount(a); err != nil {
			s.reportError(err)
		return
	}
	s.accounts = append(s.accounts, a)
	s.notify()
}

func (s *Store) UpdateAccount(id string, a Account) {
	p := s.AccountByID(id)
	if p == nil {
		return
	}
	a.ID = id
	if err := s.dbUpsertAccount(a); err != nil {
			s.reportError(err)
		return
	}
	*p = a
	s.notify()
}

// DeleteAccount removes an account only if it has no postings.
func (s *Store) DeleteAccount(id string) bool {
	if s.TxnCountForAccount(id) > 0 {
		return false
	}
	for i := range s.accounts {
		if s.accounts[i].ID == id {
			if err := s.dbDeleteAccount(id); err != nil {
					s.reportError(err)
				return false
			}
			s.accounts = append(s.accounts[:i], s.accounts[i+1:]...)
			s.notify()
			return true
		}
	}
	return false
}

func (s *Store) SetSettings(owner, currency string, year int) {
	s.owner, s.currency, s.year = owner, currency, year
	s.reportError(s.dbSetSettings())
	SetCurrencySymbol(currency)        // resets separators to the currency default…
	ApplyNumberFormat(s.numberFormat)  // …then re-apply any chosen number format.
	s.notify()
}

// SetNumberFormat overrides the grouping/decimal separator style (see
// NumberFormatStyles); "" reverts to the active currency's default.
func (s *Store) SetNumberFormat(style string) {
	s.numberFormat = style
	s.reportError(s.dbSetSettings())
	ApplyNumberFormat(style)
	s.notify()
}

func (s *Store) NumberFormat() string { return s.numberFormat }

func (s *Store) Owner() string    { return s.owner }
func (s *Store) Currency() string { return s.currency }
func (s *Store) Year() int        { return s.year }

func Slugify(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32)
		case r == ' ' || r == '-' || r == '/':
			out = append(out, '_')
		}
	}
	return string(out)
}
