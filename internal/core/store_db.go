package core

import (
	"database/sql"
	"strconv"
)

// loadStore reads an open database into an in-memory Store. The store keeps the
// DB handle and writes through on every mutation.
func loadStore(db *sql.DB) (*Store, error) {
	s := &Store{
		db:          db,
		owner:       "",
		currency:    "Rp",
		year:        settingsYear,
		assignments: map[string]int{},
	}

	// Settings.
	rows, err := db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			_ = rows.Close()
			return nil, err
		}
		switch k {
		case "owner":
			s.owner = v
		case "currency":
			s.currency = v
		case "year":
			if n, e := strconv.Atoi(v); e == nil {
				s.year = n
			}
		case "number_format":
			s.numberFormat = v
		}
	}
	_ = rows.Close()

	// Accounts (insertion order).
	arows, err := db.Query(`SELECT id, name, type, institution, notes, off_budget FROM accounts ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	for arows.Next() {
		var a Account
		var t string
		var off int
		if err := arows.Scan(&a.ID, &a.Name, &t, &a.Institution, &a.Notes, &off); err != nil {
			_ = arows.Close()
			return nil, err
		}
		a.Type = AcctType(t)
		a.OffBudget = off != 0
		s.accounts = append(s.accounts, a)
	}
	_ = arows.Close()

	// Transactions (insertion order) + their postings.
	trows, err := db.Query(`SELECT id, date, payee, memo FROM transactions ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	byID := map[string]int{}
	for trows.Next() {
		var t Transaction
		if err := trows.Scan(&t.ID, &t.Date, &t.Payee, &t.Memo); err != nil {
			_ = trows.Close()
			return nil, err
		}
		byID[t.ID] = len(s.txns)
		s.txns = append(s.txns, t)
	}
	_ = trows.Close()

	prows, err := db.Query(`SELECT txn_id, account_id, amount FROM postings ORDER BY txn_id, seq`)
	if err != nil {
		return nil, err
	}
	for prows.Next() {
		var txnID, acctID string
		var amt int
		if err := prows.Scan(&txnID, &acctID, &amt); err != nil {
			_ = prows.Close()
			return nil, err
		}
		if i, ok := byID[txnID]; ok {
			s.txns[i].Posts = append(s.txns[i].Posts, Posting{AccountID: acctID, Amount: amt})
		}
	}
	_ = prows.Close()

	// Recurring templates (table added in migration v2).
	rrows, err := db.Query(`SELECT id, kind, acct_a, acct_b, amount, payee, memo, freq, next_due, enabled FROM recurring ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	for rrows.Next() {
		var r Recurring
		var en int
		if err := rrows.Scan(&r.ID, &r.Kind, &r.AcctA, &r.AcctB, &r.Amount, &r.Payee, &r.Memo, &r.Freq, &r.NextDue, &en); err != nil {
			_ = rrows.Close()
			return nil, err
		}
		r.Enabled = en != 0
		s.recurring = append(s.recurring, r)
	}
	_ = rrows.Close()

	// Budget assignments (table added in migration v3).
	brows, err := db.Query(`SELECT month, category_id, amount FROM budget`)
	if err != nil {
		return nil, err
	}
	for brows.Next() {
		var month, catID string
		var amt int
		if err := brows.Scan(&month, &catID, &amt); err != nil {
			_ = brows.Close()
			return nil, err
		}
		s.assignments[monthKey(month, catID)] = amt
	}
	_ = brows.Close()

	// Debts (table added in migration v5) and their installment schedules.
	drows, err := db.Query(`SELECT id, name, type, lender, acct_money, purchase_date, note, acct_liability, origin_txn FROM debts ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	for drows.Next() {
		var d Debt
		if err := drows.Scan(&d.ID, &d.Name, &d.Type, &d.Lender, &d.AcctMoney, &d.PurchaseDate, &d.Note, &d.AcctLiability, &d.OriginTxnID); err != nil {
			_ = drows.Close()
			return nil, err
		}
		s.debts = append(s.debts, d)
	}
	_ = drows.Close()

	irows, err := db.Query(`SELECT id, debt_id, seq, due_date, amount, paid, txn_id FROM debt_installments ORDER BY debt_id, seq`)
	if err != nil {
		return nil, err
	}
	for irows.Next() {
		var in Installment
		var paid int
		if err := irows.Scan(&in.ID, &in.DebtID, &in.Seq, &in.DueDate, &in.Amount, &paid, &in.TxnID); err != nil {
			_ = irows.Close()
			return nil, err
		}
		in.Paid = paid != 0
		s.installments = append(s.installments, in)
	}
	_ = irows.Close()

	s.nextID = s.computeNextID()
	SetCurrencySymbol(s.currency)
	ApplyNumberFormat(s.numberFormat)
	return s, nil
}

func (s *Store) dbUpsertRecurring(r Recurring) error {
	if s.db == nil {
		return nil
	}
	en := 0
	if r.Enabled {
		en = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO recurring (id, kind, acct_a, acct_b, amount, payee, memo, freq, next_due, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind = excluded.kind, acct_a = excluded.acct_a, acct_b = excluded.acct_b,
			amount = excluded.amount, payee = excluded.payee, memo = excluded.memo,
			freq = excluded.freq, next_due = excluded.next_due, enabled = excluded.enabled`,
		r.ID, r.Kind, r.AcctA, r.AcctB, r.Amount, r.Payee, r.Memo, r.Freq, r.NextDue, en)
	return err
}

func (s *Store) dbDeleteRecurring(id string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM recurring WHERE id = ?`, id)
	return err
}

func (s *Store) dbUpsertDebt(d Debt) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`
		INSERT INTO debts (id, name, type, lender, acct_money, purchase_date, note, acct_liability, origin_txn)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name, type = excluded.type, lender = excluded.lender,
			acct_money = excluded.acct_money, purchase_date = excluded.purchase_date,
			note = excluded.note, acct_liability = excluded.acct_liability,
			origin_txn = excluded.origin_txn`,
		d.ID, d.Name, d.Type, d.Lender, d.AcctMoney, d.PurchaseDate, d.Note, d.AcctLiability, d.OriginTxnID)
	return err
}

