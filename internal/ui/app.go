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
	win         fyne.Window
	content     *fyne.Container
	currentUID  string
	statusLeft  *canvas.Text
	statusRight *canvas.Text
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
		container.NewScroll(container.NewPadded(n.build())),
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

func (c *appController) updateStatus() {
	if c.statusLeft != nil {
		if store == nil {
			c.statusLeft.Text = "  No file open"
		} else {
			c.statusLeft.Text = "  " + nav[c.currentUID].label
		}
		c.statusLeft.Refresh()
	}
	if c.statusRight != nil {
		if store == nil {
			c.statusRight.Text = ""
		} else {
			c.statusRight.Text = "Assets: " + fmtMoney(store.TotalAssets()) +
				"    Liabilities: " + fmtMoney(store.TotalLiabilities()) +
				"    Net Worth: " + fmtMoney(store.NetWorth()) + "   "
		}
		c.statusRight.Refresh()
	}
}

// assembleShell builds toolbar + status bar + content and returns the root.
func assembleShell(c *appController) fyne.CanvasObject {
	c.content = container.NewStack()
	c.statusLeft = txt("  Ready", colTextDim, 11.5, false)
	c.statusRight = txt("", colText, 11.5, true)

	toolbar := buildToolbar(c)
	statusBar := buildStatusBar(c)

	contentBG := container.NewStack(canvas.NewRectangle(colBG), c.content)
	border := container.NewBorder(toolbar, statusBar, nil, nil, contentBG)

	// A floating tooltip layer sits above everything; it never intercepts input.
	initTooltipLayer()
	return container.NewStack(border, tooltipLayer)
}

func buildStatusBar(c *appController) fyne.CanvasObject {
	bar := canvas.NewRectangle(colSurfaceHi)
	line := canvas.NewRectangle(colBorder)
	line.SetMinSize(fyne.NewSize(0, 1))
	row := container.NewBorder(line, nil,
		container.NewPadded(c.statusLeft),
		container.NewPadded(c.statusRight))
	return container.NewStack(bar, row)
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
