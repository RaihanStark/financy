package core

import (
	"testing"
)

// importStore is a clean in-memory store with the default categories plus a
// "checking" money account to import into.
func importStore(t *testing.T) *Store {
	t.Helper()
	s := &Store{accounts: seedAccounts(), owner: settingsOwner, currency: settingsCurrency, year: settingsYear}
	s.nextID = 1
	SetCurrencySymbol("$")
	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset})
	return s
}

func TestImportCommaSignedUSDates(t *testing.T) {
	s := importStore(t)
	recs, err := ParseCSV([]byte("Date,Description,Amount\n06/23/2026,WHOLE FOODS,-84.20\n06/24/2026,ACME PAYROLL,4500.00\n"))
	if err != nil {
		t.Fatal(err)
	}
	spec := GuessSpec(recs[0], recs[1:])
	spec.AccountID = "checking"
	spec.IncomeCatID, spec.ExpenseCatID = s.EnsureImportCategories()

	if spec.DateCol != 0 || spec.PayeeCol != 1 || spec.AmountCol != 2 {
		t.Fatalf("auto-detect cols wrong: %+v", spec)
	}
	if spec.DateLayout != "01/02/2006" {
		t.Fatalf("date layout = %q, want 01/02/2006", spec.DateLayout)
	}

	rows := s.BuildImport(recs, spec)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	// Row 1: expense $84.20 out of checking.
	if rows[0].Status != RowOK || rows[0].Amount != -8420 {
		t.Fatalf("row0 = %+v", rows[0])
	}
	if !rows[0].Txn.balanced() {
		t.Fatal("row0 txn not balanced")
	}
	// expense posting goes to Uncategorized expense, +8420; checking -8420
	if a := postingAmt(rows[0].Txn, "uncat_expense"); a != 8420 {
		t.Fatalf("expense posting = %d, want 8420", a)
	}
	if a := postingAmt(rows[0].Txn, "checking"); a != -8420 {
		t.Fatalf("checking posting = %d, want -8420", a)
	}
	// Row 2: income $4500 into checking.
	if rows[1].Amount != 450000 || postingAmt(rows[1].Txn, "checking") != 450000 ||
		postingAmt(rows[1].Txn, "uncat_income") != -450000 {
		t.Fatalf("row1 income wrong: %+v", rows[1].Txn)
	}
}

