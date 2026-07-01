package mobileui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// openTransaction drills into a transaction's detail page from a shell screen
// (Transactions list / Home recent). back returns to what was showing.
func (m *mobileApp) openTransaction(t core.Transaction) {
	m.pushView(m.transactionContent(t))
}

// showTransaction re-renders the detail page in place (after an edit).
func (m *mobileApp) showTransaction(t core.Transaction) {
	m.replaceView(m.transactionContent(t))
}

func (m *mobileApp) transactionContent(t core.Transaction) fyne.CanvasObject {
	d := deriveTxn(m.store, t)
	head := m.drilldownBar("Transaction")
	body := container.NewVScroll(insets(container.NewVBox(
		gap(12),
		m.txnHero(t, d),
		gap(14),
		m.txnDetails(t, d),
		gap(14),
		m.txnActions(t),
		gap(20),
	), 0, 0, 14, 14))
	return container.NewBorder(head, nil, nil, nil, container.NewStack(canvas.NewRectangle(colBg), body))
}

// txnHero is a tonal card tinted by kind, with the signed amount, payee and date.
func (m *mobileApp) txnHero(t core.Transaction, d txnDisplay) fyne.CanvasObject {
	c := accentFor(d.kind)
	inner := container.NewVBox(
		newText(upper(d.kind), colInkDim, 10, true),
		gap(2),
		newText(amountText(d), amountColor(d), 30, true),
		gap(8),
		newText(ellipsize(d.title, 24), colInk, 16, true),
		newText(core.FmtSerialDate(t.Date), colInkDim, 12, false),
	)
	return container.NewStack(rounded(withAlpha(c, 0x14), 20), insets(inner, 18, 18, 18, 18))
}

// txnDetails is a card of labelled rows describing the transaction.
func (m *mobileApp) txnDetails(t core.Transaction, d txnDisplay) fyne.CanvasObject {
	var rows []fyne.CanvasObject
	add := func(label, value string) {
		if value == "" {
			return
		}
		if len(rows) > 0 {
			rows = append(rows, thinLine())
		}
		rows = append(rows, detailRow(label, value))
	}

	add("Type", d.kind)
	if e := deriveEdit(m.store, t); e.ok {
		if e.kind == "Transfer" {
			add("From", e.money)
			add("To", e.other)
		} else {
			add("Account", e.money)
			add("Category", e.other)
		}
	} else if d.sub != "" {
		add("Detail", d.sub)
	}
	add("Date", core.FmtSerialDate(t.Date))
	add("Memo", t.Memo)

	return container.NewStack(rounded(colCard, 16), insets(container.NewVBox(rows...), 6, 6, 14, 14))
}

func detailRow(label, value string) fyne.CanvasObject {
	l := newText(label, colInkDim, 12, false)
	v := newText(ellipsize(value, 30), colInk, 14, false)
	v.Alignment = fyne.TextAlignTrailing
	return insets(container.NewBorder(nil, nil, l, nil, flexBox(v)), 9, 9, 6, 6)
}

// txnActions offers Edit and Delete.
func (m *mobileApp) txnActions(t core.Transaction) fyne.CanvasObject {
	edit := widget.NewButton("Edit", func() {
		if e := deriveEdit(m.store, t); !e.ok {
			dialog.ShowInformation("Can't edit here",
				"Opening balances and split transactions can't be edited on mobile. Delete and re-add if needed.", m.win)
			return
		}
		m.editTransaction(t, func() {
			if nt := m.store.TxnByID(t.ID); nt != nil {
				m.showTransaction(*nt) // rebuild with the edited values
			} else {
				m.back()
			}
		})
	})
	edit.Importance = widget.HighImportance

	del := widget.NewButton("Delete", func() { m.confirmDeleteTxn(t) })
	del.Importance = widget.DangerImportance

	return container.NewGridWithColumns(2, edit, del)
}

func (m *mobileApp) confirmDeleteTxn(t core.Transaction) {
	d := deriveTxn(m.store, t)
	m.pushPage(func(close func()) fyne.CanvasObject {
		delBtn := widget.NewButton("Delete", func() {
			close() // pop the confirm page
			m.store.DeleteTransaction(t.ID)
			m.back() // pop the transaction detail → its parent
		})
		delBtn.Importance = widget.DangerImportance
		cancel := widget.NewButton("Cancel", close)
		body := container.NewVBox(
			wrapText("Delete “"+d.title+"” ("+amountText(d)+")? This can't be undone."),
			gap(18),
			delBtn,
			gap(6),
			cancel,
		)
		return m.formPage("Delete transaction?", "", body, nil, close)
	})
}

func upper(s string) string {
	b := []rune(s)
	for i, r := range b {
		if r >= 'a' && r <= 'z' {
			b[i] = r - 32
		}
	}
	return string(b)
}
