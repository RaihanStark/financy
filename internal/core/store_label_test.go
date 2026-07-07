package core

import (
	"reflect"
	"testing"
)

// GroupedLabels clusters a mixed money list into Assets then Liabilities
// sections, each led by a non-selectable header, and sorts by institution then
// name within a section.
func TestGroupedLabelsMultiSection(t *testing.T) {
	accts := []Account{
		{Name: "Card", Type: Liability, Institution: "Amex"},
		{Name: "Savings", Type: Asset, Institution: "Chase"},
		{Name: "Checking", Type: Asset, Institution: "Chase"},
		{Name: "Cash", Type: Asset}, // no institution sorts before "Chase"
	}
	got := GroupedLabels(accts)
	want := []string{
		"—— Assets ——",
		"Cash (Asset)",
		"Checking (Asset · Chase)",
		"Savings (Asset · Chase)",
		"—— Liabilities ——",
		"Card (Liability · Amex)",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GroupedLabels =\n%#v\nwant\n%#v", got, want)
	}
}

// A single-section list is sorted but carries no header row.
func TestGroupedLabelsSingleSection(t *testing.T) {
	accts := []Account{
		{Name: "Rent", Type: Expense},
		{Name: "Groceries", Type: Expense},
	}
	got := GroupedLabels(accts)
	want := []string{"Groceries (Expense)", "Rent (Expense)"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GroupedLabels single section = %#v, want %#v", got, want)
	}
	for _, o := range got {
		if IsGroupHeader(o) {
			t.Errorf("single-section list should have no header, got %q", o)
		}
	}
}

// IsGroupHeader distinguishes header rows from real account labels, and
// FirstSelectable skips the leading header.
func TestGroupHeaderHelpers(t *testing.T) {
	opts := GroupedLabels([]Account{
		{Name: "Checking", Type: Asset},
		{Name: "Card", Type: Liability},
	})
	if !IsGroupHeader(opts[0]) {
		t.Errorf("opts[0] %q should be a header", opts[0])
	}
	if IsGroupHeader("Checking (Asset)") {
		t.Error("account label misclassified as header")
	}
	first := FirstSelectable(opts)
	if IsGroupHeader(first) || first == "" {
		t.Errorf("FirstSelectable returned %q, want first real option", first)
	}
	if FirstSelectable([]string{"—— Assets ——"}) != "" {
		t.Error("FirstSelectable over headers-only should be empty")
	}
	// A header resolves to no account.
	s := NewStore()
	if a := s.AccountByLabel("—— Assets ——"); a != nil {
		t.Errorf("AccountByLabel(header) = %v, want nil", a)
	}
}

// AccountLabel adds stable disambiguators (type, and institution when set) so
// account selectors can tell same-named accounts apart.
func TestAccountLabel(t *testing.T) {
	cases := []struct {
		acct Account
		want string
	}{
		{Account{Name: "Checking", Type: Asset, Institution: "Chase"}, "Checking (Asset · Chase)"},
		{Account{Name: "Groceries", Type: Expense}, "Groceries (Expense)"},
		{Account{Name: "Salary", Type: Income}, "Salary (Income)"},
		{Account{Name: "Card", Type: Liability, Institution: "Amex"}, "Card (Liability · Amex)"},
	}
	for _, c := range cases {
		if got := AccountLabel(c.acct); got != c.want {
			t.Errorf("AccountLabel(%q) = %q, want %q", c.acct.Name, got, c.want)
		}
	}
}

// LabelsOf mirrors NamesOf but yields the disambiguated labels.
func TestLabelsOf(t *testing.T) {
	accts := []Account{
		{Name: "Cash", Type: Asset},
		{Name: "Cash", Type: Asset, Institution: "Wallet"},
	}
	got := LabelsOf(accts)
	want := []string{"Cash (Asset)", "Cash (Asset · Wallet)"}
	if len(got) != len(want) {
		t.Fatalf("LabelsOf len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("LabelsOf[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// AccountByLabel resolves a selector label back to the right account even when
// two accounts share a name, and still accepts a bare name for compatibility.
func TestAccountByLabel(t *testing.T) {
	s := NewStore()
	s.AddAccount(Account{ID: "cash-a", Name: "Cash", Type: Asset, Institution: "Wallet"})
	s.AddAccount(Account{ID: "cash-b", Name: "Cash", Type: Asset, Institution: "Safe"})

	if a := s.AccountByLabel("Cash (Asset · Wallet)"); a == nil || a.ID != "cash-a" {
		t.Errorf("AccountByLabel(Wallet) = %v, want cash-a", a)
	}
	if a := s.AccountByLabel("Cash (Asset · Safe)"); a == nil || a.ID != "cash-b" {
		t.Errorf("AccountByLabel(Safe) = %v, want cash-b", a)
	}
	// Bare-name fallback still resolves (to the first match).
	if a := s.AccountByLabel("Cash"); a == nil || a.ID != "cash-a" {
		t.Errorf("AccountByLabel(bare name) = %v, want first match cash-a", a)
	}
	if a := s.AccountByLabel("Nope (Asset)"); a != nil {
		t.Errorf("AccountByLabel(unknown) = %v, want nil", a)
	}
}
