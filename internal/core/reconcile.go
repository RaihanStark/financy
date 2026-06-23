package core

// Reconciliation: bring a money account's balance into line with the real-world
// balance (e.g. what your bank statement says) by posting a single adjustment
// transaction for the difference. The contra side goes to an Equity
// "Reconciliation" account, so the correction adjusts net worth without polluting
// income/expense reporting — the same idea as opening balances.

const reconcileAccountID = "reconcile"

// ensureReconcileAccount returns the ID of the equity account that absorbs
// reconciliation adjustments, creating it on first use.
func (s *Store) ensureReconcileAccount() string {
	if s.AccountByID(reconcileAccountID) == nil {
		s.AddAccount(Account{
			ID:    reconcileAccountID,
			Name:  "Reconciliation",
			Type:  Equity,
			Notes: "Balance adjustments from reconciling accounts",
		})
	}
	return reconcileAccountID
}

// Reconcile adjusts accountID so its displayed balance equals actualDisplay (the
// real balance the user reads from their bank, in natural/positive terms). If
// they already match, nothing is posted. Returns the change applied to the
// displayed balance and whether an adjustment transaction was created.
func (s *Store) Reconcile(accountID string, actualDisplay, date int) (delta int, created bool) {
	a := s.AccountByID(accountID)
	if a == nil {
		return 0, false
	}
	currentDisplay := s.DisplayBalance(*a)
	if actualDisplay == currentDisplay {
		return 0, false
	}

	// Convert the target displayed balance back to a raw signed balance, matching
	// DisplayBalance's sign flip for liabilities.
	targetRaw := actualDisplay
	if a.Type == Liability {
		targetRaw = -actualDisplay
	}
	rawDelta := targetRaw - s.Balance(accountID)

	contra := s.ensureReconcileAccount()
	s.AddTransaction(Transaction{
		Date:  date,
		Payee: "Reconciliation adjustment",
		Memo:  "Balance reconciled to " + FmtMoneyPlain(actualDisplay),
		Posts: []Posting{P(accountID, rawDelta), P(contra, -rawDelta)},
	})
	return actualDisplay - currentDisplay, true
}
