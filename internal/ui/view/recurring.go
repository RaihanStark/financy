package view

import (
	"errors"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

var errAmount = errors.New("enter an amount greater than zero")

// ScreenRecurring lists recurring templates and lets you add/edit/delete them and
// post what's due.
func ScreenRecurring() fyne.CanvasObject {
	bar := appBar("Recurring", "Scheduled transactions — posted on your confirmation when due",
		primaryButton("Add Recurring", theme.ContentAddIcon(), func() { RecurringForm(nil) }),
	)

	rs := store.Recurrings()
	if len(rs) == 0 {
		body := panel(emptyState("No recurring transactions yet. Add rent, salary, subscriptions…"))
		return container.NewBorder(bar, nil, nil, nil, container.NewPadded(body))
	}

	rows := make([]fyne.CanvasObject, 0, len(rs))
	for _, r := range rs {
		rows = append(rows, recurringRow(r))
	}
	body := panel(container.NewVBox(rows...))
	return container.NewBorder(bar, nil, nil, nil, container.NewPadded(container.NewVBox(
		dueBanner(), body,
	)))
}

// dueBanner shows a one-line nudge when something is due.
func dueBanner() fyne.CanvasObject {
	n := len(store.PendingRecurring(todaySerial))
	if n == 0 {
		return spacerH(0)
	}
	msg := itoa(n) + " recurring entr" + plural(n, "y", "ies") + " due"
	return container.NewBorder(nil, spacerH(6), nil,
		secondaryButton("Review & Post", theme.MediaPlayIcon(), postDueNow),
		container.NewHBox(txt("●  ", colWarning, 13, true), txt(msg, colText, 12.5, true)))
}

func recurringRow(r Recurring) fyne.CanvasObject {
	amtCol := colNegative
	switch r.Kind {
	case "Income":
		amtCol = colPositive
	case "Transfer":
		amtCol = colText
	}

	menuBtn := widget.NewButtonWithIcon("", theme.MoreVerticalIcon(), nil)
	menuBtn.Importance = widget.LowImportance
	menuBtn.OnTapped = func() {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn)
		pos.Y += menuBtn.Size().Height
		showContextMenu(pos, recurringMenu(r)...)
	}

	title := r.Payee
	if title == "" {
		title = r.Kind
	}
	sub := r.Kind + " · " + orDash(nameOf(r.AcctB)) + " · " + r.Freq

	due := r.Enabled && r.NextDue <= todaySerial
	state, stateCol := "next · "+fmtSerialDate(r.NextDue), colTextDim
	switch {
	case !r.Enabled:
		state = "paused"
	case due:
		state, stateCol = "● due · "+fmtSerialDate(r.NextDue), colWarning
	}

	left := container.NewVBox(txt(title, colText, 13.5, true), txt(sub, colTextDim, 10.5, false))
	right := container.NewVBox(
		alignRight(txt(amountLabel(r.Kind, r.Amount), amtCol, 14, true)),
		alignRight(txt(state, stateCol, 10, due)),
	)
	inner := container.NewBorder(nil, nil, left, container.NewHBox(right, menuBtn))

	var fill color.Color = colSurface
	if due {
		fill = withAlpha(colWarning, 0x14) // faint amber tint so due rows stand out
	}
	row := newTappableRow(container.New(padCell(8, 12), inner), fill, func() { RecurringForm(&r) })
	row.SetBordered()
	row.SetOnSecondary(func(pos fyne.Position) { showContextMenu(pos, recurringMenu(r)...) })
	return row
}

func recurringMenu(r Recurring) []*fyne.MenuItem {
	postNow := fyne.NewMenuItem("Post now (early)…", func() { postRecurringNow(r) })
	postNow.Icon = theme.MediaPlayIcon()
	edit := fyne.NewMenuItem("Edit…", func() { RecurringForm(&r) })
	edit.Icon = theme.DocumentCreateIcon()
	toggleLabel := "Pause"
	if !r.Enabled {
		toggleLabel = "Resume"
	}
	toggle := fyne.NewMenuItem(toggleLabel, func() {
		r.Enabled = !r.Enabled
		store.UpdateRecurring(r.ID, r)
	})
	toggle.Icon = theme.MediaPauseIcon()
	del := fyne.NewMenuItem("Delete", func() {
		confirmDelete("recurring “"+orDash(r.Payee)+"”", func() { store.DeleteRecurring(r.ID) })
	})
	del.Icon = theme.DeleteIcon()
	return []*fyne.MenuItem{postNow, edit, toggle, fyne.NewMenuItemSeparator(), del}
}

