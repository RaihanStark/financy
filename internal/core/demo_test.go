package core

import (
	"path/filepath"
	"testing"
)

func TestSeedDemo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.financy")
	s, err := NewDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SeedDemo(s, "$")

	if s.Currency() != "$" {
		t.Fatalf("currency = %q, want $", s.Currency())
	}
	if got := len(s.MoneyAccounts()); got != 4 {
		t.Fatalf("money accounts = %d, want 4", got)
	}
	// Amounts are minor units (cents): net worth $17,143.06, CC owed $550.00.
	if got := s.NetWorth(); got != 1714306 {
		t.Fatalf("net worth = %d, want 1714306", got)
	}
	if got := s.TotalLiabilities(); got != 55000 {
		t.Fatalf("liabilities = %d, want 55000", got)
	}
	// SeedDemo is idempotent (no-op when money accounts exist).
	before := len(s.Transactions())
	SeedDemo(s, "$")
	if len(s.Transactions()) != before {
		t.Fatal("SeedDemo duplicated data")
	}
}
