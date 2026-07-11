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

func TestMigrateV8DebtsBackfillPrincipal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v8.financy")

	// Build a schema-v8 document by hand — the state of a real pre-rework file —
	// with one BNPL debt: one paid, one linked, one unpaid installment.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for v := 0; v < 8; v++ {
		if _, err := db.Exec(migrations[v]); err != nil {
			t.Fatalf("apply v%d: %v", v+1, err)
		}
	}
	if _, err := db.Exec(`PRAGMA user_version = 8;`); err != nil {
		t.Fatal(err)
	}
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	mustExec(`INSERT INTO accounts (id, name, type, off_budget) VALUES
		('checking', 'Checking', 'Asset', 0),
		('shopping', 'Shopping', 'Expense', 0),
		('equity_financed', 'Financed Purchases', 'Equity', 0),
		('debt_d1', 'TV', 'Liability', 1)`)
	mustExec(`INSERT INTO debts (id, name, type, lender, acct_money, note, acct_liability, origin_txn, purchase_date)
		VALUES ('d1', 'TV', 'BNPL', 'Kredivo', 'checking', '', 'debt_d1', 't1', 46000)`)
	mustExec(`INSERT INTO transactions (id, date, payee, memo) VALUES
		('t1', 46000, 'Kredivo', 'TV · purchase'),
		('t2', 46010, 'Kredivo', 'TV · installment 1'),
		('t3', 46040, 'Kredivo', 'TV · installment 2')`)
	mustExec(`INSERT INTO postings (txn_id, seq, account_id, amount) VALUES
		('t1', 0, 'debt_d1', -3000), ('t1', 1, 'equity_financed', 3000),
		('t2', 0, 'debt_d1', 1000), ('t2', 1, 'checking', -1000),
		('t3', 0, 'debt_d1', 1000), ('t3', 1, 'checking', -1000)`)
	mustExec(`INSERT INTO debt_installments (id, debt_id, seq, due_date, amount, paid, txn_id, linked, orig_posts) VALUES
		('i1', 'd1', 1, 46010, 1000, 1, 't2', 0, ''),
		('i2', 'd1', 2, 46040, 1000, 1, 't3', 1, '[{"AccountID":"checking","Amount":-1000},{"AccountID":"shopping","Amount":1000}]'),
		('i3', 'd1', 3, 46070, 1000, 0, '', 0, '')`)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Opening migrates v8 → v9; the BNPL debt must behave identically.
	s, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore after migration: %v", err)
	}
	defer func() { _ = s.Close() }()

	if v, _ := FileVersion(path); v != LatestVersion() {
		t.Fatalf("file version after open = %d, want %d", v, LatestVersion())
	}
	d := s.Debts()[0]
	if d.Type != DebtBNPL || !d.HasSchedule() || !d.HasEnvelope() || d.IsRevolving() {
		t.Fatalf("migrated debt misclassified: %+v", d)
	}
	insts := s.Installments("d1")
	if len(insts) != 3 {
		t.Fatalf("got %d installments, want 3", len(insts))
	}
	for _, in := range insts {
		// The backfill marks every legacy installment 100% principal.
		if in.Principal != in.Amount || in.Interest != 0 {
			t.Errorf("installment %s split = P%d/I%d, want P%d/I0", in.ID, in.Principal, in.Interest, in.Amount)
		}
	}
	if !insts[0].Paid || insts[0].TxnID != "t2" {
		t.Errorf("paid installment lost: %+v", insts[0])
	}
	if !insts[1].Linked || len(insts[1].OrigPosts) != 2 {
		t.Errorf("linked installment lost: %+v", insts[1])
	}
	st := s.DebtStatus("d1", 46070)
	if st.PaidCount != 2 || st.Paid != 2000 || st.Remaining != 1000 || st.OverdueCount != 1 {
		t.Errorf("status after migration = %+v, want PaidCount=2 Paid=2000 Remaining=1000 Overdue=1", st)
	}
	// The whole flow still works post-migration: pay the last installment.
	if !s.PayInstallment("i3", 46070) {
		t.Fatal("PayInstallment on migrated debt failed")
	}
	if out := s.DisplayBalance(*s.AccountByID("debt_d1")); out != 0 {
		t.Errorf("liability after final payment = %d, want 0", out)
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