// postRecurringNow opens the early-post review: create a new transaction now, or
// link this occurrence to an existing transaction. Either way the schedule
// advances one period.
func postRecurringNow(r Recurring) {
	if win == nil {
		return
	}
	cands, _ := store.RecurringCandidates(r.AcctA, r.Amount, todaySerial)
	next := fmtSerialDate(core.NextDate(r.NextDue, r.Freq))

	// Default to creating, since you're posting early; candidates + Browse-all are
	// there for "actually I already recorded it".
	sel := candidateSelect(postNewTodayOption, cands, r.AcctA, false)

	body := container.NewVBox(
		txt(orDash(r.Payee)+"   ·   "+amountLabel(r.Kind, r.Amount), colText, 14, true),
		txt("Posting early. The next occurrence will move to "+next+".", colTextDim, 11.5, false),
		spacerH(10),
		txt("Record this as:", colTextDim, 11.5, false),
		sel,
	)

	var d *dialog.CustomDialog
	cancel := secondaryButton("Cancel", nil, func() { d.Hide() })
	confirm := primaryButton("Confirm", theme.ConfirmIcon(), func() {
		create := sel.Selected == postNewTodayOption
		d.Hide()
		if create {
			if store.PostRecurringNow(r.ID, todaySerial) {
				showInfo("Posted", orDash(r.Payee)+" posted. Next due "+next+".")
			}
			return
		}
		if store.AdvanceRecurring(r.ID) {
			showInfo("Linked", orDash(r.Payee)+" marked as already recorded. Next due "+next+".")
		}
	})

	content := container.NewVBox(body, spacerH(16),
		container.NewHBox(layout.NewSpacer(), cancel, confirm))
	d = dialog.NewCustomWithoutButtons("Post now (early)", content, win)
	d.Resize(fyne.NewSize(520, 300))
	d.Show()
}

const postNewTodayOption = "➕ Create a new transaction (today)"

// candidateLabel describes an existing transaction offered for linking, including
// its amount on the account (which may differ from the template's).
func candidateLabel(c Transaction, accountID string) string {
	amt := acctAmount(c, accountID)
	if amt < 0 {
		amt = -amt
	}
	return "🔗 " + fmtSerialDate(c.Date) + " · " + orDash(c.Payee) + " · " + fmtMoney(amt)
}

func acctAmount(c Transaction, accountID string) int {
	sum := 0
	for _, p := range c.Posts {
		if p.AccountID == accountID {
			sum += p.Amount
		}
	}
	return sum
}

func acctTouches(c Transaction, accountID string) bool {
	for _, p := range c.Posts {
		if p.AccountID == accountID {
			return true
		}
	}
	return false
}

const browseAllOption = "🔍 Browse all transactions…"

// candidateSelect builds the "record this occurrence as" dropdown: the post-new
// option, the suggested candidates, and a Browse-all entry that opens a searchable
// picker of every transaction on the account (so a match outside the suggestions
// can still be linked).
func candidateSelect(postNew string, candidates []Transaction, accountID string, preselect bool) *widget.Select {
	opts := []string{postNew}
	for _, c := range candidates {
		opts = append(opts, candidateLabel(c, accountID))
	}
	opts = append(opts, browseAllOption)

	sel := widget.NewSelect(opts, nil)
	if preselect && len(candidates) > 0 {
		sel.SetSelected(opts[1])
	} else {
		sel.SetSelected(postNew)
	}
	last := sel.Selected
	sel.OnChanged = func(v string) {
		if v != browseAllOption {
			last = v
			return
		}
		showTxnPicker(accountID, func(picked *Transaction) {
			if picked == nil {
				sel.SetSelected(last) // cancelled → restore previous choice
				return
			}
			label := candidateLabel(*picked, accountID)
			if !optContains(sel.Options, label) {
				n := len(sel.Options)
				merged := append([]string{}, sel.Options[:n-1]...) // all but Browse
				sel.Options = append(merged, label, browseAllOption)
				sel.Refresh()
			}
			last = label
			sel.SetSelected(label)
		})
	}
	return sel
}

func optContains(opts []string, v string) bool {
	for _, o := range opts {
		if o == v {
			return true
		}
	}
	return false
}

