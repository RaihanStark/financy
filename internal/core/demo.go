package core

// SeedDemo populates a freshly-created document with about six months of
// illustrative USD activity — recurring income, bills, groceries, dining,
// transfers and credit-card spending — so new users can see balances, the
// per-account register and period flows behave like a real file. It no-ops if
// the document already has money accounts.
//
// All amounts are integer minor units (cents); USD demo, so $1 == 100. The data
// is fully deterministic (no randomness) so tests and screenshots are stable.
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
	// Sign conventions (double-entry):
	//   expense from a bank:  P(category, +amt), P(bank, -amt)
	//   expense on credit:    P(category, +amt), P("visa", -amt)   (owed grows)
	//   pay the card:         P("visa", +amt),   P(bank, -amt)     (owed shrinks)
	//   income:               P(bank, +amt),     P(incomeCat, -amt)
	//   transfer:             P(toBank, +amt),   P(fromBank, -amt)
	expense := func(date int, payee, cat, bank string, amt int) { add(date, payee, P(cat, amt), P(bank, -amt)) }
	credit := func(date int, payee, cat string, amt int) { add(date, payee, P(cat, amt), P("visa", -amt)) }
	income := func(date int, payee, cat string, amt int) { add(date, payee, P("checking", amt), P(cat, -amt)) }

	// Opening balances, ~6 months back.
	start := d - 182
	add(start, "Opening Balance", P("checking", 250000), P("opening", -250000))
	add(start, "Opening Balance", P("savings", 800000), P("opening", -800000))
	add(start, "Opening Balance", P("cash", 12000), P("opening", -12000))
	add(start, "Credit card opening", P("visa", -60000), P("opening", 60000))

	// Six monthly cycles of recurring + occasional activity. Amounts drift a
	// little month to month so the journal looks lived-in rather than copied.
	for m := range 6 {
		base := start + m*30

		// Income — monthly payroll (with a small raise from month 3).
		salary := 450000
		if m >= 3 {
			salary = 470000
		}
		income(base+1, "Acme Corp Payroll", "salary", salary)

		// Fixed monthly bills.
		expense(base+2, "Landlord", "housing", "checking", 140000)
		expense(base+5, "City Utilities", "utilities", "checking", 9000+(m%3)*1500)
		expense(base+6, "Netflix & Spotify", "subs", "checking", 2998)

		// Groceries — a couple from checking, one on the card.
		expense(base+4, "Whole Foods", "groceries", "checking", 9800+(m%2)*1200)
		credit(base+13, "Trader Joe's", "groceries", 7300+(m%3)*900)
		expense(base+22, "Costco", "groceries", "checking", 11200+(m%2)*1500)

		// Dining — small cash lunch, a nicer dinner on the card.
		expense(base+8, "Chipotle", "dining", "cash", 1499+m*120)
		credit(base+17, "Sushi Night", "dining", 4200+(m%2)*1100)

		// Transport — fuel on the card, transit from checking.
		credit(base+9, "Shell Gas", "transport", 4500+(m%3)*700)
		expense(base+19, "City Transit", "transport", "checking", 2300+(m%2)*600)

		// Move some money to savings every month.
		add(base+3, "Transfer to Savings", P("savings", 50000), P("checking", -50000))

		// Refill the wallet from the bank every couple of months (ATM).
		if m%2 == 0 {
			add(base+7, "ATM Withdrawal", P("cash", 10000), P("checking", -10000))
		}

		// Pay down the card from the second month onward.
		if m >= 1 {
			add(base+25, "Credit Card Payment", P("visa", 18000+(m%2)*4000), P("checking", -(18000 + (m%2)*4000)))
		}

		// Occasional, staggered so months differ.
		if m%2 == 0 {
			credit(base+14, "Amazon", "shopping", 15800+m*1500)
		}
		if m%3 == 1 {
			expense(base+20, "Cinema", "ent", "checking", 3600+m*400)
		}
		if m%3 == 2 {
			expense(base+11, "Pharmacy", "health", "checking", 7200)
			income(base+27, "Brokerage Dividend", "invest", 4200+m*300)
		}
		if m%2 == 1 {
			income(base+15, "Freelance Project", "side", 32000+m*2500)
		}
	}
}
