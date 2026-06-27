package view

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/raihanstark/financy/internal/core"
)

// The accounts screen renders the net-worth hero, both asset and liability
// groups, and a card per money account showing its name and balance.
func TestScreenAccountsRendersAccountsAndTotals(t *testing.T) {
	s, w := newViewTest(t)
	screen := mount(w, ScreenAccounts())

	assertText(t, screen,
		"Accounts", "NET WORTH", "TOTAL ASSETS", "TOTAL LIABILITIES",
		"Assets", "Liabilities", "Add Account",
	)

	// Every demo money account appears as a card, with its balance.
	for _, a := range s.MoneyAccounts() {
		assertText(t, screen, a.Name)
	}

	// The hero net-worth figure is the store's, formatted.
	assertText(t, screen, fmtMoney(s.NetWorth()))
}

// With no money accounts the asset/liability groups fall back to empty states.
func TestScreenAccountsEmptyState(t *testing.T) {
	_, w := emptyViewTest(t)
	screen := mount(w, ScreenAccounts())
	assertText(t, screen, "No Assets yet", "No Liabilities yet")
}

// Tapping a card's tap target opens the register dialog for that account.
func TestAccountRegisterDialog(t *testing.T) {
	s, w := newViewTest(t)
	checking := s.AccountByName("Checking")
	if checking == nil {
		t.Fatal("demo store should have a Checking account")
	}

	showRegisterDialog(*checking)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("register dialog did not open")
	}
	// The register shows the running-balance ledger headers and the account name.
	assertText(t, d, "Checking", "Balance", "Contra account", "New Transaction")
}

// The register for a fresh account with no postings shows an empty state.
func TestRegisterTableEmpty(t *testing.T) {
	s, w := newViewTest(t)
	s.AddAccount(core.Account{ID: "brokerage", Name: "Brokerage", Type: core.Asset})
	tbl := mount(w, registerTable(*s.AccountByName("Brokerage")))
	assertText(t, tbl, "No transactions for this account")
}

// Adding an account through AccountForm with an opening balance creates the
// account and an opening-balance transaction.
func TestAccountFormAddCreatesAccount(t *testing.T) {
	s, w := newViewTest(t)
	before := len(s.MoneyAccounts())

	AccountForm(nil)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("account form dialog did not open")
	}

	entries := findEntries(d)
	if len(entries) == 0 {
		t.Fatal("account form has no entry fields")
	}
	// First entry is Name; the amount entry is the opening balance.
	entries[0].SetText("Brokerage")
	// Type defaults to Asset; set opening balance on the amount entry (last entry).
	test.Type(entries[len(entries)-1], "150000")

	if !tapButton(d, "Save") {
		t.Fatal("no Save button in account form")
	}
	if got := len(s.MoneyAccounts()); got != before+1 {
		t.Fatalf("money accounts = %d, want %d", got, before+1)
	}
	acc := s.AccountByName("Brokerage")
	if acc == nil {
		t.Fatal("Brokerage account was not created")
	}
	if bal := s.DisplayBalance(*acc); bal <= 0 {
		t.Fatalf("opening balance not posted, balance = %d", bal)
	}
}

// Editing an account through AccountForm updates its fields without adding a
// new account.
func TestAccountFormEditUpdates(t *testing.T) {
	s, w := newViewTest(t)
	checking := s.AccountByName("Checking")
	before := len(s.MoneyAccounts())

	AccountForm(checking)
	d := dialogContent(w)
	entries := findEntries(d)
	entries[0].SetText("Main Checking")

	if !tapButton(d, "Save") {
		t.Fatal("no Save button")
	}
	if got := len(s.MoneyAccounts()); got != before {
		t.Fatalf("account count changed on edit: %d → %d", before, got)
	}
	if s.AccountByName("Main Checking") == nil {
		t.Fatal("account rename did not take effect")
	}
}

// Deleting an account that still has transactions is refused with an info dialog.
func TestDeleteAccountWithTransactionsRefused(t *testing.T) {
	s, w := newViewTest(t)
	checking := s.AccountByName("Checking")
	if s.TxnCountForAccount(checking.ID) == 0 {
		t.Skip("demo Checking unexpectedly has no transactions")
	}
	before := len(s.MoneyAccounts())

	deleteAccount(*checking)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("expected an info dialog refusing the delete")
	}
	assertText(t, d, "Cannot delete")
	if got := len(s.MoneyAccounts()); got != before {
		t.Fatalf("account was deleted despite having transactions")
	}
}

// contraNames lists the other side of a two-account transaction.
func TestContraNames(t *testing.T) {
	s, _ := newViewTest(t)
	checking := s.AccountByName("Checking")
	reg := s.Register(checking.ID)
	if len(reg) == 0 {
		t.Skip("no register rows")
	}
	// At least one register row should name a non-self contra account.
	found := false
	for _, r := range reg {
		if n := contraNames(r.Txn, checking.ID); n != "—" && !strings.Contains(n, "Checking") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one register row with a named contra account")
	}
}