// showTxnPicker is a searchable list of every transaction touching accountID;
// onPick receives the chosen transaction, or nil if cancelled.
func showTxnPicker(accountID string, onPick func(*Transaction)) {
	if win == nil {
		onPick(nil)
		return
	}
	var all []Transaction
	for _, t := range store.Transactions() { // newest first
		if acctTouches(t, accountID) {
			all = append(all, t)
		}
	}

	search := widget.NewEntry()
	search.SetPlaceHolder("Search payee…")
	listBox := container.NewVBox()

	var d *dialog.CustomDialog
	render := func(q string) {
		listBox.Objects = nil
		ql := strings.ToLower(strings.TrimSpace(q))
		shown := 0
		for _, t := range all {
			if ql != "" && !strings.Contains(strings.ToLower(t.Payee), ql) {
				continue
			}
			tt := t
			amt := acctAmount(tt, accountID)
			if amt < 0 {
				amt = -amt
			}
			line := container.NewBorder(nil, nil,
				txt(fmtSerialDate(tt.Date)+"    "+orDash(tt.Payee), colText, 12.5, false),
				mono(fmtMoney(amt), colTextDim, 12, false), nil)
			row := newTappableRow(container.New(padCell(6, 10), line), colSurface, func() {
				d.Hide()
				onPick(&tt)
			})
			row.SetBordered()
			listBox.Add(row)
			if shown++; shown >= 300 {
				break
			}
		}
		if shown == 0 {
			listBox.Add(container.New(padCell(12, 8), txt("No matching transactions", colTextDim, 12, false)))
		}
		listBox.Refresh()
	}
	search.OnChanged = render
	render("")

	cancel := secondaryButton("Cancel", nil, func() { d.Hide(); onPick(nil) })
	body := container.NewBorder(
		container.NewVBox(search, spacerH(4), divider()),
		container.NewVBox(spacerH(6), container.NewHBox(layout.NewSpacer(), cancel)),
		nil, nil,
		container.NewScroll(listBox),
	)
	d = dialog.NewCustomWithoutButtons("Link to a transaction", body, win)
	d.Resize(fyne.NewSize(560, 520))
	d.Show()
}

// RecurringForm is the add/edit dialog for a template (mirrors the transaction
// form, with a frequency and a next-due date instead of a single date).
func RecurringForm(existing *Recurring) {
	kind := widget.NewSelect([]string{"Expense", "Income", "Transfer"}, nil)
	amount := newAmountEntry()
	amount.Validator = amountValidator
	payee := widget.NewEntry()
	memo := widget.NewEntry()
	freq := widget.NewSelect(frequencies(), nil)
	next := newDateEntry(todaySerial)

	moneyNames := namesOf(store.MoneyAccounts())
	assetNames := namesOf(store.AssetAccounts())
	expenseNames := namesOf(store.ExpenseAccounts())
	incomeNames := namesOf(store.IncomeAccounts())
	acctA := widget.NewSelect(nil, nil)
	acctB := widget.NewSelect(nil, nil)
	configure := func(k string) {
		switch k {
		case "Income":
			acctA.Options, acctB.Options = assetNames, incomeNames
		case "Transfer":
			acctA.Options, acctB.Options = moneyNames, moneyNames
		default:
			acctA.Options, acctB.Options = moneyNames, expenseNames
		}
		acctA.Refresh()
		acctB.Refresh()
	}
	kind.OnChanged = configure

	title := "Add Recurring"
	if existing != nil {
		title = "Edit Recurring"
		kind.SetSelected(existing.Kind)
		configure(existing.Kind)
		acctA.SetSelected(nameOf(existing.AcctA))
		acctB.SetSelected(nameOf(existing.AcctB))
		amount.SetText(fmtMoneyInput(existing.Amount))
		payee.SetText(existing.Payee)
		memo.SetText(existing.Memo)
		freq.SetSelected(existing.Freq)
		next.SetText(fmtSerialDate(existing.NextDue))
	} else {
		kind.SetSelected("Expense")
		configure("Expense")
		freq.SetSelected("Monthly")
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Type", kind),
		widget.NewFormItem("Money account", acctA),
		widget.NewFormItem("Category / To", acctB),
		widget.NewFormItem("Amount", amount),
		widget.NewFormItem("Frequency", freq),
		widget.NewFormItem("Next due", next),
		widget.NewFormItem("Payee", payee),
		widget.NewFormItem("Memo", memo),
	}

	showForm(title, items, func() {
		amt := parseAmount(amount.Text)
		due := parseDateSerial(next.Text)
		if amt <= 0 || acctA.Selected == "" || acctB.Selected == "" || due == 0 {
			return
		}
		if kind.Selected == "Transfer" && acctA.Selected == acctB.Selected {
			return
		}
		r := Recurring{
			Kind: kind.Selected, AcctA: idOf(acctA.Selected), AcctB: idOf(acctB.Selected),
			Amount: amt, Payee: payee.Text, Memo: memo.Text,
			Freq: freq.Selected, NextDue: due, Enabled: true,
		}
		if existing != nil {
			r.Enabled = existing.Enabled
			store.UpdateRecurring(existing.ID, r)
		} else {
			store.AddRecurring(r)
		}
	})
}

