package mobileui

import (
	"image/color"
	"sort"
	"unicode"

	"github.com/raihanstark/financy/internal/core"
)

// txnDisplay is a transaction reduced to what a mobile row shows: a title, a
// subtitle (category / accounts + date), and a signed amount. The derivation
// mirrors the desktop's, but it's domain/UX logic, not desktop UI.
type txnDisplay struct {
	kind  string // Expense, Income, Transfer, Opening, Split
	title string
	sub   string
	money int // magnitude, minor units
}

func deriveTxn(s *core.Store, t core.Transaction) txnDisplay {
	d := txnDisplay{kind: "Split", title: t.Payee}
	if len(t.Posts) == 2 {
		a0 := s.AccountByID(t.Posts[0].AccountID)
		a1 := s.AccountByID(t.Posts[1].AccountID)
		if a0 != nil && a1 != nil {
			pick := func(tp core.AcctType) (*core.Account, int, *core.Account, bool) {
				if a0.Type == tp {
					return a0, t.Posts[0].Amount, a1, true
				}
				if a1.Type == tp {
					return a1, t.Posts[1].Amount, a0, true
				}
				return nil, 0, nil, false
			}
			if cat, amt, _, ok := pick(core.Expense); ok {
				d.kind, d.sub, d.money = "Expense", cat.Name, absInt(amt)
				if d.title == "" {
					d.title = cat.Name
				}
				return d
			}
			if cat, amt, _, ok := pick(core.Income); ok {
				d.kind, d.sub, d.money = "Income", cat.Name, absInt(amt)
				if d.title == "" {
					d.title = cat.Name
				}
				return d
			}
			if _, amt, other, ok := pick(core.Equity); ok {
				d.kind, d.sub, d.money = "Opening", other.Name, absInt(amt)
				if d.title == "" {
					d.title = "Opening balance"
				}
				return d
			}
			from, to := a0, a1
			if t.Posts[0].Amount >= 0 {
				from, to = a1, a0
			}
			d.kind, d.sub, d.money = "Transfer", from.Name+" → "+to.Name, absInt(t.Posts[0].Amount)
			if d.title == "" {
				d.title = "Transfer"
			}
			return d
		}
	}
	for _, p := range t.Posts {
		if absInt(p.Amount) > d.money {
			d.money = absInt(p.Amount)
		}
	}
	if d.title == "" {
		d.title = "Split"
	}
	return d
}

// txnEdit is a 2-posting transaction reduced to the add/edit form's fields.
type txnEdit struct {
	kind   string // Expense, Income, Transfer
	money  string // money account name (the "From" account for a transfer)
	other  string // category, or the "To" account for a transfer
	amount int    // positive magnitude, minor units
	ok     bool   // false when the txn can't be represented by the simple form
}

// deriveEdit maps a transaction back to editable form values. Opening balances
// (Equity) and splits (≠2 postings) return ok=false.
func deriveEdit(s *core.Store, t core.Transaction) txnEdit {
	if len(t.Posts) != 2 {
		return txnEdit{}
	}
	a0 := s.AccountByID(t.Posts[0].AccountID)
	a1 := s.AccountByID(t.Posts[1].AccountID)
	if a0 == nil || a1 == nil {
		return txnEdit{}
	}
	pick := func(tp core.AcctType) (*core.Account, int, *core.Account, bool) {
		if a0.Type == tp {
			return a0, t.Posts[0].Amount, a1, true
		}
		if a1.Type == tp {
			return a1, t.Posts[1].Amount, a0, true
		}
		return nil, 0, nil, false
	}
	if cat, amt, other, ok := pick(core.Expense); ok {
		return txnEdit{kind: "Expense", money: other.Name, other: cat.Name, amount: absInt(amt), ok: true}
	}
	if cat, amt, other, ok := pick(core.Income); ok {
		return txnEdit{kind: "Income", money: other.Name, other: cat.Name, amount: absInt(amt), ok: true}
	}
	isMoney := func(a *core.Account) bool { return a.Type == core.Asset || a.Type == core.Liability }
	if isMoney(a0) && isMoney(a1) {
		from, to := a0, a1
		if t.Posts[0].Amount >= 0 {
			from, to = a1, a0
		}
		return txnEdit{kind: "Transfer", money: from.Name, other: to.Name, amount: absInt(t.Posts[0].Amount), ok: true}
	}
	return txnEdit{}
}

// sortedTxns returns all transactions newest-first (stable within a date).
func sortedTxns(s *core.Store) []core.Transaction {
	txns := s.Transactions()
	out := make([]core.Transaction, len(txns))
	copy(out, txns)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date > out[j].Date
		}
		return out[i].ID > out[j].ID
	})
	return out
}

func amountText(d txnDisplay) string {
	switch d.kind {
	case "Income":
		return "+" + core.FmtMoney(d.money)
	case "Expense":
		return "−" + core.FmtMoney(d.money)
	default:
		return core.FmtMoney(d.money)
	}
}

func amountColor(d txnDisplay) color.Color {
	switch d.kind {
	case "Income":
		return colPos
	case "Expense":
		return colNeg
	default:
		return colInk
	}
}

// accentFor is the per-kind accent used for a transaction's avatar.
func accentFor(kind string) color.NRGBA {
	switch kind {
	case "Income":
		return colPos
	case "Expense":
		return colNeg
	case "Transfer":
		return colPrimary
	default:
		return colInkDim
	}
}

func initialOf(s string) string {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return string(unicode.ToUpper(r))
		}
	}
	return "•"
}

// shortDate renders a serial date compactly for list rows, e.g. "Jul 1".
func shortDate(serial int) string {
	return core.SerialToTime(serial).Format("Jan 2")
}

// ellipsize trims s to at most max runes, adding an ellipsis when cut. It's a
// safety net against pathologically long payees/categories overflowing a row.
func ellipsize(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max < 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
