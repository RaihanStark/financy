package core

// RecategorizeTransactions reassigns the income/expense category of each listed
// transaction to newCatID. It only touches transactions whose single category
// posting matches the new category's type (Expense→Expense, Income→Income);
// transfers, opening balances, splits, and mismatched-type entries are skipped so
// the books stay balanced and sign-correct. Returns the number actually changed.
func (s *Store) RecategorizeTransactions(ids []string, newCatID string) int {
	newCat := s.AccountByID(newCatID)
	if newCat == nil || (newCat.Type != Income && newCat.Type != Expense) {
		return 0
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}

	var changed []Transaction
	for _, t := range s.txns {
		if !want[t.ID] {
			continue
		}
		ci := s.categoryPostingIndex(t, newCat.Type)
		if ci < 0 || t.Posts[ci].AccountID == newCatID {
			continue // not a single-category match, or already there
		}
		nt := t
		nt.Posts = append([]Posting(nil), t.Posts...)
		nt.Posts[ci].AccountID = newCatID
		changed = append(changed, nt)
	}
	if len(changed) == 0 {
		return 0
	}
	if err := s.dbUpsertTxnBatch(changed); err != nil {
		s.reportError(err)
		return 0
	}
	pos := map[string]int{}
	for i := range s.txns {
		pos[s.txns[i].ID] = i
	}
	for _, nt := range changed {
		s.txns[pos[nt.ID]] = nt
	}
	s.notify()
	return len(changed)
}

// categoryPostingIndex returns the index of the one posting that targets an
// account of the given category type, or -1 when there isn't exactly one (e.g.
// a split with several expense legs, or a transaction of a different kind).
func (s *Store) categoryPostingIndex(t Transaction, typ AcctType) int {
	idx := -1
	for i, p := range t.Posts {
		a := s.AccountByID(p.AccountID)
		if a == nil || a.Type != typ {
			continue
		}
		if idx >= 0 {
			return -1 // ambiguous
		}
		idx = i
	}
	return idx
}
