package core

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

func TestPersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.financy")

	// Create a fresh document — should be seeded with default categories.
	s, err := NewDocument(path)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	if len(s.ExpenseAccounts()) == 0 || len(s.IncomeAccounts()) == 0 {
		t.Fatal("new document not seeded with categories")
	}
	if v, _ := FileVersion(path); v != LatestVersion() {
		t.Fatalf("file version %d, want %d", v, LatestVersion())
	}

	// Mutate: add account, opening balance, change settings.
	s.AddAccount(Account{Name: "Bank", Type: Asset, Institution: "Demo"})
	if !s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Open",
		Posts: []Posting{P("bank", 1_000_000), P("opening", -1_000_000)}}) {
		t.Fatal("AddTransaction failed")
	}
	s.SetSettings("", "$", 2030)
	wantNW := s.NetWorth()
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopen and verify everything persisted.
	s2, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = s2.Close() }()

	if s2.NetWorth() != wantNW || s2.NetWorth() != 1_000_000 {
		t.Fatalf("net worth after reopen = %d, want %d", s2.NetWorth(), wantNW)
	}
	if s2.Currency() != "$" || s2.Year() != 2030 {
		t.Fatalf("settings not persisted: currency=%q year=%d", s2.Currency(), s2.Year())
	}
	if len(s2.Transactions()) != 1 {
		t.Fatalf("transactions after reopen = %d, want 1", len(s2.Transactions()))
	}
	if a := s2.AccountByName("Bank"); a == nil || a.Institution != "Demo" {
		t.Fatal("account not persisted correctly")
	}

	// Delete round-trips too.
	tx := s2.Transactions()[0]
	s2.DeleteTransaction(tx.ID)
	if len(s2.Transactions()) != 0 {
		t.Fatal("delete not applied in memory")
	}
	_ = s2.Close()

	s3, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore 3: %v", err)
	}
	defer func() { _ = s3.Close() }()
	if len(s3.Transactions()) != 0 {
		t.Fatal("delete not persisted")
	}
}

func TestOpenRefusesFileFromNewerVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.financy")

	// Create a valid document, then forge a user_version higher than this build
	// knows about — as if a newer Financy had migrated it forward.
	s, err := NewDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d;`, LatestVersion()+1)); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	if _, err := OpenStore(path); !errors.Is(err, ErrFileTooNew) {
		t.Fatalf("OpenStore on newer file = %v, want ErrFileTooNew", err)
	}
}

func TestPersistenceErrorSurfaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "err.financy")
	s, err := NewDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	var captured error
	s.SetErrorHandler(func(e error) { captured = e })
	_ = s.Close() // close the DB so subsequent writes fail

	ok := s.AddTransaction(Transaction{Date: TodaySerial,
		Posts: []Posting{P("checking", 100), P("opening", -100)}})
	if ok {
		t.Fatal("expected AddTransaction to fail after Close")
	}
	if captured == nil {
		t.Fatal("persistence error was not surfaced to the handler")
	}
}
