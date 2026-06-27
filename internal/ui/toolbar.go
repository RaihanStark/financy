package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/ui/component"
	"github.com/raihanstark/financy/internal/ui/view"
)

// ---- tooltip layer ----
//
// Fyne 2.7 has no built-in tooltips, so we keep a tiny floating bubble in a
// no-layout layer at the root of the window and position it under whatever
// toolbar button is being hovered.

var (
	tooltipLayer *fyne.Container
	ttBox        *fyne.Container
	ttText       *canvas.Text
)

func initTooltipLayer() {
	ttText = canvas.NewText("", color.NRGBA{R: 0xf2, G: 0xf4, B: 0xf6, A: 0xff})
	ttText.TextSize = 11.5
	bg := canvas.NewRectangle(color.NRGBA{R: 0x2a, G: 0x2f, B: 0x35, A: 0xff})
	bg.CornerRadius = 4
	ttBox = container.NewStack(bg, container.New(padCell(3, 8), ttText))
	ttBox.Hide()
	tooltipLayer = container.NewWithoutLayout(ttBox)

	// Let interactive components (charts) drive the same floating tooltip.
	component.ShowTooltip = showToolTip
	component.HideTooltip = hideToolTip
}

func showToolTip(text string, pos fyne.Position) {
	if ttBox == nil {
		return
	}
	ttText.Text = text
	ttText.Refresh()
	sz := ttBox.MinSize()
	ttBox.Resize(sz)

	// Keep the bubble inside the window — important for far-right buttons.
	// pos is in canvas-absolute coordinates (from AbsolutePositionForObject or a
	// mouse event's AbsolutePosition), so clamp against the canvas here.
	if ctl != nil && ctl.win != nil {
		const margin = 6
		cs := ctl.win.Canvas().Size()
		if pos.X+sz.Width > cs.Width-margin {
			pos.X = cs.Width - sz.Width - margin
		}
		if pos.X < margin {
			pos.X = margin
		}
		if pos.Y+sz.Height > cs.Height-margin {
			pos.Y = cs.Height - sz.Height - margin
		}
		if pos.Y < margin {
			pos.Y = margin
		}
	}

	// ttBox.Move is relative to the tooltip layer, which can sit below the canvas
	// origin (e.g. the main menu bar pushes the content down). Convert the
	// canvas-absolute position into layer-local coords so the bubble lands exactly
	// at the cursor rather than menu-height too low.
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(tooltipLayer)
	ttBox.Move(pos.Subtract(origin))
	ttBox.Show()
	ttBox.Refresh()
}

func hideToolTip() {
	if ttBox != nil {
		ttBox.Hide()
	}
}

// ---- toolbar button ----

type toolBtn struct {
	widget.BaseWidget
	icon   fyne.Resource
	label  string
	uid    string // nav target, "" for non-nav actions
	action func()
	bg     *canvas.Rectangle
	active bool
}

// navButtons collects the screen-switching toolbar buttons so the active one
// can be highlighted as the user moves between screens.
var navButtons []*toolBtn

func newToolBtn(icon fyne.Resource, label string, action func()) *toolBtn {
	b := &toolBtn{icon: icon, label: label, action: action}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 6
	b.ExtendBaseWidget(b)
	return b
}

// newNavBtn is a toolbar button that switches to a screen and can show an active
// state. Registered in navButtons so highlightNav can manage the selection.
func newNavBtn(icon fyne.Resource, label, uid string, action func()) *toolBtn {
	b := newToolBtn(icon, label, action)
	b.uid = uid
	navButtons = append(navButtons, b)
	return b
}

func (b *toolBtn) CreateRenderer() fyne.WidgetRenderer {
	ic := widget.NewIcon(b.icon)
	content := container.NewGridWrap(fyne.NewSize(40, 34),
		container.NewStack(b.bg, container.NewPadded(ic)))
	return widget.NewSimpleRenderer(content)
}

// restFill is the button's color when not hovered — tinted if it's the active
// screen, otherwise transparent.
func (b *toolBtn) restFill() color.Color {
	if b.active {
		return withAlpha(colPrimary, 0x24)
	}
	return color.Transparent
}

func (b *toolBtn) setActive(on bool) {
	if b.active == on {
		return
	}
	b.active = on
	b.bg.FillColor = b.restFill()
	b.bg.Refresh()
}

func (b *toolBtn) Tapped(_ *fyne.PointEvent) {
	hideToolTip()
	if b.action != nil {
		b.action()
	}
}

func (b *toolBtn) MouseIn(_ *desktop.MouseEvent) {
	b.bg.FillColor = withAlpha(colPrimary, 0x33)
	b.bg.Refresh()
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(b)
	pos.Y += b.Size().Height + 3
	showToolTip(b.label, pos)
}

func (b *toolBtn) MouseMoved(_ *desktop.MouseEvent) {}

func (b *toolBtn) MouseOut() {
	b.bg.FillColor = b.restFill()
	b.bg.Refresh()
	hideToolTip()
}

func (b *toolBtn) Cursor() desktop.Cursor { return desktop.PointerCursor }

// highlightNav marks the toolbar button for uid as active and clears the rest.
func highlightNav(uid string) {
	for _, b := range navButtons {
		b.setActive(b.uid == uid)
	}
}

// toolSep is a thin vertical divider between toolbar groups.
func toolSep() fyne.CanvasObject {
	line := canvas.NewRectangle(colBorder)
	line.SetMinSize(fyne.NewSize(1, 18))
	return container.NewPadded(line)
}

// buildToolbar assembles the custom toolbar with hover + tooltips. The
// screen-switching buttons highlight to show which screen is active.
func buildToolbar(c *appController) fyne.CanvasObject {
	navButtons = nil
	left := container.NewHBox(
		newToolBtn(theme.ContentAddIcon(), "Add (new entry)", func() { quickAdd(c) }),
		toolSep(),
		newNavBtn(theme.StorageIcon(), "Accounts", "accounts", func() { c.show("accounts") }),
		newNavBtn(theme.HistoryIcon(), "Transactions", "transactions", func() { c.show("transactions") }),
		newNavBtn(theme.AccountIcon(), "Budget", "budget", func() { c.show("budget") }),
		newNavBtn(theme.MediaReplayIcon(), "Recurring", "recurring", func() { c.show("recurring") }),
		newNavBtn(iconDebts(), "Debts", "debts", func() { c.show("debts") }),
		toolSep(),
		newNavBtn(theme.GridIcon(), "Analytics", "analytics", func() { c.show("analytics") }),
		newNavBtn(theme.DocumentIcon(), "Reports", "reports", func() { c.show("reports") }),
	)
	right := container.NewHBox(
		newToolBtn(theme.ColorPaletteIcon(), themeToggleLabel(), toggleTheme),
		toolSep(),
		newToolBtn(theme.ListIcon(), "Categories", func() {
			if store != nil {
				view.OpenConfig(1)
			}
		}),
		newToolBtn(theme.SettingsIcon(), "Preferences", func() {
			if store != nil {
				view.OpenConfig(0)
			}
		}),
	)

	bar := canvas.NewRectangle(colSurface)
	line := canvas.NewRectangle(colBorder)
	line.SetMinSize(fyne.NewSize(0, 1))
	inner := container.NewBorder(nil, nil, container.NewPadded(left), container.NewPadded(right))
	return container.NewBorder(nil, line, nil, nil, container.NewStack(bar, inner))
}