// postDueNow is the "Post Due Now" / "Review & Post" action.
func postDueNow() {
	items := store.PendingRecurring(todaySerial)
	if len(items) == 0 {
		showInfo("Nothing due", "No recurring transactions are due right now.")
		return
	}
	showDuePrompt(items)
}

// CheckRecurringDue is called after a document opens; it prompts if anything's due.
func CheckRecurringDue() {
	if store == nil || win == nil {
		return
	}
	if items := store.PendingRecurring(todaySerial); len(items) > 0 {
		showDuePrompt(items)
	}
}

const postNewOption = "➕ Post a new transaction"

// showDuePrompt lists each due occurrence with a dropdown to either post a new
// transaction or link it to the existing transaction that already paid it.
func showDuePrompt(items []DueItem) {
	selects := make([]*widget.Select, len(items))
	rows := make([]fyne.CanvasObject, 0, len(items))
	for i, it := range items {
		amtCol := colNegative
		switch it.Kind {
		case "Income":
			amtCol = colPositive
		case "Transfer":
			amtCol = colText
		}

		sel := candidateSelect(postNewOption, it.Candidates, it.AcctA, it.AutoMatch)
		selects[i] = sel

		head := container.NewBorder(nil, nil,
			txt(fmtSerialDate(it.Date)+"    "+orDash(it.Payee), colText, 12.5, true),
			txt(amountLabel(it.Kind, it.Amount), amtCol, 12.5, true), nil)
		rows = append(rows, container.NewVBox(head, sel, spacerH(8)))
	}

	body := container.NewBorder(
		container.NewVBox(
			txt(itoa(len(items))+" recurring entr"+plural(len(items), "y", "ies")+" due", colText, 15, true),
			txt("For each one, post a new transaction or link it to the matching entry already in your books.", colTextDim, 11.5, false),
			spacerH(6), divider(),
		),
		nil, nil, nil,
		container.NewScroll(container.NewVBox(rows...)),
	)

	var d *dialog.CustomDialog
	later := secondaryButton("Later", nil, func() { d.Hide() })
	apply := primaryButton("Apply", theme.ConfirmIcon(), func() {
		var toPost []DueItem
		linked := 0
		for i, sel := range selects {
			if sel.Selected == postNewOption {
				toPost = append(toPost, items[i])
			} else {
				linked++ // already paid by an existing transaction → don't post
			}
		}
		d.Hide()
		n, err := store.PostRecurring(todaySerial, toPost)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		msg := "Posted " + itoa(n) + " new transaction" + plural(n, "", "s") + "."
		if linked > 0 {
			msg += "\nLinked " + itoa(linked) + " to existing entr" + plural(linked, "y", "ies") + "."
		}
		showInfo("Done", msg)
	})

	content := container.NewBorder(nil,
		container.NewVBox(spacerH(8), container.NewHBox(layout.NewSpacer(), later, apply)),
		nil, nil, body)
	d = dialog.NewCustomWithoutButtons("Recurring due", content, win)
	d.Resize(fyne.NewSize(580, 500))
	d.Show()
}

// ---- helpers ----

func amountValidator(s string) error {
	if parseAmount(s) <= 0 {
		return errAmount
	}
	return nil
}

func amountLabel(kind string, amt int) string {
	switch kind {
	case "Income":
		return "+" + fmtMoney(amt)
	case "Transfer":
		return fmtMoney(amt)
	default:
		return "−" + fmtMoney(amt)
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
