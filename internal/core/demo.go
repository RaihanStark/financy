package core

// SeedDemo populates a freshly-created document with a small, illustrative set
// of USD accounts and transactions so new users can explore a working file.
// It no-ops if the document already has money accounts.
func SeedDemo(s *Store, currency string) {
	if len(s.MoneyAccounts()) > 0 {
		return
	}
	s.SetSettings("", currency, s.year)

	s.AddAccount(Account{ID: "checking", Name: "Checking", Type: Asset, Institution: "Demo Bank", Notes: "Everyday spending"})
	s.AddAccount(Account{ID: "savings", Name: "Savings", Type: Asset, Institution: "Demo Bank", Notes: "Rainy-day fund"})
	s.AddAccount(Account{ID: "cash", Name: "Cash", Type: Asset, Institution: "Wallet"})
	s.AddAccount(Account{ID: "visa", Name: "Credit Card", Type: Liability, Institution: "Demo Card"})

	d := TodaySerial
	add := func(date int, payee string, posts ...Posting) {
		s.AddTransaction(Transaction{Date: date, Payee: payee, Posts: posts})
	}

	// Amounts are in minor units (cents). USD demo, so $1 == 100.
	// Opening balances.
	add(d-40, "Opening Balance", P("checking", 320000), P("opening", -320000))
	add(d-40, "Opening Balance", P("savings", 1200000), P("opening", -1200000))
	add(d-40, "Opening Balance", P("cash", 15000), P("opening", -15000))
	add(d-40, "Credit card opening", P("visa", -85000), P("opening", 85000))

	// Activity.
	add(d-25, "Acme Corp", P("checking", 450000), P("salary", -450000))
	add(d-24, "Landlord", P("housing", 140000), P("checking", -140000))
	add(d-20, "Whole Foods", P("groceries", 24050), P("checking", -24050))
	add(d-18, "Chipotle", P("dining", 1499), P("cash", -1499))
	add(d-15, "Transfer to Savings", P("savings", 50000), P("checking", -50000))
	add(d-12, "Netflix & Spotify", P("subs", 2998), P("checking", -2998))
	add(d-10, "Uber", P("transport", 2375), P("checking", -2375))
	add(d-8, "Trader Joe's", P("groceries", 9532), P("checking", -9532))
	add(d-5, "Credit card payment", P("visa", 30000), P("checking", -30000))
	add(d-3, "Sushi night", P("dining", 5240), P("checking", -5240))
}
