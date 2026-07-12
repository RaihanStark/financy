package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/ui/view"
)

// sidebarWidth is the fixed width of the navigation rail.
const sidebarWidth float32 = 224

// ---- sidebar item ----

// sideItem is a full-width sidebar row: icon + label with a hover highlight and
// a primary-tinted pill when it is the active screen.
type sideItem struct {
	widget.BaseWidget
	icon   fyne.Resource
	label  string
	uid    string // nav target, "" for non-nav actions
	action func()
	bg     *canvas.Rectangle
	lbl    *canvas.Text
	// Both icon tints are built up front and toggled via Show/Hide — calling
	// SetResource on a live widget.Icon re-runs the SVG decode, which is the
	// slow path we avoid on every nav change.
	ic       *widget.Icon
	icActive *widget.Icon
	active   bool
}

// navItems collects the screen-switching sidebar items so the active one can be
// highlighted as the user moves between screens.
var navItems []*sideItem

func newSideItem(icon fyne.Resource, label string, action func()) *sideItem {
	b := &sideItem{icon: icon, label: label, action: action}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 8
	b.bg.SetMinSize(fyne.NewSize(0, 34))
	b.lbl = txt(label, colText, 13, false)
	b.ic = widget.NewIcon(icon)
	b.icActive = widget.NewIcon(theme.NewPrimaryThemedResource(icon))
	b.icActive.Hide()
	b.ExtendBaseWidget(b)
	return b
}

// newSideNav is a sidebar item that switches to a screen and shows an active
// state. Registered in navItems so highlightNav can manage the selection.
func newSideNav(icon fyne.Resource, label, uid string, action func()) *sideItem {
	b := newSideItem(icon, label, action)
	b.uid = uid
	navItems = append(navItems, b)
	return b
}

func (b *sideItem) CreateRenderer() fyne.WidgetRenderer {
	icCell := container.NewGridWrap(fyne.NewSize(18, 18), container.NewStack(b.ic, b.icActive))
	row := container.NewHBox(spacerW(6), container.NewCenter(icCell), container.NewCenter(b.lbl))
	return widget.NewSimpleRenderer(container.NewStack(b.bg, row))
}

// restFill is the item's background when not hovered — tinted if it's the
// active screen, otherwise transparent.
func (b *sideItem) restFill() color.Color {
	if b.active {
		return withAlpha(colPrimary, 0x1f)
	}
	return color.Transparent
}

func (b *sideItem) setActive(on bool) {
	if b.active == on {
		return
	}
	b.active = on
	b.bg.FillColor = b.restFill()
	if on {
		b.lbl.Color = colPrimary
		b.lbl.TextStyle = fyne.TextStyle{Bold: true}
		b.ic.Hide()
		b.icActive.Show()
	} else {
		b.lbl.Color = colText
		b.lbl.TextStyle = fyne.TextStyle{}
		b.icActive.Hide()
		b.ic.Show()
	}
	b.bg.Refresh()
	b.lbl.Refresh()
}

func (b *sideItem) Tapped(_ *fyne.PointEvent) {
	if b.action != nil {
		b.action()
	}
}

func (b *sideItem) MouseIn(_ *desktop.MouseEvent) {
	if !b.active {
		b.bg.FillColor = colSurfaceHi
		b.bg.Refresh()
	}
}

func (b *sideItem) MouseMoved(_ *desktop.MouseEvent) {}

func (b *sideItem) MouseOut() {
	b.bg.FillColor = b.restFill()
	b.bg.Refresh()
}

func (b *sideItem) Cursor() desktop.Cursor { return desktop.PointerCursor }

// highlightNav marks the sidebar item for uid as active and clears the rest.
func highlightNav(uid string) {
	for _, b := range navItems {
		b.setActive(b.uid == uid)
	}
}

// ---- sidebar assembly ----

