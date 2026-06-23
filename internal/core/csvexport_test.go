package core

import (
	"strings"
	"testing"
)

func TestExportCSV(t *testing.T) {
	SetCurrencySymbol("$")
	s := &Store{accounts: seedAccounts(), nextID: 1}
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	s.AddAccount(Account{ID: "savings", Name: "Savings", Type: Asset})

	// Income, expense, and a transfer — one of each shape.
	s.AddTransaction(Transaction{Date: serialOf(2026, 6, 1), Payee: "Acme Payroll",
		Posts: PostingsFor(KindIncome, "checking", "salary", 450000)})
	s.AddTransaction(Transaction{Date: serialOf(2026, 6, 2), Payee: "Whole Foods", Memo: "weekly",
		Posts: PostingsFor(KindExpense, "checking", "groceries", 8420)})
	s.AddTransaction(Transaction{Date: serialOf(2026, 6, 3), Payee: "To Savings",
		Posts: PostingsFor(KindTransfer, "checking", "savings", 50000)})

	data, err := s.ExportCSV()
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n\r"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	if len(lines) != 4 { // header + 3
		t.Fatalf("got %d lines, want 4:\n%s", len(lines), data)
	}
	if lines[0] != "Date,Payee,Category,Account,Amount,Memo" {
		t.Fatalf("header = %q", lines[0])
	}
	// Chronological. Income: +4500 on Checking, category Salary.
	if lines[1] != "2026-06-01,Acme Payroll,Salary,Checking,4500.00," {
		t.Fatalf("income row = %q", lines[1])
	}
	// Expense: money out of Checking is negative, category Groceries, memo kept.
	if lines[2] != "2026-06-02,Whole Foods,Groceries,Checking,-84.20,weekly" {
		t.Fatalf("expense row = %q", lines[2])
	}
	// Transfer: from Checking (negative) → Savings as the category.
	if lines[3] != "2026-06-03,To Savings,Savings,Checking,-500.00," {
		t.Fatalf("transfer row = %q", lines[3])
	}
}
