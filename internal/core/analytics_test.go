package core

import (
	"testing"
)

// freshDemoStore returns an in-memory store seeded with the 6-month demo data.
func freshDemoStore() *Store {
	s := &Store{accounts: seedAccounts(), owner: settingsOwner, currency: settingsCurrency, year: settingsYear}
	s.nextID = 1
	SeedDemo(s, "$")
	return s
}

func TestMonthlyFlowsSpanAndTotals(t *testing.T) {
	s := freshDemoStore()
	start := TodaySerial - 182
	end := TodaySerial

	flows := s.MonthlyFlows(start, end)
	if len(flows) < 6 || len(flows) > 8 {
		t.Fatalf("got %d month buckets, want ~6-7 for a 182-day span", len(flows))
	}

	// Monthly sums must equal the period summary (no double counting / gaps).
	sumInc, sumExp := 0, 0
	for _, f := range flows {
		if f.Income < 0 || f.Expense < 0 {
			t.Fatalf("month %s has negative totals: inc=%d exp=%d", f.Label, f.Income, f.Expense)
		}
		sumInc += f.Income
		sumExp += f.Expense
	}
	got := s.AnalyticsSummaryFor(start, end)
	if got.Income != sumInc || got.Expense != sumExp {
		t.Fatalf("summary {inc=%d exp=%d} != monthly sums {inc=%d exp=%d}", got.Income, got.Expense, sumInc, sumExp)
	}
	if got.Income == 0 || got.Expense == 0 {
		t.Fatal("expected non-zero income and expense over 6 months of demo data")
	}
}

func TestNetWorthSeriesEndsAtCurrentNetWorth(t *testing.T) {
	s := freshDemoStore()
	series := s.NetWorthSeries(TodaySerial-182, TodaySerial)
	if len(series) == 0 {
		t.Fatal("empty net-worth series")
	}
	last := series[len(series)-1].NetWorth
	if last != s.NetWorth() {
		t.Fatalf("last series point = %d, want current net worth %d", last, s.NetWorth())
	}
	// Net worth should be non-decreasing-ish here, but at minimum the demo grows
	// it: the final point must exceed the first.
	if last <= series[0].NetWorth {
		t.Fatalf("net worth did not grow: first=%d last=%d", series[0].NetWorth, last)
	}
}

func TestCategoryBreakdownRankedAndShares(t *testing.T) {
	s := freshDemoStore()
	cats := s.CategoryBreakdown(TodaySerial-182, TodaySerial, Expense)
	if len(cats) == 0 {
		t.Fatal("no expense categories")
	}
	// Sorted descending by total, and percentages sum to ~100.
	pct := 0.0
	for i, c := range cats {
		if c.Total <= 0 {
			t.Fatalf("category %s has non-positive total %d", c.Account.Name, c.Total)
		}
		if i > 0 && c.Total > cats[i-1].Total {
			t.Fatal("categories not sorted descending by total")
		}
		pct += c.Pct
	}
	if pct < 99.0 || pct > 101.0 {
		t.Fatalf("category percentages sum to %.2f, want ~100", pct)
	}
}

func TestAnalyticsSummarySane(t *testing.T) {
	s := freshDemoStore()
	got := s.AnalyticsSummaryFor(TodaySerial-182, TodaySerial)
	if got.NetWorth != s.NetWorth() {
		t.Fatalf("summary net worth %d != store net worth %d", got.NetWorth, s.NetWorth())
	}
	if got.Net != got.Income-got.Expense {
		t.Fatalf("net %d != income-expense %d", got.Net, got.Income-got.Expense)
	}
	if got.SavingsRate <= 0 || got.SavingsRate >= 1 {
		t.Fatalf("savings rate %.2f out of expected (0,1) for the demo", got.SavingsRate)
	}
	if got.NetWorthDelta <= 0 {
		t.Fatalf("net worth delta %d, want positive growth over the period", got.NetWorthDelta)
	}
}
