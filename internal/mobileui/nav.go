package mobileui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ---- bottom navigation ----

type navItem struct {
	widget.BaseWidget
	icon  fyne.Resource
	label string
	idx   int
	tap   func(int)

	pill *canvas.Rectangle
	text *canvas.Text
}

func newNavItem(icon fyne.Resource, label string, idx int, tap func(int)) *navItem {
	n := &navItem{icon: icon, label: label, idx: idx, tap: tap}
	n.ExtendBaseWidget(n)
	return n
}

func (n *navItem) Tapped(_ *fyne.PointEvent) {
	if n.tap != nil {
		n.tap(n.idx)
	}
}

func (n *navItem) CreateRenderer() fyne.WidgetRenderer {
	n.pill = rounded(color.Transparent, 13)
	icon := widget.NewIcon(theme.NewThemedResource(n.icon))
	iconBox := container.NewStack(n.pill,
		insets(container.NewGridWrap(fyne.NewSize(20, 20), container.NewCenter(icon)), 4, 4, 18, 18))
	n.text = newText(n.label, colInkDim, 10, false)
	n.text.Alignment = fyne.TextAlignCenter
	col := container.NewVBox(container.NewCenter(iconBox), gap(1), container.NewCenter(n.text))
	return widget.NewSimpleRenderer(insets(col, 6, 4, 2, 2))
}

func (n *navItem) setActive(on bool) {
	if n.pill == nil {
		return
	}
	if on {
		n.pill.FillColor = withAlpha(colPrimary, 0x22)
		n.text.Color = colPrimary
		n.text.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		n.pill.FillColor = color.Transparent
		n.text.Color = colInkDim
		n.text.TextStyle = fyne.TextStyle{}
	}
	n.pill.Refresh()
	n.text.Refresh()
}

type navBar struct {
	items []*navItem
	bar   fyne.CanvasObject
}

func newNavBar(tap func(int)) *navBar {
	defs := []struct {
		icon  fyne.Resource
		label string
	}{
		{theme.HomeIcon(), "Home"},
		{theme.ListIcon(), "Transactions"},
		{theme.DocumentIcon(), "Budget"},
		{theme.WarningIcon(), "Debts"},
	}
	nb := &navBar{}
	cells := make([]fyne.CanvasObject, 0, len(defs))
	for i, d := range defs {
		it := newNavItem(d.icon, d.label, i, tap)
		nb.items = append(nb.items, it)
		cells = append(cells, it)
	}
	grid := container.NewGridWithColumns(len(defs), cells...)

	bg := canvas.NewRectangle(colCard)
	line := canvas.NewRectangle(colLine)
	line.SetMinSize(fyne.NewSize(0, 1))
	nb.bar = container.NewStack(bg, container.NewBorder(line, nil, nil, nil, insets(grid, 4, 8, 6, 6)))
	return nb
}

func (nb *navBar) setActive(i int) {
	for _, it := range nb.items {
		it.setActive(it.idx == i)
	}
}

// ---- floating action button ----

type fabButton struct {
	widget.BaseWidget
	tap  func()
	icon fyne.Resource
	fill color.NRGBA
	size float32
}

// newFab is the standard 58px primary add button.
func newFab(tap func()) *fabButton {
	return newFabWith(theme.ContentAddIcon(), colPrimary, 58, tap)
}

// newFabWith builds a circular action button of a given icon, colour and size —
// used both for the main FAB and the smaller speed-dial actions.
func newFabWith(icon fyne.Resource, fill color.NRGBA, size float32, tap func()) *fabButton {
	f := &fabButton{tap: tap, icon: icon, fill: fill, size: size}
	f.ExtendBaseWidget(f)
	return f
}

func (f *fabButton) Tapped(_ *fyne.PointEvent) {
	if f.tap != nil {
		f.tap()
	}
}

func (f *fabButton) CreateRenderer() fyne.WidgetRenderer {
	circle := canvas.NewCircle(f.fill)
	icon := widget.NewIcon(theme.NewInvertedThemedResource(f.icon))
	content := container.NewGridWrap(fyne.NewSize(f.size, f.size),
		container.NewStack(circle, container.NewCenter(icon)))
	return widget.NewSimpleRenderer(content)
}

// speedDialItem is one row of the expanded FAB: a tappable text pill next to a
// circular action button, right-aligned so it stacks above the main FAB.
func speedDialItem(label string, icon fyne.Resource, size float32, tap func()) fyne.CanvasObject {
	pill := newTappableCard(
		container.NewStack(rounded(colCard, 10),
			insets(newText(label, colInk, 12, true), 8, 8, 12, 12)),
		tap)
	fab := newFabWith(icon, colPrimary, size, tap)
	row := container.NewHBox(container.NewCenter(pill), container.NewCenter(fab))
	return container.NewBorder(nil, nil, nil, row)
}
