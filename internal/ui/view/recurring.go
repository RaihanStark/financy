package view

import (
	"errors"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

var errAmount = errors.New("enter an amount greater than zero")

// ScreenRecurring is the forward-looking recurring view: what the schedule
// costs per month, what's coming in the next 30 days, and the templates
// themselves. Nothing here posts without confirmation.
func ScreenRecurring() fyne.CanvasObject {
	bar := appBar("Recurring", "Financy drafts each entry when due — you review and confirm before anything posts",
		primaryButton("Add Recurring", theme.ContentAddIcon(), func() { RecurringForm(nil) }),
	)

	rs := store.Recurrings()
	if len(rs) == 0 {
		return container.NewBorder(bar, nil, nil, nil, container.NewPadded(recurringWelcome()))
	}

	sum := store.RecurringSummaryFor(todaySerial)
	return container.NewBorder(bar, nil, nil, nil, container.NewPadded(container.NewVBox(
		dueBanner(sum),
		recurringSummaryRow(sum),
		spacerH(6),
		upcomingPanel(),
		spacerH(6),
		templatesPanel(rs),
	)))
}

// recurringWelcome explains the feature when no templates exist yet.
func recurringWelcome() fyne.CanvasObject {
	box := container.NewVBox(
		txt("Put your bills on autopilot — with a veto", colText, 18, true),
		spacerH(4),
		txt("Add rent, salary and subscriptions once. Financy tracks the schedule,", colTextDim, 12, false),
		txt("shows what's coming and what it all costs per month, and drafts each", colTextDim, 12, false),
		txt("entry when it's due. Nothing posts without your confirmation.", colTextDim, 12, false),
		spacerH(12),
		container.NewHBox(primaryButton("Add Recurring", theme.ContentAddIcon(), func() { RecurringForm(nil) })),
	)
	return container.NewCenter(panel(container.New(padCell(20, 28), box)))
}

// dueBanner shows a one-line nudge when something is due.
func dueBanner(sum RecurringSummary) fyne.CanvasObject {
	if sum.DueCount == 0 {
		return spacerH(0)
	}
	msg := itoa(sum.DueCount) + " recurring entr" + plural(sum.DueCount, "y", "ies") + " due · " + fmtMoney(sum.DueTotal)
	return container.NewVBox(
		alertBanner(theme.WarningIcon(), msg, colWarning,
			secondaryButton("Review & Post", theme.MediaPlayIcon(), postDueNow)),
		spacerH(6),
	)
}

// recurringSummaryRow is the commitment header: what the enabled templates add
// up to per month, normalized from each one's frequency.
func recurringSummaryRow(sum RecurringSummary) fyne.CanvasObject {
	nExp, nInc := 0, 0
	for _, r := range store.Recurrings() {
		if !r.Enabled {
			continue
		}
		switch r.Kind {
		case "Expense":
			nExp++
		case "Income":
			nInc++
		}
	}
	pausedSub := "all running"
	if sum.Paused > 0 {
		pausedSub = itoa(sum.Paused) + " paused"
	}
	return container.NewGridWithColumns(4,
		statCard("MONTHLY BILLS", fmtMoney(sum.MonthlyExpense), colNegative, itoa(nExp)+" template"+plural(nExp, "", "s")),
		statCard("MONTHLY INCOME", fmtMoney(sum.MonthlyIncome), colPositive, itoa(nInc)+" template"+plural(nInc, "", "s")),
		statCard("NET PER MONTH", fmtMoney(sum.MonthlyNet), moneyColor(sum.MonthlyNet), "income − bills"),
		statCard("ACTIVE", itoa(sum.Active), colText, pausedSub),
	)
}

// maxBucketRows caps each timeline bucket so a dense schedule stays scannable.
const maxBucketRows = 8

// upcomingPanel is the 30-day timeline: overdue and due first, then this week,
// then later this month.
func upcomingPanel() fyne.CanvasObject {
	occ := store.UpcomingRecurring(todaySerial, todaySerial+30)

	if len(occ) == 0 {
		msg := "Nothing scheduled in the next 30 days."
		if next := store.UpcomingRecurring(todaySerial, todaySerial+366); len(next) > 0 {
			msg = "Nothing scheduled in the next 30 days — next up: " +
				fmtSerialDate(next[0].Date) + " · " + orDash(next[0].Payee)
		}
		return panel(container.New(padCell(10, 12), container.NewVBox(
			sectionTitle("Next 30 days"), spacerH(6),
			txt(msg, colTextDim, 12, false))))
	}

	var due, week, later []Occurrence
	for _, o := range occ {
		switch {
		case o.Due:
			due = append(due, o)
		case o.Date <= todaySerial+7:
			week = append(week, o)
		default:
			later = append(later, o)
		}
	}

	rows := []fyne.CanvasObject{sectionTitle("Next 30 days"), spacerH(4)}
	bucket := func(label string, labelCol color.Color, items []Occurrence, action fyne.CanvasObject) {
		if len(items) == 0 {
			return
		}
		head := container.NewBorder(nil, nil, txt(label, labelCol, 10.5, true), action)
		rows = append(rows, spacerH(4), head, spacerH(2))
		for i, o := range items {
			if i == maxBucketRows {
				rows = append(rows, container.New(padCell(4, 12),
					txt("+ "+itoa(len(items)-maxBucketRows)+" more", colTextDim, 11, false)))
				break
			}
			rows = append(rows, upcomingRow(o))
		}
	}
	bucket("OVERDUE & DUE", colWarning, due,
		secondaryButton("Review & Post", theme.MediaPlayIcon(), postDueNow))
	bucket("THIS WEEK", colTextDim, week, nil)
	bucket("LATER THIS MONTH", colTextDim, later, nil)
	return panel(container.New(padCell(10, 12), container.NewVBox(rows...)))
}

// upcomingRow is one projected occurrence on the timeline. Tapping opens the
// template it comes from; right-click offers the template actions.
func upcomingRow(o Occurrence) fyne.CanvasObject {
	amtCol := colNegative
	switch o.Kind {
	case "Income":
		amtCol = colPositive
	case "Transfer":
		amtCol = colText
	}
	title := o.Payee
	if title == "" {
		title = o.Kind
	}
	state, stateCol := dueLabel(o.Date)

	left := container.NewHBox(
		mono(fmtSerialDate(o.Date), colTextDim, 11.5, false),
		spacerW(10),
		container.NewVBox(
			txt(title, colText, 13, true),
			txt(o.Kind+" · "+orDash(nameOf(o.AcctB)), colTextDim, 10.5, false),
		),
	)
	right := container.NewVBox(
		alignRight(txt(amountLabel(o.Kind, o.Amount), amtCol, 13.5, true)),
		alignRight(txt(state, stateCol, 10, o.Due)),
	)
	inner := container.NewBorder(nil, nil, left, right)

	var fill color.Color = colSurface
	if o.Due {
		fill = withAlpha(colWarning, 0x14) // faint amber tint so due rows stand out
	}
	open := func() {
		if r := store.RecurringByID(o.RecurringID); r != nil {
			RecurringForm(r)
		}
	}
	row := newTappableRow(container.New(padCell(7, 12), inner), fill, open)
	row.SetBordered()
	row.SetOnSecondary(func(pos fyne.Position) {
		if r := store.RecurringByID(o.RecurringID); r != nil {
			showContextMenu(pos, recurringMenu(*r)...)
		}
	})
	return row
}

// dueLabel words an occurrence's distance from today ("today", "in 5 days",
// "3 days overdue"), amber once it's due.
func dueLabel(date int) (string, color.Color) {
	d := date - todaySerial
	switch {
	case d < -1:
		return itoa(-d) + " days overdue", colWarning
	case d == -1:
		return "1 day overdue", colWarning
	case d == 0:
		return "today", colWarning
	case d == 1:
		return "tomorrow", colTextDim
	default:
		return "in " + itoa(d) + " days", colTextDim
	}
}

// templatesPanel lists every template for managing the schedule itself.
func templatesPanel(rs []Recurring) fyne.CanvasObject {
	rows := []fyne.CanvasObject{
		container.NewBorder(nil, nil, sectionTitle("All templates"),
			txt(itoa(len(rs))+" template"+plural(len(rs), "", "s"), colTextDim, 11, false)),
		spacerH(6),
	}
	for _, r := range rs {
		rows = append(rows, recurringRow(r))
	}
	return panel(container.New(padCell(10, 12), container.NewVBox(rows...)))
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
		sel.widget,
	)

	var d *modal
	cancel := secondaryButton("Cancel", nil, func() { d.Hide() })
	confirm := primaryButton("Confirm", theme.ConfirmIcon(), func() {
		create := sel.isPostNew()
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
	d = newModal("Post now (early)", content)
	d.SetCardSize(fyne.NewSize(520, 300))
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

// candidateSel pairs the "record this as" dropdown with a lookup from each option
// label back to the transaction it represents, so callers can both ask "post a new
// one?" and resolve which existing transaction was chosen for linking.
type candidateSel struct {
	widget  *widget.Select
	known   map[string]Transaction // option label → its transaction (excludes post-new / Browse)
	postNew string
}

// isPostNew reports whether the post-a-new-transaction option is selected.
func (c *candidateSel) isPostNew() bool { return c.widget.Selected == c.postNew }

// picked returns the existing transaction the dropdown currently points at, or nil
// when post-new (or nothing resolvable) is selected.
func (c *candidateSel) picked() *Transaction {
	if t, ok := c.known[c.widget.Selected]; ok {
		return &t
	}
	return nil
}

// candidateSelect builds the "record this occurrence as" dropdown: the post-new
// option, the suggested candidates, and a Browse-all entry that opens a searchable
// picker of every transaction on the account (so a match outside the suggestions
// can still be linked).
func candidateSelect(postNew string, candidates []Transaction, accountID string, preselect bool) *candidateSel {
	known := make(map[string]Transaction)
	opts := []string{postNew}
	for _, c := range candidates {
		label := candidateLabel(c, accountID)
		opts = append(opts, label)
		known[label] = c
	}
	opts = append(opts, browseAllOption)

	sel := widget.NewSelect(opts, nil)
	cs := &candidateSel{widget: sel, known: known, postNew: postNew}
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
			known[label] = *picked
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
	return cs
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

	var d *modal
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
	d = newModal("Link to a transaction", body)
	d.SetCardSize(fyne.NewSize(560, 520))
	d.Show()
}

// RecurringForm is the add/edit dialog for a template (mirrors the transaction
// form, with a frequency and a next-due date instead of a single date).
func RecurringForm(existing *Recurring) {
	kind := widget.NewSelect([]string{"Expense", "Income", "Transfer"}, nil)
	amount := newAmountEntry()
	amount.Validator = amountValidator
	payee := newCompletionEntry(store.Payees())
	memo := widget.NewEntry()
	freq := widget.NewSelect(frequencies(), nil)
	next := newDateEntry(todaySerial)

	moneyNames := groupedLabels(store.MoneyAccounts())
	assetNames := groupedLabels(store.AssetAccounts())
	expenseNames := groupedLabels(store.ExpenseAccounts())
	incomeNames := groupedLabels(store.IncomeAccounts())
	acctA := widget.NewSelect(nil, nil)
	acctB := widget.NewSelect(nil, nil)
	guardGroupHeaders(acctA)
	guardGroupHeaders(acctB)
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
		acctA.SetSelected(labelOf(existing.AcctA))
		acctB.SetSelected(labelOf(existing.AcctB))
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
			Amount: amt, Payee: payee.Text(), Memo: memo.Text,
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
		lastDueSig = dueSignature(items)
		showDuePrompt(items)
	}
}

// lastDueSig is the due set last shown (or deferred with Later), so periodic
// re-checks only prompt again when something new becomes due.
var lastDueSig string

func dueSignature(items []DueItem) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = it.RecurringID + ":" + itoa(it.Date)
	}
	return strings.Join(parts, ",")
}