func TestImportSemicolonDebitCreditEUDates(t *testing.T) {
	s := importStore(t)
	csv := "Transaction Date;Payee;Debit;Credit\n23-06-2026;Whole Foods;84,20;\n24-06-2026;Acme Payroll;;4500,00\n"
	recs, err := ParseCSV([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs[0]) != 4 {
		t.Fatalf("semicolon not detected, header=%v", recs[0])
	}
	spec := GuessSpec(recs[0], recs[1:])
	spec.AccountID = "checking"
	spec.IncomeCatID, spec.ExpenseCatID = s.EnsureImportCategories()

	if spec.AmountMode != AmountDebitCredit || spec.DebitCol != 2 || spec.CreditCol != 3 {
		t.Fatalf("debit/credit detect wrong: %+v", spec)
	}
	rows := s.BuildImport(recs, spec)
	if rows[0].Amount != -8420 {
		t.Fatalf("debit row = %d, want -8420", rows[0].Amount)
	}
	if rows[1].Amount != 450000 {
		t.Fatalf("credit row = %d, want 450000", rows[1].Amount)
	}
}

func TestImportParenthesesNegative(t *testing.T) {
	if v, ok := parseSignedAmount("(1,234.50)"); !ok || v != -123450 {
		t.Fatalf("paren parse = %d ok=%v, want -123450", v, ok)
	}
	if v, ok := parseSignedAmount("1,234.50"); !ok || v != 123450 {
		t.Fatalf("plain parse = %d ok=%v, want 123450", v, ok)
	}
	if _, ok := parseSignedAmount(" "); ok {
		t.Fatal("blank should not parse")
	}
}

func TestImportDedupAndCommit(t *testing.T) {
	s := importStore(t)
	data := []byte("Date,Description,Amount\n2026-06-01,Landlord,-1400.00\n2026-06-02,Cafe,-12.50\n")
	recs, _ := ParseCSV(data)
	spec := GuessSpec(recs[0], recs[1:])
	spec.AccountID = "checking"
	spec.IncomeCatID, spec.ExpenseCatID = s.EnsureImportCategories()

	rows := s.BuildImport(recs, spec)
	var commit []Transaction
	for _, r := range rows {
		if r.Include {
			commit = append(commit, r.Txn)
		}
	}
	n, err := s.AddTransactions(commit)
	if err != nil || n != 2 {
		t.Fatalf("first import added %d (err %v), want 2", n, err)
	}

	// Re-import the same file: both rows should now be flagged duplicate.
	rows2 := s.BuildImport(recs, spec)
	dups := 0
	for _, r := range rows2 {
		if r.Status == RowDuplicate {
			dups++
		}
		if r.Include {
			t.Fatal("duplicate row should not be pre-selected")
		}
	}
	if dups != 2 {
		t.Fatalf("re-import flagged %d duplicates, want 2", dups)
	}
}

func TestImportAutoCategorizeByPayee(t *testing.T) {
	s := importStore(t)
	inc, exp := s.EnsureImportCategories()

	// A previously categorized expense teaches the importer: NETFLIX → subs.
	s.AddTransaction(Transaction{Date: TodaySerial - 30, Payee: "NETFLIX",
		Posts: []Posting{P("subs", 1599), P("checking", -1599)}})

	recs, _ := ParseCSV([]byte("Date,Description,Amount\n2026-06-10,NETFLIX,-15.99\n2026-06-11,UNKNOWN SHOP,-20.00\n"))
	spec := GuessSpec(recs[0], recs[1:])
	spec.AccountID = "checking"
	spec.IncomeCatID, spec.ExpenseCatID = inc, exp
	rows := s.BuildImport(recs, spec)

	if rows[0].CategoryID != "subs" {
		t.Fatalf("NETFLIX auto-category = %q, want subs", rows[0].CategoryID)
	}
	if postingAmt(rows[0].Txn, "subs") != 1599 {
		t.Fatalf("expected subs contra posting 1599, got txn %+v", rows[0].Txn)
	}
	if rows[1].CategoryID != exp {
		t.Fatalf("unknown payee category = %q, want default %q", rows[1].CategoryID, exp)
	}
}

func TestImportTokenAndSubstringMatching(t *testing.T) {
	s := importStore(t)
	inc, exp := s.EnsureImportCategories()

	// Clean history teaches short, canonical payees.
	learn := func(payee, cat string) {
		s.AddTransaction(Transaction{Date: TodaySerial - 30, Payee: payee,
			Posts: []Posting{P(cat, 1000), P("checking", -1000)}})
	}
	learn("Amazon", "shopping")
	learn("Whole Foods", "groceries")
	learn("Blue Bottle", "dining")
	learn("Netflix", "subs")

	csv := "Date,Description,Amount\n" +
		"2026-06-10,AMAZON.COM*A1B2C,-30.00\n" + // substring/token: amazon
		"2026-06-11,\"WHOLE FOODS MARKET, NYC\",-50.00\n" + // token subset
		"2026-06-12,SQ *BLUE BOTTLE 0042,-6.00\n" + // tokens blue+bottle (noise dropped)
		"2026-06-13,NETFLIX.COM,-15.99\n" + // substring
		"2026-06-14,APPLE STORE,-999.00\n" // matches nothing learned
	recs, _ := ParseCSV([]byte(csv))
	spec := GuessSpec(recs[0], recs[1:])
	spec.AccountID = "checking"
	spec.IncomeCatID, spec.ExpenseCatID = inc, exp
	rows := s.BuildImport(recs, spec)

	want := []string{"shopping", "groceries", "dining", "subs", exp}
	for i, w := range want {
		if rows[i].CategoryID != w {
			t.Errorf("row %d (%q) category = %q, want %q", i, rows[i].Payee, rows[i].CategoryID, w)
		}
	}
}

func TestImportBadRowsFlagged(t *testing.T) {
	s := importStore(t)
	recs, _ := ParseCSV([]byte("Date,Description,Amount\nnot-a-date,X,-10.00\n2026-06-01,Y,notmoney\n2026-06-02,Z,0\n"))
	spec := GuessSpec(recs[0], recs[1:])
	spec.AccountID = "checking"
	spec.IncomeCatID, spec.ExpenseCatID = s.EnsureImportCategories()
	rows := s.BuildImport(recs, spec)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	for i, r := range rows {
		if r.Status != RowError {
			t.Fatalf("row %d expected RowError, got %v (%s)", i, r.Status, r.Reason)
		}
	}
}

func postingAmt(t Transaction, account string) int {
	sum := 0
	for _, p := range t.Posts {
		if p.AccountID == account {
			sum += p.Amount
		}
	}
	return sum
}
