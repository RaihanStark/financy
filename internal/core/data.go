package core

import "time"

// Double-entry data model.
//
// Every Transaction is a set of Postings whose signed amounts sum to zero.
// Account balances are DERIVED by summing postings — the books can never drift
// out of balance. Sign convention (as in Ledger/beancount):
//
//	Asset, Expense   → debit-normal  (increase is +)
//	Liability, Income, Equity → credit-normal (increase is −)
//
// So an asset carries a positive balance, a liability carries a negative
// balance (we display its absolute value as "outstanding"), an income account
// carries a negative balance (period income = −balance) and an expense account
// carries a positive balance.

type AcctType string

const (
	Asset     AcctType = "Asset"
	Liability AcctType = "Liability"
	Income    AcctType = "Income"
	Expense   AcctType = "Expense"
	Equity    AcctType = "Equity"
)

// Account is any node in the chart of accounts.
type Account struct {
	ID          string
	Name        string
	Type        AcctType
	Institution string // for Asset/Liability
	Notes       string
}

// Posting is one leg of a transaction: a signed change to an account balance.
type Posting struct {
	AccountID string
	Amount    int // signed; positive = debit, negative = credit
}

// Transaction is a balanced set of postings.
type Transaction struct {
	ID    string
	Date  int // Excel serial
	Payee string
	Memo  string
	Posts []Posting
}

func (t Transaction) balanced() bool {
	sum := 0
	for _, p := range t.Posts {
		sum += p.Amount
	}
	return sum == 0
}

// ---- Settings ----

var settingsOwner = ""
var settingsCurrency = "Rp"
var settingsYear = time.Now().Year()

// P builds a posting (account + signed amount).
func P(acct string, amt int) Posting { return Posting{AccountID: acct, Amount: amt} }

// seedAccounts starts the app empty of personal data: no asset/liability
// accounts and no transactions. We keep a generic set of income/expense
// categories plus the required Opening Balances equity account so the app is
// usable immediately. Users add their own accounts and transactions.
func seedAccounts() []Account {
	return []Account{
		// Income categories
		{"salary", "Salary", Income, "", "Primary income"},
		{"side", "Side Income", Income, "", "Freelance / extra"},
		{"invest", "Investment Income", Income, "", "Interest / dividends"},
		// Expense categories
		{"housing", "Housing", Expense, "", "Rent / mortgage"},
		{"utilities", "Utilities", Expense, "", "Water / gas / electricity"},
		{"groceries", "Groceries", Expense, "", ""},
		{"dining", "Dining Out", Expense, "", "Restaurants / takeout"},
		{"transport", "Transportation", Expense, "", "Fuel / transit"},
		{"insurance", "Insurance", Expense, "", "Health / vehicle"},
		{"health", "Healthcare", Expense, "", "Medical"},
		{"subs", "Subscriptions", Expense, "", "Streaming / SaaS"},
		{"ent", "Entertainment", Expense, "", "Hobbies / fun"},
		{"shopping", "Shopping", Expense, "", "Goods"},
		{"loanfees", "Loan Fees & Interest", Expense, "", "Financing cost"},
		{"misc", "Miscellaneous", Expense, "", "Catch-all"},
		// Equity (internal, used by opening balances)
		{"opening", "Opening Balances", Equity, "", "Starting balances when an account is created"},
	}
}

// seedTransactions starts with an empty journal.
func seedTransactions() []Transaction { return nil }
