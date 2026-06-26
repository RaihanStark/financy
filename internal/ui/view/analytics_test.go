package view

import (
	"testing"
)

// The analytics dashboard renders all five KPI cards and the three chart panels.
func TestScreenAnalyticsRenders(t *testing.T) {
	_, w := newViewTest(t)
	screen := mount(w, ScreenAnalytics())

	assertText(t, screen,
		"Analytics",
		"NET WORTH", "INCOME", "EXPENSES", "NET SAVED", "SAVINGS RATE",
		"Income vs Expenses", "Net Worth Over Time", "Spending by Category",
	)
}

// The KPI net-worth figure matches the store's analytics summary for the period.
func TestAnalyticsKPIMatchesSummary(t *testing.T) {
	s, w := newViewTest(t)
	start, end := analyticsRange(analyticsPeriod)
	sum := s.AnalyticsSummaryFor(start, end)

	screen := mount(w, ScreenAnalytics())
	assertText(t, screen, fmtMoney(sum.NetWorth))
}

// With no data the chart panels collapse to their empty states.
func TestScreenAnalyticsEmptyState(t *testing.T) {
	_, w := emptyViewTest(t)
	screen := mount(w, ScreenAnalytics())
	// KPIs still render; the charts show empty messaging.
	assertText(t, screen, "Spending by Category", "No spending in this period")
}

// analyticsRange produces an inclusive window ending today for each period.
func TestAnalyticsRange(t *testing.T) {
	_, _ = newViewTest(t)
	for _, p := range []string{periodThisMonth, periodLast3, periodLast6, periodLast12, periodYTD} {
		start, end := analyticsRange(p)
		if start > end {
			t.Errorf("period %q: start %d after end %d", p, start, end)
		}
		if end != todaySerial {
			t.Errorf("period %q: end %d, want today %d", p, end, todaySerial)
		}
	}
}

// Changing the period select re-derives the subtitle range.
func TestAnalyticsPeriodSubtitle(t *testing.T) {
	_, _ = newViewTest(t)
	start, end := analyticsRange(periodThisMonth)
	sub := periodSubtitle(start, end)
	if sub == "" {
		t.Fatal("empty period subtitle")
	}
}
