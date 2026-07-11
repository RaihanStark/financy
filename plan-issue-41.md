# Issue #41 — Deleting a debt payment transaction doesn't undo the installment

## Problem
When a debt installment is paid, `PayInstallment` (`internal/core/debt.go:316`) posts a
transaction and sets the installment's `Paid=true` + `TxnID`. If the user later deletes
**that transaction** from the Transactions page, `DeleteTransaction`
(`internal/core/store.go:606`) removes it with no debt awareness — it never touches
`s.installments`. The installment keeps `Paid=true` and a dangling `TxnID`, so the Debt page
still shows the installment as paid and the schedule further along than it really is. The
liability balance self-corrects (it's derived from postings), but the installment's paid
flag goes stale.

The correct reversal already exists in `UnpayInstallment` (`debt.go:393`) — it's just not
reached when the transaction is deleted directly.

## Approach
Make `DeleteTransaction` debt-aware: before finishing the delete, if any installment points
at the txn (`TxnID == id`), reset that installment to unpaid. Do it **inline** (mutate +
persist the installment), NOT by calling `UnpayInstallment`, to avoid recursion —
`UnpayInstallment` itself calls `DeleteTransaction` for non-linked payments.

## Changes

### `internal/core/debt.go` — add a helper
```go
// clearInstallmentForTxn resets any installment satisfied by txnID back to unpaid.
// Called from DeleteTransaction so deleting a debt payment from the Transactions
// page also rolls back the installment on the Debt page. It only mutates the
// installment (never deletes the txn), so it's safe to call mid-delete without
// recursing back into DeleteTransaction.
func (s *Store) clearInstallmentForTxn(txnID string) {
    for i := range s.installments {
        in := &s.installments[i]
        if in.TxnID != txnID {
            continue
        }
        in.Paid = false
        in.TxnID = ""
        in.Linked = false
        in.OrigPosts = nil
        s.reportError(s.dbUpsertInstallment(*in))
    }
}
```
For a **linked** installment (`Linked=true`) the transaction was the user's own; it's being
deleted, so we can't restore `OrigPosts` — reverting the installment to unpaid is the right
outcome (user can re-link later).

### `internal/core/store.go` — call the helper in `DeleteTransaction`
```go
func (s *Store) DeleteTransaction(id string) {
    for i := range s.txns {
        if s.txns[i].ID == id {
            if err := s.dbDeleteTxn(id); err != nil {
                s.reportError(err)
                return
            }
            s.txns = append(s.txns[:i], s.txns[i+1:]...)
            s.clearInstallmentForTxn(id) // roll back any debt installment paid by this txn
            s.notify()
            return
        }
    }
}
```

## Recursion / callers — verified safe
- `UnpayInstallment` non-linked branch calls `DeleteTransaction(in.TxnID)`; the helper now
  also clears the installment first — idempotent with the field-clearing that follows. No
  recursion, since the helper never calls `DeleteTransaction`.
- `DeleteDebt` (`debt.go:206`) deletes all liability-touching txns then drops the
  installments anyway (`s.installments = kept`) — extra unpay writes are harmless.
- UI deletes (desktop `internal/ui/view/transactions.go`, mobile
  `internal/mobileui/transaction.go`) both funnel through core `DeleteTransaction`, so no UI
  changes needed.

## Tests — add to `internal/core/debt_test.go`
- `TestDeleteTransaction_UnpaysInstallment`: create debt, `PayInstallment`, capture the
  installment's `TxnID`, `DeleteTransaction(txnID)`, assert the installment is now
  `Paid=false` / `TxnID==""` and `DebtStatus.PaidCount` + `Paid` dropped back.
- `TestDeleteTransaction_UnlinksLinkedInstallment`: `LinkInstallment` an existing txn, delete
  it, assert the installment reverts to unpaid and `Linked=false`.
- (optional) Regression: deleting an unrelated transaction leaves installments untouched.

Existing patterns to mirror: `TestPayInstallmentPostsTransaction` (debt_test.go:42) and
`TestLinkInstallmentReconcilesLiability` (debt_test.go:112).

## Out of scope
No schema change (relationship stays `Installment.TxnID → Transaction.ID`). No UI change.
The liability account balance already self-corrects from posting removal.
