package mobileui

import (
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// transactionsScreen lists the whole history in a recycling widget.List, so
// scrolling stays smooth no matter how many transactions there are.
func (m *mobileApp) transactionsScreen() fyne.CanvasObject {
	txns := sortedTxns(m.store)
	header := insets(screenHeader("Transactions", pluralCount(len(txns), "transaction")), 12, 8, 16, 16)

	if len(txns) == 0 {
		return container.NewBorder(header, nil, nil, nil,
			container.NewVScroll(insets(mutedCard("No transactions yet — tap + to add one"), 4, 0, 16, 16)))
	}

	store := m.store
	list := widget.NewList(
		func() int { return len(txns) },
		func() fyne.CanvasObject { return newTxnRowWidget() },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*txnRowWidget).set(store, txns[i])
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		list.Unselect(id)
		m.openTransaction(txns[id])
	}
	list.HideSeparators = true
	return container.NewBorder(header, nil, nil, nil, list)
}

func pluralCount(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// ---- recycled transaction row ----

type txnRowWidget struct {
	widget.BaseWidget
	circle *canvas.Circle
	letter *canvas.Text
	title  *canvas.Text
	sub    *canvas.Text
	amt    *canvas.Text
	date   *canvas.Text
}

func newTxnRowWidget() *txnRowWidget {
	r := &txnRowWidget{}
	r.ExtendBaseWidget(r)
	return r
}

func (r *txnRowWidget) CreateRenderer() fyne.WidgetRenderer {
	r.circle = canvas.NewCircle(color.Transparent)
	r.letter = newText("", colInk, 14, true)
	av := container.NewGridWrap(fyne.NewSize(36, 36),
		container.NewStack(r.circle, centerInitial(r.letter, 36)))

	r.title = newText("", colInk, 14, true)
	r.sub = newText("", colInkDim, 11, false)
	left := flexBox(container.NewVBox(r.title, r.sub))

	r.amt = newText("", colInk, 14, true)
	r.amt.Alignment = fyne.TextAlignTrailing
	r.date = newText("", colInkFaint, 10, false)
	r.date.Alignment = fyne.TextAlignTrailing
	right := container.NewVBox(r.amt, r.date)

	row := container.NewBorder(nil, nil, av, right, insets(left, 0, 0, 10, 10))
	return widget.NewSimpleRenderer(insets(row, 7, 7, 12, 12))
}

func (r *txnRowWidget) set(s *core.Store, t core.Transaction) {
	d := deriveTxn(s, t)
	c := accentFor(d.kind)
	r.circle.FillColor = withAlpha(c, 0x22)
	r.circle.Refresh()
	r.letter.Text = initialOf(d.title)
	r.letter.Color = c
	r.letter.Refresh()
	r.title.Text = ellipsize(d.title, 26)
	r.title.Refresh()
	r.sub.Text = ellipsize(d.sub, 26)
	r.sub.Refresh()
	r.amt.Text = amountText(d)
	r.amt.Color = amountColor(d)
	r.amt.Refresh()
	r.date.Text = shortDate(t.Date)
	r.date.Refresh()
}