// buildSidebar assembles the left navigation rail: brand, the New Transaction
// primary action, labeled nav items, utilities and a live net-worth footer.
func buildSidebar(c *appController) fyne.CanvasObject {
	navItems = nil

	brand := buildBrand()
	newBtn := primaryButton("New Transaction", theme.ContentAddIcon(), func() { quickAdd(c) })

	nav := container.NewVBox(
		newSideNav(iconAccounts(), "Accounts", "accounts", func() { c.show("accounts") }),
		newSideNav(iconTransactions(), "Transactions", "transactions", func() { c.show("transactions") }),
		newSideNav(iconBudget(), "Budget", "budget", func() { c.show("budget") }),
		newSideNav(iconRecurring(), "Recurring", "recurring", func() { c.show("recurring") }),
		newSideNav(iconDebts(), "Debts", "debts", func() { c.show("debts") }),
		newSideNav(iconAnalytics(), "Analytics", "analytics", func() { c.show("analytics") }),
	)

	utilities := container.NewVBox(
		newSideItem(theme.ListIcon(), "Categories", func() {
			if store != nil {
				view.OpenConfig(1)
			}
		}),
		newSideItem(theme.SettingsIcon(), "Preferences", func() {
			if store != nil {
				view.OpenConfig(0)
			}
		}),
		newSideItem(theme.ColorPaletteIcon(), themeToggleLabel(), toggleTheme),
	)

	top := container.NewVBox(
		spacerH(14),
		brand,
		spacerH(14),
		container.New(padCell(0, 10), newBtn),
		spacerH(10),
	)
	bottom := container.NewVBox(
		sideRule(),
		spacerH(8),
		container.New(padCell(0, 6), utilities),
		spacerH(10),
		container.New(padCell(0, 10), buildNetWorthFooter(c)),
		spacerH(12),
	)
	mid := container.NewVScroll(container.New(padCell(0, 6), nav))

	inner := container.NewBorder(top, bottom, nil, nil, mid)

	bg := canvas.NewRectangle(colSurface)
	width := canvas.NewRectangle(color.Transparent)
	width.SetMinSize(fyne.NewSize(sidebarWidth, 0))
	rightLine := canvas.NewRectangle(colBorder)
	rightLine.SetMinSize(fyne.NewSize(1, 0))

	side := container.NewStack(bg, width, inner)
	return container.NewBorder(nil, nil, nil, rightLine, side)
}

// buildBrand is the logo mark + wordmark at the top of the sidebar.
func buildBrand() fyne.CanvasObject {
	mark := canvas.NewRectangle(colPrimary)
	mark.CornerRadius = 8
	letter := txt("F", color.White, 14, true)
	markCell := container.NewGridWrap(fyne.NewSize(26, 26),
		container.NewStack(mark, container.NewCenter(letter)))
	name := txt("Financy", colText, 16.5, true)
	return container.NewHBox(spacerW(10), container.NewCenter(markCell), container.NewCenter(name))
}

// sideRule is a hairline separator inset from the sidebar edges.
func sideRule() fyne.CanvasObject {
	line := canvas.NewRectangle(colBorder)
	line.SetMinSize(fyne.NewSize(0, 1))
	return container.New(padCell(0, 10), line)
}

// buildNetWorthFooter is the live summary tile pinned to the bottom of the
// sidebar; updateStatus keeps its figures current.
func buildNetWorthFooter(c *appController) fyne.CanvasObject {
	c.sbCaption = txt("NET WORTH", colTextDim, 10, true)
	c.sbNet = txt("—", colText, 16, true)
	c.sbAssets = txt("", colPositive, 10.5, false)
	c.sbAssets.Alignment = fyne.TextAlignTrailing
	c.sbLiab = txt("", colNegative, 10.5, false)
	c.sbLiab.Alignment = fyne.TextAlignTrailing

	assetsRow := container.NewBorder(nil, nil, txt("Assets", colTextDim, 10.5, false), nil, c.sbAssets)
	liabRow := container.NewBorder(nil, nil, txt("Liabilities", colTextDim, 10.5, false), nil, c.sbLiab)

	bg := canvas.NewRectangle(colSurfaceHi)
	bg.CornerRadius = 10
	box := container.NewVBox(c.sbCaption, spacerH(2), c.sbNet, spacerH(8), assetsRow, spacerH(2), liabRow)
	return container.NewStack(bg, container.New(padCell(10, 12), box))
}
