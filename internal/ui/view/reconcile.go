package view

import (
	"fyne.io/fyne/v2/widget"
)

// reconcileAccount opens the reconcile dialog for a money account: show Financy's
// balance, ask for the real (bank) balance, and post an adjustment for any
// difference so the two match.
func reconcileAccount(a Account) {
	if win == nil {
		return
	}
	current := store.DisplayBalance(a)

	actual := newAmountEntry()
	actual.SetText(fmtMoneyInput(current)) // start from the current balance
	date := newDateEntry(todaySerial)

	owedOrBalance := "balance"
	if a.Type == Liability {
		owedOrBalance = "amount owed"
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Financy "+owedOrBalance, txt(fmtMoney(current), moneyColor(current), 14, true)),
		widget.NewFormItem("Actual "+owedOrBalance, actual),
		widget.NewFormItem("As of", date),
	}

	showForm("Reconcile "+a.Name, items, func() {
		target := parseAmount(actual.Text)
		d := parseDateSerial(date.Text)
		if d == 0 {
			d = todaySerial
		}
		delta, created := store.Reconcile(a.ID, target, d)
		if !created {
			showInfo("Already balanced",
				a.Name+" already matches at "+fmtMoney(current)+". No adjustment was needed.")
			return
		}
		sign := ""
		if delta > 0 {
			sign = "+"
		}
		showInfo("Reconciled",
			"Adjusted "+a.Name+" by "+sign+fmtMoneyPlain(delta)+".\nNew "+owedOrBalance+": "+fmtMoney(target)+".")
	})
}
