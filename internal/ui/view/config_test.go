package view

import (
	"testing"
)

// OpenConfig shows the Preferences dialog with its three tabs.
func TestOpenConfigShowsTabs(t *testing.T) {
	_, w := newViewTest(t)
	OpenConfig(0)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("preferences dialog did not open")
	}
	assertText(t, d, "Configuration", "Categories", "Data Summary")
}

// The configuration tab shows the currency and number-format pickers.
func TestConfigBody(t *testing.T) {
	s, w := newViewTest(t)
	body := mount(w, configBody(nil))
	assertText(t, body, "Currency & Format", "Save Changes")

	// The currency select is seeded from the store.
	selects := findSelects(body)
	if len(selects) < 1 || selects[0].Selected != s.Currency() {
		t.Fatalf("currency select not seeded from store (%q)", s.Currency())
	}
}

// Saving the configuration tab writes the chosen currency back to the store.
func TestConfigSaveChangesCurrency(t *testing.T) {
	s, w := newViewTest(t)
	saved := false
	body := mount(w, configBody(func() { saved = true }))

	selects := findSelects(body)
	selects[0].SetSelected("€") // currency
	if !tapButton(body, "Save Changes") {
		t.Fatal("no Save Changes button")
	}
	if s.Currency() != "€" {
		t.Fatalf("currency = %q, want €", s.Currency())
	}
	if !saved {
		t.Error("onSaved callback not invoked")
	}
}

// The categories tab lists income and expense categories.
func TestCategoriesBody(t *testing.T) {
	_, w := newViewTest(t)
	body := mount(w, categoriesBody(func() {}))
	assertText(t, body, "INCOME", "EXPENSES", "Salary", "Groceries", "Add Category")
}

// The data summary tab reports counts and totals from the store.
func TestDataSummaryBody(t *testing.T) {
	s, w := newViewTest(t)
	body := mount(w, dataSummaryBody())
	assertText(t, body,
		"Data Summary", "Transactions", "Net worth",
		itoa(len(s.Transactions())),
		fmtMoney(s.NetWorth()),
		"Financy v"+version,
	)
}

// Adding a category through CategoryForm persists it.
func TestCategoryFormAdd(t *testing.T) {
	s, w := newViewTest(t)
	before := len(s.ExpenseAccounts())

	CategoryForm(nil, nil)
	d := dialogContent(w)
	if d == nil {
		t.Fatal("category form did not open")
	}
	entries := findEntries(d)
	entries[0].SetText("Travel")
	if !tapButton(d, "Save") {
		t.Fatal("no Save button")
	}
	if got := len(s.ExpenseAccounts()); got != before+1 {
		t.Fatalf("expense categories = %d, want %d", got, before+1)
	}
}

// Deleting an unused category removes it.
func TestDeleteUnusedCategory(t *testing.T) {
	s, w := newViewTest(t)
	s.AddAccount(Account{Name: "Temp Cat", Type: Expense})
	cat := s.AccountByName("Temp Cat")
	before := len(s.ExpenseAccounts())

	deleteCategory(*cat, func() {})
	// Confirm dialog: tap its confirm button.
	d := dialogContent(w)
	if d == nil {
		t.Fatal("expected delete-confirm dialog")
	}
	// The delete-confirm modal's affirmative button is labeled "Delete".
	if !tapButton(d, "Delete") {
		t.Fatal("no confirm (Delete) button in delete dialog")
	}
	if got := len(s.ExpenseAccounts()); got != before-1 {
		t.Fatalf("expense categories = %d, want %d", got, before-1)
	}
}
