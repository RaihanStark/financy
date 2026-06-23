package view

// Screenshot helpers — thin exported wrappers so the headless shot harness (in
// package ui) can open dialog states that are normally reached only through user
// interaction. They are not used by the running app.

// ShotReconcileDialog opens the reconcile dialog for the named money account.
func ShotReconcileDialog(account string) {
	if a := store.AccountByName(account); a != nil {
		reconcileAccount(*a)
	}
}

// ShotReconcileResult posts a reconciliation adjustment of +delta minor units to
// the named account (so the bank shows a little more than Financy) and opens its
// register, where the adjustment row is visible. This one mutates data, so the
// harness runs it after the read-only screen captures.
func ShotReconcileResult(account string, delta int) {
	a := store.AccountByName(account)
	if a == nil {
		return
	}
	store.Reconcile(a.ID, store.DisplayBalance(*a)+delta, todaySerial)
	showRegisterDialog(*a)
}

// ShotDuePrompt opens the recurring "due" prompt for everything due within
// daysAhead of today, so the screenshot shows several occurrences to act on.
func ShotDuePrompt(daysAhead int) {
	if items := store.PendingRecurring(todaySerial + daysAhead); len(items) > 0 {
		showDuePrompt(items)
	}
}
