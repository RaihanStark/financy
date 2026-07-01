package mobileui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/raihanstark/financy/internal/core"
)

// The pay-installment flow mirrors the desktop: rather than always posting a new
// payment, you can link the installment to a transaction already in your books
// (paid manually, imported, or a slightly different amount) so the outstanding
// balance stays correct instead of double-posting.

const (
	postNewTodayOption = "➕ Create a new transaction (today)"
	browseAllOption    = "🔍 Browse all transactions…"
)

func orDashText(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func acctTouches(t core.Transaction, accountID string) bool {
	for _, p := range t.Posts {
		if p.AccountID == accountID {
			return true
		}
	}
	return false
}

func acctAmount(t core.Transaction, accountID string) int {
	sum := 0
	for _, p := range t.Posts {
		if p.AccountID == accountID {
			sum += p.Amount
		}
	}
	return sum
}

// candidateLabel describes an existing transaction offered for linking, showing
// its amount on the pay-from account (which may differ from the installment's).
func candidateLabel(c core.Transaction, accountID string) string {
	amt := acctAmount(c, accountID)
	if amt < 0 {
		amt = -amt
	}
	return "🔗 " + core.FmtSerialDate(c.Date) + " · " + orDashText(c.Payee) + " · " + core.FmtMoney(amt)
}

// payInstallment shows the "record this payment as" page: post a new payment now,
// or link an existing transaction (with a confident match pre-selected). The same
// dropdown UX as the desktop, including a Browse-all searchable picker.
func (m *mobileApp) payInstallment(d core.Debt, in core.Installment) {
	cands, auto := m.store.RecurringCandidates(d.AcctMoney, in.Amount, core.TodaySerial)

	known := make(map[string]core.Transaction) // option label → its transaction
	opts := []string{postNewTodayOption}
	for _, c := range cands {
		label := candidateLabel(c, d.AcctMoney)
		opts = append(opts, label)
		known[label] = c
	}
	opts = append(opts, browseAllOption)

	sel := newSelect(opts, nil)
	// Default to a new payment; pre-select a confident match so an already-recorded
	// installment links in one tap.
	if auto && len(cands) > 0 {
		sel.SetSelected(opts[1])
	} else {
		sel.SetSelected(postNewTodayOption)
	}
	last := sel.Selected
	sel.OnChanged = func(v string) {
		if v != browseAllOption {
			last = v
			return
		}
		sel.SetSelected(last) // revert immediately; the picker sets it if a txn is chosen
		m.txnPickerPage(d.AcctMoney, func(picked core.Transaction) {
			label := candidateLabel(picked, d.AcctMoney)
			if _, ok := known[label]; !ok {
				known[label] = picked
				n := len(sel.Options)
				merged := append([]string{}, sel.Options[:n-1]...) // all but Browse
				sel.Options = append(merged, label, browseAllOption)
				sel.Refresh()
			}
			last = label
			sel.SetSelected(label)
		})
	}

	m.pushPage(func(close func()) fyne.CanvasObject {
		confirm := func() {
			v := sel.Selected
			close()
			if t, ok := known[v]; ok {
				m.store.LinkInstallment(in.ID, t.ID)
			} else {
				m.store.PayInstallment(in.ID, core.TodaySerial)
			}
			m.rebuildDebt(d.ID)
		}

		info := container.NewStack(rounded(colCard, 16), insets(container.NewVBox(
			newText(d.Name+"  ·  installment "+strconv.Itoa(in.Seq), colInk, 14, true),
			gap(2),
			newText(core.FmtMoney(in.Amount), colInkDim, 13, false),
		), 10, 10, 14, 14))

		body := container.NewVBox(
			info,
			gap(14),
			wrapText("Post a new payment now, or link one already in your books."),
			gap(8),
			field("Record this payment as", sel),
		)
		return m.formPage("Pay installment", "Confirm", body, confirm, close)
	})
}

// txnPickerPage is a searchable list of every transaction touching accountID;
// onPick receives the chosen transaction (nothing happens if the user backs out).
func (m *mobileApp) txnPickerPage(accountID string, onPick func(core.Transaction)) {
	var all []core.Transaction
	for _, t := range m.store.Transactions() { // newest first
		if acctTouches(t, accountID) {
			all = append(all, t)
		}
	}

	m.pushPage(func(close func()) fyne.CanvasObject {
		search := newEntry()
		search.SetPlaceHolder("Search payee…")

		listBox := container.NewVBox()
		render := func(q string) {
			listBox.Objects = nil
			ql := strings.ToLower(strings.TrimSpace(q))
			shown := 0
			for _, t := range all {
				t := t
				if ql != "" && !strings.Contains(strings.ToLower(t.Payee), ql) {
					continue
				}
				amt := acctAmount(t, accountID)
				if amt < 0 {
					amt = -amt
				}
				label := core.FmtSerialDate(t.Date) + "  ·  " + orDashText(t.Payee) + "  ·  " + core.FmtMoney(amt)
				if shown > 0 {
					listBox.Add(rowDivider())
				}
				listBox.Add(newTappableCard(insets(newText(ellipsize(label, 40), colInk, 13, false), 10, 10, 12, 12),
					func() { close(); onPick(t) }))
				shown++
				if shown >= 100 {
					break
				}
			}
			if shown == 0 {
				listBox.Add(insets(newText("No matching transactions", colInkDim, 13, false), 12, 12, 12, 12))
			}
			listBox.Refresh()
		}
		search.OnChanged = render
		render("")

		card := container.NewStack(rounded(colCard, 16), insets(listBox, 4, 4, 6, 6))
		body := container.NewVBox(field("Search", search), gap(4), card)
		return m.formPage("Link transaction", "", body, nil, close)
	})
}
