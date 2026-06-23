package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	"github.com/raihanstark/financy/internal/ui/style"
)

// The floating tooltip is positioned with canvas-absolute coordinates, but
// ttBox.Move is relative to the tooltip layer — which the in-canvas File menu
// pushes down. showToolTip must convert to layer-local coords so the bubble
// lands at the requested canvas position, not menu-height too low.
func TestTooltipLandsAtRequestedPosition(t *testing.T) {
	a := test.NewApp()
	a.Settings().SetTheme(style.Theme{})
	initTooltipLayer() // assembleShell does this in the real app
	win := test.NewWindow(container.NewStack(container.NewVBox(), tooltipLayer))
	win.SetMainMenu(fyne.NewMainMenu(fyne.NewMenu("File", fyne.NewMenuItem("New", nil))))
	ctl = &appController{win: win}
	win.Resize(fyne.NewSize(900, 600))
	_ = win.Canvas()

	target := fyne.NewPos(300, 200)
	showToolTip("hello", target)

	got := fyne.CurrentApp().Driver().AbsolutePositionForObject(ttBox)
	if dx, dy := got.X-target.X, got.Y-target.Y; dx*dx+dy*dy > 4 {
		t.Fatalf("tooltip landed at %v, want ~%v", got, target)
	}
}
