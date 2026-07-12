package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"github.com/raihanstark/financy/internal/ui/component"
)

// ---- tooltip layer ----
//
// Fyne 2.7 has no built-in tooltips, so we keep a tiny floating bubble in a
// no-layout layer at the root of the window and position it under whatever
// element is being hovered (charts drive it via component.ShowTooltip).

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

	// Keep the bubble inside the window — important near the edges.
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
