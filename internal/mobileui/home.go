package mobileui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	"github.com/raihanstark/financy/internal/core"
)

// homeScreen is the dashboard: greeting, a compact net-worth hero, an accounts
// card, and recent activity. Everything is single-column so nothing can spill
// past the screen edge.
func (m *mobileApp) homeScreen() fyne.CanvasObject {
	col := container.NewVBox(
		gap(8),
		m.homeHeader(),
		gap(6),
		m.heroCard(),
		gap(16),
		sectionHeader("Accounts", "", nil),
		gap(4),
		m.accountsCard(),
		gap(16),
		sectionHeader("Recent activity", "See all", func() { m.selectTab(1) }),
		gap(4),
		m.recentList(),
		gap(110), // clear the tab bar + FAB
	)
	return container.NewVScroll(insets(col, 4, 0, 14, 14))
}

func (m *mobileApp) homeHeader() fyne.CanvasObject {
	title := newText("Financy", colInk, 20, true)
	sub := newText("Your money at a glance", colInkDim, 11, false)
	gear := iconButton(theme.SettingsIcon(), m.openSettings)
	return container.NewBorder(nil, nil, container.NewVBox(title, sub), container.NewCenter(gear))
}

func (m *mobileApp) heroCard() fyne.CanvasObject {
	s := m.store
	inner := container.NewVBox(
		newText("NET WORTH", withAlpha(white, 0xAB), 10, true),
		gap(1),
		newText(core.FmtMoney(s.NetWorth()), white, 26, true),
		gap(14),
		container.NewGridWithColumns(2,
			heroStat("Assets", core.FmtMoney(s.TotalAssets())),
			heroStat("Liabilities", core.FmtMoney(s.TotalLiabilities())),
		),
	)
	return container.NewStack(rounded(colHero, 20), insets(inner, 16, 16, 18, 18))
}

// accountsCard is a single rounded card listing every money account as a row —
// compact and guaranteed to fit the width.
func (m *mobileApp) accountsCard() fyne.CanvasObject {
	accts := m.store.MoneyAccounts()
	if len(accts) == 0 {
		return mutedCard("No accounts yet")
	}
	rows := make([]fyne.CanvasObject, 0, len(accts)*2)
	for i, a := range accts {
		if i > 0 {
			rows = append(rows, rowDivider())
		}
		rows = append(rows, m.accountRow(a))
	}
	return listCard(rows...)
}

func (m *mobileApp) accountRow(a core.Account) fyne.CanvasObject {
	c := colPrimary
	switch a.Type {
	case core.Asset:
		c = colPos
	case core.Liability:
		c = colNeg
	}
	sub := strings.TrimSpace(a.Institution)
	if sub == "" {
		sub = string(a.Type)
	}
	name := newText(ellipsize(a.Name, 22), colInk, 14, true)
	meta := newText(sub, colInkFaint, 11, false)
	bal := newText(core.FmtMoney(m.store.DisplayBalance(a)), colInk, 14, true)
	bal.Alignment = fyne.TextAlignTrailing

	texts := flexBox(container.NewVBox(name, meta))
	dot := initialCircle(initialOf(a.Name), c, 34)
	row := container.NewBorder(nil, nil, dot, container.NewCenter(bal), insets(texts, 0, 0, 10, 8))
	return newTappableCard(insets(row, 7, 7, 8, 8), func() { m.openAccount(a) })
}

func (m *mobileApp) recentList() fyne.CanvasObject {
	txns := sortedTxns(m.store)
	if len(txns) == 0 {
		return mutedCard("No transactions yet — tap + to add one")
	}
	if len(txns) > 5 {
		txns = txns[:5]
	}
	rows := make([]fyne.CanvasObject, 0, len(txns)*2)
	for i, t := range txns {
		t := t
		if i > 0 {
			rows = append(rows, rowDivider())
		}
		rows = append(rows, newTappableCard(m.txnRow(t), func() { m.openTransaction(t) }))
	}
	return listCard(rows...)
}

// txnRow is a static (non-recycled) transaction row used on Home. It mirrors the
// recycled row in transactions.go so the two lists look identical.
func (m *mobileApp) txnRow(t core.Transaction) fyne.CanvasObject {
	d := deriveTxn(m.store, t)
	title := newText(ellipsize(d.title, 26), colInk, 14, true)
	sub := newText(ellipsize(d.sub, 26), colInkDim, 11, false)
	left := flexBox(container.NewVBox(title, sub))

	amt := newText(amountText(d), amountColor(d), 14, true)
	amt.Alignment = fyne.TextAlignTrailing
	date := newText(shortDate(t.Date), colInkFaint, 10, false)
	date.Alignment = fyne.TextAlignTrailing
	right := container.NewVBox(amt, date)

	row := container.NewBorder(nil, nil, avatar(d.kind, d.title), right, insets(left, 0, 0, 10, 10))
	return insets(row, 7, 7, 8, 8)
}
