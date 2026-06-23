package core

// Financial statements — read-only, derived entirely from postings (like the
// analytics screen). No new data, no schema change. The double-entry model means
// every figure here is recomputed and always consistent.

// StmtLine is one labelled amount in a statement (display sense, positive).
type StmtLine struct {
	Name   string
	Amount int
}

// ---- balances as of a date ----

func (s *Store) balanceAsOf(id string, asOf int) int {
	sum := 0
	for _, t := range s.txns {
		if t.Date > asOf {
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

func (s *Store) displayBalanceAsOf(a Account, asOf int) int {
	b := s.balanceAsOf(a.ID, asOf)
	switch a.Type {
	case Liability, Income, Equity:
		return -b
	default:
		return b
	}
}

// ---- income statement (profit & loss) ----

type IncomeStatement struct {
	Start, End    int
	Income        []StmtLine
	TotalIncome   int
	Expenses      []StmtLine
	TotalExpenses int
	NetIncome     int
}

func (s *Store) IncomeStatementFor(start, end int) IncomeStatement {
	is := IncomeStatement{Start: start, End: end}
	for _, c := range s.CategoryBreakdown(start, end, Income) {
		is.Income = append(is.Income, StmtLine{c.Account.Name, c.Total})
		is.TotalIncome += c.Total
	}
	for _, c := range s.CategoryBreakdown(start, end, Expense) {
		is.Expenses = append(is.Expenses, StmtLine{c.Account.Name, c.Total})
		is.TotalExpenses += c.Total
	}
	is.NetIncome = is.TotalIncome - is.TotalExpenses
	return is
}

// ---- balance sheet ----

type BalanceSheet struct {
	AsOf             int
	Assets           []StmtLine
	TotalAssets      int
	Liabilities      []StmtLine
	TotalLiabilities int
	Equity           []StmtLine // equity accounts + a Retained Earnings line
	TotalEquity      int
}

// BalanceSheetAsOf proves the accounting identity Assets = Liabilities + Equity,
// where equity is the equity accounts plus retained earnings (income − expenses
// to date).
func (s *Store) BalanceSheetAsOf(asOf int) BalanceSheet {
	bs := BalanceSheet{AsOf: asOf}
	for _, a := range s.AssetAccounts() {
		bal := s.displayBalanceAsOf(a, asOf)
		bs.Assets = append(bs.Assets, StmtLine{a.Name, bal})
		bs.TotalAssets += bal
	}
	for _, a := range s.LiabilityAccounts() {
		bal := s.displayBalanceAsOf(a, asOf)
		bs.Liabilities = append(bs.Liabilities, StmtLine{a.Name, bal})
		bs.TotalLiabilities += bal
	}
	for _, a := range s.accountsOfType(Equity) {
		if bal := s.displayBalanceAsOf(a, asOf); bal != 0 {
			bs.Equity = append(bs.Equity, StmtLine{a.Name, bal})
			bs.TotalEquity += bal
		}
	}
	retained := 0
	for _, a := range s.IncomeAccounts() {
		retained += s.displayBalanceAsOf(a, asOf)
	}
	for _, a := range s.ExpenseAccounts() {
		retained -= s.displayBalanceAsOf(a, asOf)
	}
	bs.Equity = append(bs.Equity, StmtLine{"Retained Earnings", retained})
	bs.TotalEquity += retained
	return bs
}

// ---- cash flow (simplified) ----

type CashLine struct {
	Name             string
	Opening, Closing int
	Change           int
}

type CashFlow struct {
	Start, End  int
	Accounts    []CashLine // asset (cash) accounts
	OpeningCash int
	ClosingCash int
	NetChange   int
	Income      int // period context
	Expenses    int
}

// CashFlowFor reports the change in cash (asset accounts) over the period, per
// account, plus the period's income/expense totals for context. A full
// operating/investing/financing breakdown would need per-transaction
// classification Financy doesn't store, so this stays a straight cash movement.
func (s *Store) CashFlowFor(start, end int) CashFlow {
	cf := CashFlow{Start: start, End: end}
	for _, a := range s.AssetAccounts() {
		open := s.balanceAsOf(a.ID, start-1)
		clo := s.balanceAsOf(a.ID, end)
		cf.Accounts = append(cf.Accounts, CashLine{a.Name, open, clo, clo - open})
		cf.OpeningCash += open
		cf.ClosingCash += clo
	}
	cf.NetChange = cf.ClosingCash - cf.OpeningCash
	is := s.IncomeStatementFor(start, end)
	cf.Income, cf.Expenses = is.TotalIncome, is.TotalExpenses
	return cf
}
