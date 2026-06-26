package view

import (
	"testing"

	"github.com/raihanstark/financy/internal/core"
)

var sampleCSV = [][]string{
	{"Date", "Payee", "Amount"},
	{"2026-06-01", "Coffee Shop", "-4.50"},
	{"2026-06-02", "Acme Payroll", "2000.00"},
	{"2026-06-03", "Grocery Mart", "-63.20"},
}

// ImportCSV refuses to start when there are no money accounts to import into.
func TestImportCSVNoAccounts(t *testing.T) {
	_, w := emptyViewTest(t)
	ImportCSV()
	d := dialogContent(w)
	if d == nil {
		t.Fatal("expected an info dialog")
	}
	assertText(t, d, "No accounts yet")
}

// The mapping step opens, guesses the columns and offers a Preview action.
func TestImportMappingDialog(t *testing.T) {
	s, w := newViewTest(t)
	s.EnsureImportCategories()

	showImportMapping(sampleCSV)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("import mapping dialog did not open")
	}
	assertText(t, d, "Import CSV", "Into account", "Date", "Payee", "Amount", "Preview")
}

// Driving the full wizard — map, preview, import — commits the parsed rows as
// new transactions in the target account.
func TestImportWizardCommits(t *testing.T) {
	s, w := newViewTest(t)
	s.EnsureImportCategories()
	before := len(s.Transactions())

	showImportMapping(sampleCSV)
	mapping := dialogContent(w)
	if mapping == nil {
		t.Fatal("mapping dialog missing")
	}
	if !tapButton(mapping, "Preview") {
		t.Fatal("no Preview button")
	}

	preview := dialogContent(w)
	if preview == nil {
		t.Fatal("preview dialog did not open")
	}
	assertText(t, preview, "Review import", "Date", "Payee", "Amount", "Category", "Status")

	if !tapButton(preview, "Import") {
		t.Fatal("no Import button in preview")
	}
	if got := len(s.Transactions()); got <= before {
		t.Fatalf("no transactions imported: %d → %d", before, got)
	}
	// The completion info dialog reports the result.
	if done := dialogContent(w); done != nil {
		assertText(t, done, "Import complete")
	}
}

// columnLabels names columns from the header row, numbered.
func TestColumnLabels(t *testing.T) {
	labels := columnLabels(sampleCSV)
	if len(labels) != 3 {
		t.Fatalf("got %d labels, want 3", len(labels))
	}
	assertContains(t, labels[0], "Date")
	assertContains(t, labels[1], "Payee")
}

// colIndexOf parses the 1-based prefix back to a 0-based column index.
func TestColIndexOf(t *testing.T) {
	labels := columnLabels(sampleCSV)
	if got := colIndexOf(labels[2]); got != 2 {
		t.Errorf("colIndexOf(%q) = %d, want 2", labels[2], got)
	}
	if got := colIndexOf(noneOpt); got != -1 {
		t.Errorf("colIndexOf(none) = %d, want -1", got)
	}
	if got := colIndexOf(""); got != -1 {
		t.Errorf("colIndexOf(empty) = %d, want -1", got)
	}
}

// Date-format label/layout round-trips through both lookup helpers.
func TestDateFormatRoundTrip(t *testing.T) {
	for _, o := range dateFormatOptions {
		if got := layoutForLabel(o.label); got != o.layout {
			t.Errorf("layoutForLabel(%q) = %q, want %q", o.label, got, o.layout)
		}
		if got := labelForLayout(o.layout); got != o.label {
			t.Errorf("labelForLayout(%q) = %q, want %q", o.layout, got, o.label)
		}
	}
	// Unknown layout falls back to Auto-detect.
	if labelForLayout("nonsense") != dateFormatOptions[0].label {
		t.Error("unknown layout should fall back to Auto-detect")
	}
}

// idForName and orFirst behave for present and absent values.
func TestImportNameHelpers(t *testing.T) {
	s, _ := newViewTest(t)
	accts := s.ExpenseAccounts()
	if len(accts) == 0 {
		t.Skip("no expense accounts")
	}
	if got := idForName(accts, accts[0].Name); got != accts[0].ID {
		t.Errorf("idForName = %q, want %q", got, accts[0].ID)
	}
	if got := idForName(accts, "no-such-name"); got != "" {
		t.Errorf("idForName(absent) = %q, want empty", got)
	}
	opts := []string{"a", "b", "c"}
	if orFirst(opts, "b") != "b" {
		t.Error("orFirst should return the present value")
	}
	if orFirst(opts, "z") != "a" {
		t.Error("orFirst should fall back to the first option")
	}
	if orFirst(nil, "z") != "" {
		t.Error("orFirst(nil) should be empty")
	}
}

// sampleRows skips the header and caps at five data rows.
func TestSampleRows(t *testing.T) {
	got := sampleRows(sampleCSV)
	if len(got) != 3 {
		t.Fatalf("sampleRows = %d rows, want 3", len(got))
	}
	if got[0][1] != "Coffee Shop" {
		t.Errorf("first sample row payee = %q", got[0][1])
	}
	if sampleRows([][]string{{"only-header"}}) != nil {
		t.Error("single-row input should yield no sample rows")
	}
}

// importedDups counts duplicate rows only when they're being included.
func TestImportedDups(t *testing.T) {
	rows := []core.ImportRow{
		{Status: core.RowOK}, {Status: core.RowDuplicate}, {Status: core.RowDuplicate},
	}
	if got := importedDups(rows, false); got != 0 {
		t.Errorf("importedDups(exclude) = %d, want 0", got)
	}
	if got := importedDups(rows, true); got != 2 {
		t.Errorf("importedDups(include) = %d, want 2", got)
	}
}