func (s *Store) dbDeleteDebt(id string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM debts WHERE id = ?`, id) // cascades to installments
	return err
}

func (s *Store) dbUpsertInstallment(in Installment) error {
	if s.db == nil {
		return nil
	}
	paid := 0
	if in.Paid {
		paid = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO debt_installments (id, debt_id, seq, due_date, amount, paid, txn_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			debt_id = excluded.debt_id, seq = excluded.seq, due_date = excluded.due_date,
			amount = excluded.amount, paid = excluded.paid, txn_id = excluded.txn_id`,
		in.ID, in.DebtID, in.Seq, in.DueDate, in.Amount, paid, in.TxnID)
	return err
}

func (s *Store) dbUpsertAssignment(month, catID string, amount int) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`
		INSERT INTO budget (month, category_id, amount) VALUES (?, ?, ?)
		ON CONFLICT(month, category_id) DO UPDATE SET amount = excluded.amount`,
		month, catID, amount)
	return err
}

func (s *Store) dbDeleteAssignment(month, catID string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM budget WHERE month = ? AND category_id = ?`, month, catID)
	return err
}

// computeNextID derives the next "t<N>" id from existing transaction ids.
func (s *Store) computeNextID() int {
	max := 0
	for _, t := range s.txns {
		if len(t.ID) > 1 && t.ID[0] == 't' {
			if n, err := strconv.Atoi(t.ID[1:]); err == nil && n > max {
				max = n
			}
		}
	}
	return max + 1
}

// ---- write-through helpers (no-ops when db is nil) ----

func (s *Store) dbUpsertAccount(a Account) error {
	if s.db == nil {
		return nil
	}
	off := 0
	if a.OffBudget {
		off = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO accounts (id, name, type, institution, notes, off_budget)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name, type = excluded.type,
			institution = excluded.institution, notes = excluded.notes,
			off_budget = excluded.off_budget`,
		a.ID, a.Name, string(a.Type), a.Institution, a.Notes, off)
	return err
}

func (s *Store) dbDeleteAccount(id string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	return err
}

// dbUpsertTxn writes a transaction and its postings atomically.
func (s *Store) dbUpsertTxn(t Transaction) error {
	if s.db == nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO transactions (id, date, payee, memo) VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET date = excluded.date, payee = excluded.payee, memo = excluded.memo`,
		t.ID, t.Date, t.Payee, t.Memo); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM postings WHERE txn_id = ?`, t.ID); err != nil {
		_ = tx.Rollback()
		return err
	}
	for i, p := range t.Posts {
		if _, err := tx.Exec(`INSERT INTO postings (txn_id, seq, account_id, amount) VALUES (?, ?, ?, ?)`,
			t.ID, i, p.AccountID, p.Amount); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// dbInsertTxnBatch inserts many transactions in a single DB transaction — much
// faster and atomic (all-or-nothing) compared with calling dbUpsertTxn per row.
func (s *Store) dbInsertTxnBatch(txns []Transaction) error {
	if s.db == nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, t := range txns {
		if _, err := tx.Exec(`INSERT INTO transactions (id, date, payee, memo) VALUES (?, ?, ?, ?)`,
			t.ID, t.Date, t.Payee, t.Memo); err != nil {
			_ = tx.Rollback()
			return err
		}
		for i, p := range t.Posts {
			if _, err := tx.Exec(`INSERT INTO postings (txn_id, seq, account_id, amount) VALUES (?, ?, ?, ?)`,
				t.ID, i, p.AccountID, p.Amount); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

// dbUpsertTxnBatch rewrites many existing transactions (and their postings) in a
// single DB transaction — used by bulk edits like recategorization.
func (s *Store) dbUpsertTxnBatch(txns []Transaction) error {
	if s.db == nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, t := range txns {
		if _, err := tx.Exec(`
			INSERT INTO transactions (id, date, payee, memo) VALUES (?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET date = excluded.date, payee = excluded.payee, memo = excluded.memo`,
			t.ID, t.Date, t.Payee, t.Memo); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`DELETE FROM postings WHERE txn_id = ?`, t.ID); err != nil {
			_ = tx.Rollback()
			return err
		}
		for i, p := range t.Posts {
			if _, err := tx.Exec(`INSERT INTO postings (txn_id, seq, account_id, amount) VALUES (?, ?, ?, ?)`,
				t.ID, i, p.AccountID, p.Amount); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) dbDeleteTxn(id string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM transactions WHERE id = ?`, id) // cascades to postings
	return err
}

func (s *Store) dbSetSettings() error {
	if s.db == nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	set := func(k, v string) error {
		_, e := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v)
		return e
	}
	if err := set("owner", s.owner); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := set("currency", s.currency); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := set("year", Itoa(s.year)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := set("number_format", s.numberFormat); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
