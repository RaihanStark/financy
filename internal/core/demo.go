package core

// SeedDemo populates a freshly-created document with about six months of
// illustrative USD activity so every screen has something coherent to show:
// recurring income, bills, groceries and dining, transfers, credit-card spending
// and paydowns, an off-budget investments account that grows via auto-invest, and a full
// zero-based budget (monthly assignments, sinking funds and a vacation drawdown).
// New users can see balances, the per-account register, period flows, and the
// Budget all behave like a real file. It no-ops if the document already has
// money accounts.
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
	// A long-term investment account, marked off-budget: it counts toward Net
	// Worth but its balance isn't money you can assign in the budget.
	s.AddAccount(Account{ID: "investments", Name: "Investments", Type: Asset, Institution: "Demo Invest", Notes: "Long-term investments", OffBudget: true})

	// Two extra budget categories so the budget shows real sinking funds.
	s.AddAccount(Account{ID: "emergency", Name: "Emergency Fund", Type: Expense, Notes: "Savings goal"})
	s.AddAccount(Account{ID: "vacation", Name: "Vacation", Type: Expense, Notes: "Sinking fund"})

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
	add(start, "Investments opening", P("investments", 1500000), P("opening", -1500000))

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

		// Automatic monthly investment into the off-budget investments account.
		add(base+28, "Auto-Invest", P("investments", 30000), P("checking", -30000))

		// A vacation paid from savings mid-way, drawing down its sinking fund.
		if m == 4 {
			expense(base+18, "Beach Trip", "vacation", "savings", 120000)
		}

		// Refill the wallet from the bank every couple of months (ATM).
		if m%2 == 0 {
			add(base+7, "ATM Withdrawal", P("cash", 10000), P("checking", -10000))
		}

		// Pay down the card from the second month onward.
		if m >= 1 {
			add(base+25, "Credit Card Payment", P("visa", 18000+(m%2)*4000), P("checking", -(18000+(m%2)*4000)))
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
			income(base+27, "Investments Dividend", "invest", 4200+m*300)
		}
		if m%2 == 1 {
			income(base+15, "Freelance Project", "side", 32000+m*2500)
		}
	}

	// ---- Budget plan ----
	//
	// Budget the current month, zero-based: give every category a job and assign
	// the rest so Ready to Assign reaches 0. Assignments don't move money — they
	// give the cash you already hold a purpose — so this connects the Budget
	// screen to the journal: bills are funded, sinking funds (insurance, vacation,
	// emergency) hold money aside, and a couple of categories are left tight so
	// overspending can show. We only assign the current month — past months aren't
	// rewritten — matching how you'd budget going forward.
	plan := []struct {
		cat string
		amt int
	}{
		{"housing", 140000},   // matches rent exactly
		{"utilities", 13000},  // comfortably covers the bill
		{"groceries", 32000},  // runs tight some months
		{"dining", 7000},      // a little tight → occasional overspend
		{"transport", 8000},   // a little tight → occasional overspend
		{"subs", 3000},        // covers Netflix & Spotify
		{"shopping", 12000},   // tight vs. the Amazon months → overspends
		{"health", 9000},      // covers the occasional pharmacy run
		{"ent", 4000},         // covers the occasional cinema
		{"insurance", 50000},  // sinking fund, never spent → grows
		{"vacation", 24000},   // sinking fund, spent once mid-way
		{"emergency", 100000}, // savings goal, never spent → grows steadily
		{"misc", 5000},        // small buffer
	}
	month := CurrentMonthKey()
	for _, pl := range plan {
		s.SetAssigned(month, pl.cat, pl.amt)
	}
	// Zero-based finish: give every remaining dollar a job by topping up the
	// Emergency Fund, so Ready to Assign lands exactly on 0.
	if left := s.ReadyToAssign(month); left > 0 {
		s.SetAssigned(month, "emergency", s.Assigned(month, "emergency")+left)
	}

	// Recurring templates, due in the next couple of weeks so the Recurring screen
	// is populated without nagging on load.
	s.AddRecurring(Recurring{Kind: KindIncome, AcctA: "checking", AcctB: "salary",
		Amount: 470000, Payee: "Acme Corp Payroll", Freq: "Monthly", NextDue: d + 5, Enabled: true})
	s.AddRecurring(Recurring{Kind: KindExpense, AcctA: "checking", AcctB: "housing",
		Amount: 140000, Payee: "Landlord", Freq: "Monthly", NextDue: d + 2, Enabled: true})
	s.AddRecurring(Recurring{Kind: KindTransfer, AcctA: "checking", AcctB: "savings",
		Amount: 50000, Payee: "Transfer to Savings", Freq: "Monthly", NextDue: d + 3, Enabled: true})
	s.AddRecurring(Recurring{Kind: KindExpense, AcctA: "checking", AcctB: "subs",
		Amount: 2998, Payee: "Netflix & Spotify", Freq: "Monthly", NextDue: d + 8, Enabled: true})
}