// RecheckRecurringDue is the periodic due check (called from the app's minute
// ticker). It re-prompts only when the due set changed since the last prompt —
// so "Later" defers until something new becomes due — and never over an open
// dialog or menu.
func RecheckRecurringDue() {
	if store == nil || win == nil {
		return
	}
	if win.Canvas().Overlays().Top() != nil {
		return // don't stomp whatever the user is doing
	}
	items := store.PendingRecurring(todaySerial)
	if len(items) == 0 {
		lastDueSig = ""
		return
	}
	if sig := dueSignature(items); sig != lastDueSig {
		lastDueSig = sig
		showDuePrompt(items)
	}
}

const postNewOption = "➕ Post a new transaction"

// showDuePrompt lists each due occurrence with a dropdown to either post a new
// transaction or link it to the existing transaction that already paid it.
func showDuePrompt(items []DueItem) {
	selects := make([]*candidateSel, len(items))
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
		rows = append(rows, container.NewVBox(head, sel.widget, spacerH(8)))
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

	var d *modal
	later := secondaryButton("Later", nil, func() { d.Hide() })
	apply := primaryButton("Apply", theme.ConfirmIcon(), func() {
		var toPost []DueItem
		linked := 0
		for i, sel := range selects {
			if sel.isPostNew() {
				toPost = append(toPost, items[i])
			} else {
				linked++ // already paid by an existing transaction → don't post
			}
		}
		d.Hide()
		n, err := store.PostRecurring(todaySerial, toPost)
		if err != nil {
			showError(err)
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
	d = newModal("Recurring due", content)
	d.SetCardSize(fyne.NewSize(580, 500))
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
