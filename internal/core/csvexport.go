package core

import (
	"bytes"
	"encoding/csv"
	"sort"
)

// ExportCSV writes all transactions to a flat, spreadsheet-friendly CSV (and one
// that the importer can read back). Each transaction is one row: the money
// account and its signed amount, plus the contra account as the category. Rows
// are chronological. Amounts are plain decimals (no symbol/grouping).
func (s *Store) ExportCSV() ([]byte, error) {
	txns := s.Transactions()
	sort.SliceStable(txns, func(i, j int) bool {
		if txns[i].Date != txns[j].Date {
			return txns[i].Date < txns[j].Date
		}
		return txns[i].ID < txns[j].ID
	})

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Date", "Payee", "Category", "Account", "Amount", "Memo"})
	for _, t := range txns {
		account, category, amount := s.exportRow(t)
		_ = w.Write([]string{
			FmtSerialDate(t.Date), t.Payee, category, account, AmountToInput(amount), t.Memo,
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// exportRow flattens a transaction's postings into (money account, contra
// category, signed amount on the money account). Handles expense/income (1 money
// posting) and transfers (2 money postings → from is the account, to is the
// category).
func (s *Store) exportRow(t Transaction) (account, category string, amount int) {
	name := func(id string) string {
		if a := s.AccountByID(id); a != nil {
			return a.Name
		}
		return id
	}
	var money, other []Posting
	for _, p := range t.Posts {
		if a := s.AccountByID(p.AccountID); a != nil && (a.Type == Asset || a.Type == Liability) {
			money = append(money, p)
		} else {
			other = append(other, p)
		}
	}

	switch {
	case len(money) == 1:
		account, amount = name(money[0].AccountID), money[0].Amount
		if len(other) > 0 {
			category = name(other[0].AccountID)
		}
	case len(money) >= 2: // transfer: from (negative) → to (positive)
		from, to := money[0], money[1]
		if from.Amount > 0 {
			from, to = to, from
		}
		account, category, amount = name(from.AccountID), name(to.AccountID), from.Amount
	default:
		if len(t.Posts) > 0 {
			account, amount = name(t.Posts[0].AccountID), t.Posts[0].Amount
			if len(t.Posts) > 1 {
				category = name(t.Posts[1].AccountID)
			}
		}
	}
	return account, category, amount
}
