package core

import (
	"reflect"
	"testing"
)

func TestPayeesDistinctSortedAndFiltered(t *testing.T) {
	s := NewStore()
	s.AddAccount(Account{ID: "bank", Name: "Bank", Type: Asset})

	// Two distinct payees, a duplicate (different case), and a blank/whitespace
	// payee that must be filtered out.
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Netflix",
		Posts: []Posting{P("bank", 100), P("opening", -100)}})
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Acme Corp",
		Posts: []Posting{P("bank", 100), P("opening", -100)}})
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "netflix",
		Posts: []Posting{P("bank", 100), P("opening", -100)}})
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "   ",
		Posts: []Posting{P("bank", 100), P("opening", -100)}})
	s.AddTransaction(Transaction{Date: TodaySerial, Payee: "",
		Posts: []Posting{P("bank", 100), P("opening", -100)}})

	// Recurring payees are included and also deduped against transactions.
	s.AddRecurring(Recurring{Kind: "Expense", AcctA: "bank", AcctB: "groceries",
		Amount: 100, Payee: "Acme Corp", Freq: "Monthly", NextDue: TodaySerial, Enabled: true})
	s.AddRecurring(Recurring{Kind: "Expense", AcctA: "bank", AcctB: "groceries",
		Amount: 100, Payee: "Spotify", Freq: "Monthly", NextDue: TodaySerial, Enabled: true})

	got := s.Payees()
	want := []string{"Acme Corp", "Netflix", "Spotify"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Payees() = %v, want %v", got, want)
	}
}

func TestPayeesEmpty(t *testing.T) {
	s := NewStore()
	if got := s.Payees(); len(got) != 0 {
		t.Fatalf("fresh store Payees() = %v, want empty", got)
	}
}
