package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"github.com/raihanstark/financy/internal/ui/view"
)

// appController owns the live shell: it swaps the active screen and re-renders
// the current screen whenever the store changes.
type appController struct {
	win        fyne.Window
	content    *fyne.Container
	currentUID string
	// Sidebar footer figures, kept live by updateStatus.
	sbCaption *canvas.Text
	sbNet     *canvas.Text
	sbAssets  *canvas.Text
	sbLiab    *canvas.Text
}

var ctl *appController

func (c *appController) show(uid string) {
	n, ok := nav[uid]
	if !ok || n.build == nil {
		return
	}
	c.currentUID = uid
	c.render()
}

func (c *appController) render() {
	if store == nil {
		c.renderWelcome()
		return
	}
	n := nav[c.currentUID]
	if n.build == nil {
		c.renderWelcome()
		return
	}
	highlightNav(c.currentUID)
	c.content.Objects = []fyne.CanvasObject{
		container.NewScroll(container.New(padCell(12, 16), n.build())),
	}
	c.content.Refresh()
	c.updateStatus()
}

func (c *appController) renderWelcome() {
	highlightNav("")
	c.content.Objects = []fyne.CanvasObject{welcomeScreen()}
	c.content.Refresh()
	c.updateStatus()
}

// refresh re-renders the active screen — wired to store change notifications.
func (c *appController) refresh() {
	if c.currentUID != "" {
		c.render()
	}
}

// updateStatus refreshes the sidebar's net-worth footer from the open store.
func (c *appController) updateStatus() {
	if c.sbNet == nil {
		return
	}
	if store == nil {
		c.sbNet.Text = "No file open"
		c.sbNet.Color = colTextDim
		c.sbAssets.Text = "—"
		c.sbLiab.Text = "—"
	} else {
		c.sbNet.Text = fmtMoney(store.NetWorth())
		c.sbNet.Color = moneyColor(store.NetWorth())
		c.sbAssets.Text = fmtMoney(store.TotalAssets())
		c.sbLiab.Text = fmtMoney(store.TotalLiabilities())
	}
	c.sbNet.Refresh()
	c.sbAssets.Refresh()
	c.sbLiab.Refresh()
}

// assembleShell builds sidebar + content and returns the root.
func assembleShell(c *appController) fyne.CanvasObject {
	c.content = container.NewStack()

	sidebar := buildSidebar(c)
	contentBG := container.NewStack(canvas.NewRectangle(colBG), c.content)
	border := container.NewBorder(nil, nil, sidebar, nil, contentBG)

	// A floating tooltip layer sits above everything; it never intercepts input.
	initTooltipLayer()
	return container.NewStack(border, tooltipLayer)
}

// quickAdd is the toolbar "+" — it always opens the add-transaction form so the
// button means the same thing on every screen. Per-screen adds (accounts,
// budget categories, …) live as dedicated buttons within those screens.
func quickAdd(c *appController) {
	if store == nil {
		return
	}
	view.TransactionForm("", "")
}
