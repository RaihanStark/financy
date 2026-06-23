package core

import (
	"sort"
	"time"
)

// Analytics: read-only, derived time-series over the journal. Nothing here is
// persisted — exactly like balances, every figure is recomputed from postings,
// so the numbers can never drift from the double-entry truth. All amounts are
// integer minor units; ranges are inclusive Excel-serial dates [start, end].

// MonthFlow is one calendar month's income and expense totals.
type MonthFlow struct {
	MonthStart int    // serial of the first day of the month
	Label      string // short label, e.g. "Jan"
	Income     int
	Expense    int
}

// Net is income minus expense for the month (positive = saved).
func (f MonthFlow) Net() int { return f.Income - f.Expense }

// NetWorthPoint is net worth as of a month-end.
type NetWorthPoint struct {
	MonthStart int
	Label      string
	NetWorth   int
}

// CategorySlice is one category's total over a period, with its share of the
// period's total for that side (income or expense).
type CategorySlice struct {
	Account Account
	Total   int
	Pct     float64
}

// AnalyticsSummary is the headline set of KPIs for a period.
type AnalyticsSummary struct {
	NetWorth        int
	NetWorthDelta   int     // change in net worth across the period
	Income          int
	Expense         int
	Net             int     // income - expense
	SavingsRate     float64 // net / income, 0 when there's no income
	AvgMonthlySpend int
	Months          int
}

// monthStarts lists the first day of each calendar month spanning [start, end].
func monthStarts(start, end int) []time.Time {
	s, e := SerialToTime(start), SerialToTime(end)
	cur := time.Date(s.Year(), s.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(e.Year(), e.Month(), 1, 0, 0, 0, 0, time.UTC)
	var out []time.Time
	for !cur.After(last) {
		out = append(out, cur)
		cur = cur.AddDate(0, 1, 0)
	}
	return out
}

func idSet(accts []Account) map[string]bool {
	m := make(map[string]bool, len(accts))
	for _, a := range accts {
		m[a.ID] = true
	}
	return m
}

// MonthlyFlows returns income and expense per calendar month across [start, end].
// Empty months are included as zeroes so a chart shows a continuous axis.
func (s *Store) MonthlyFlows(start, end int) []MonthFlow {
	income := idSet(s.IncomeAccounts())
	expense := idSet(s.ExpenseAccounts())
	months := monthStarts(start, end)

	idx := make(map[string]int, len(months))
	out := make([]MonthFlow, len(months))
	for i, m := range months {
		out[i] = MonthFlow{MonthStart: TimeToSerial(m), Label: m.Format("Jan")}
		idx[m.Format("2006-01")] = i
	}
	for _, t := range s.txns {
		if t.Date < start || t.Date > end {
			continue
		}
		i, ok := idx[SerialToTime(t.Date).Format("2006-01")]
		if !ok {
			continue
		}
		for _, p := range t.Posts {
			switch {
			case income[p.AccountID]:
				out[i].Income += -p.Amount // income postings are stored negative
			case expense[p.AccountID]:
				out[i].Expense += p.Amount
			}
		}
	}
	return out
}

// netWorthAsOf is net worth (assets + liabilities, liabilities stored negative)
// considering only postings dated on or before asOf.
func (s *Store) netWorthAsOf(asOf int) int {
	money := idSet(s.MoneyAccounts())
	sum := 0
	for _, t := range s.txns {
		if t.Date > asOf {
			continue
		}
		for _, p := range t.Posts {
			if money[p.AccountID] {
				sum += p.Amount
			}
		}
	}
	return sum
}

// NetWorthSeries returns net worth as of each month-end across [start, end]
// (the final point is taken as of end itself).
func (s *Store) NetWorthSeries(start, end int) []NetWorthPoint {
	months := monthStarts(start, end)
	out := make([]NetWorthPoint, len(months))
	for i, m := range months {
		asOf := min(TimeToSerial(m.AddDate(0, 1, -1)), end) // month-end, capped at end
		out[i] = NetWorthPoint{MonthStart: TimeToSerial(m), Label: m.Format("Jan"), NetWorth: s.netWorthAsOf(asOf)}
	}
	return out
}

// CategoryBreakdown ranks income or expense categories by their total over the
// period, with each category's percentage share. Zero categories are omitted.
func (s *Store) CategoryBreakdown(start, end int, typ AcctType) []CategorySlice {
	accts := s.ExpenseAccounts()
	if typ == Income {
		accts = s.IncomeAccounts()
	}
	set := idSet(accts)
	totals := make(map[string]int, len(accts))
	for _, t := range s.txns {
		if t.Date < start || t.Date > end {
			continue
		}
		for _, p := range t.Posts {
			if !set[p.AccountID] {
				continue
			}
			if typ == Income {
				totals[p.AccountID] += -p.Amount
			} else {
				totals[p.AccountID] += p.Amount
			}
		}
	}
	grand := 0
	for _, v := range totals {
		grand += v
	}
	var out []CategorySlice
	for _, a := range accts {
		v := totals[a.ID]
		if v == 0 {
			continue
		}
		pct := 0.0
		if grand > 0 {
			pct = float64(v) / float64(grand) * 100
		}
		out = append(out, CategorySlice{Account: a, Total: v, Pct: pct})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out
}

// AnalyticsSummaryFor computes the headline KPIs for the period [start, end].
func (s *Store) AnalyticsSummaryFor(start, end int) AnalyticsSummary {
	flows := s.MonthlyFlows(start, end)
	inc, exp := 0, 0
	for _, f := range flows {
		inc += f.Income
		exp += f.Expense
	}
	net := inc - exp
	rate := 0.0
	if inc > 0 {
		rate = float64(net) / float64(inc)
	}
	divisor := len(flows)
	if divisor == 0 {
		divisor = 1
	}
	return AnalyticsSummary{
		NetWorth:        s.netWorthAsOf(end),
		NetWorthDelta:   s.netWorthAsOf(end) - s.netWorthAsOf(start-1),
		Income:          inc,
		Expense:         exp,
		Net:             net,
		SavingsRate:     rate,
		AvgMonthlySpend: exp / divisor,
		Months:          len(flows),
	}
}
