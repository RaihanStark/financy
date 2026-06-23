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
	}

	ttBox.Move(pos)
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
	action func()
	bg     *canvas.Rectangle
}

func newToolBtn(icon fyne.Resource, label string, action func()) *toolBtn {
	b := &toolBtn{icon: icon, label: label, action: action}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 4
	b.ExtendBaseWidget(b)
	return b
}

func (b *toolBtn) CreateRenderer() fyne.WidgetRenderer {
	ic := widget.NewIcon(b.icon)
	content := container.NewGridWrap(fyne.NewSize(38, 32),
		container.NewStack(b.bg, container.NewPadded(ic)))
	return widget.NewSimpleRenderer(content)
}

func (b *toolBtn) Tapped(_ *fyne.PointEvent) {
	hideToolTip()
	if b.action != nil {
		b.action()
	}
}

func (b *toolBtn) MouseIn(_ *desktop.MouseEvent) {
	b.bg.FillColor = withAlpha(colPrimary, 0x2a)
	b.bg.Refresh()
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(b)
	pos.Y += b.Size().Height + 3
	showToolTip(b.label, pos)
}

func (b *toolBtn) MouseMoved(_ *desktop.MouseEvent) {}

func (b *toolBtn) MouseOut() {
	b.bg.FillColor = color.Transparent
	b.bg.Refresh()
	hideToolTip()
}

func (b *toolBtn) Cursor() desktop.Cursor { return desktop.PointerCursor }

// toolSep is a thin vertical divider between toolbar groups.
func toolSep() fyne.CanvasObject {
	line := canvas.NewRectangle(colBorder)
	line.SetMinSize(fyne.NewSize(1, 18))
	return container.NewPadded(line)
}

// buildToolbar assembles the custom toolbar with hover + tooltips.
func buildToolbar(c *appController) fyne.CanvasObject {
	left := container.NewHBox(
		newToolBtn(theme.ContentAddIcon(), "Add (new entry)", func() { quickAdd(c) }),
		toolSep(),
		newToolBtn(theme.StorageIcon(), "Accounts", func() { c.show("accounts") }),
		newToolBtn(theme.HistoryIcon(), "Transactions", func() { c.show("transactions") }),
		newToolBtn(theme.GridIcon(), "Analytics", func() { c.show("analytics") }),
	)
	right := container.NewHBox(
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

	bar := canvas.NewRectangle(colSurfaceHi)
	line := canvas.NewRectangle(colBorder)
	line.SetMinSize(fyne.NewSize(0, 1))
	inner := container.NewBorder(nil, nil, container.NewPadded(left), container.NewPadded(right))
	return container.NewBorder(nil, line, nil, nil, container.NewStack(bar, inner))
}
