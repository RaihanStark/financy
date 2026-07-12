package component

import (
	"image/color"

	"github.com/raihanstark/financy/internal/ui/style"
)

// Local aliases for the shared palette so component code stays concise. They are
// copies of the active palette; SyncPalette refreshes them after a theme switch.
var (
	colSurface   color.NRGBA
	colSurfaceHi color.NRGBA
	colAltRow    color.NRGBA
	colBorder    color.NRGBA
	colText      color.NRGBA
	colTextDim   color.NRGBA
	colPrimary   color.NRGBA
	colPositive  color.NRGBA
	colNegative  color.NRGBA
)

func init() { SyncPalette() }

// SyncPalette refreshes the local palette aliases from the active style palette.
// Call it after style.SetDark so freshly built widgets pick up the new colors.
func SyncPalette() {
	colSurface = style.Surface
	colSurfaceHi = style.SurfaceHi
	colAltRow = style.AltRow
	colBorder = style.Border
	colText = style.Text
	colTextDim = style.TextDim
	colPrimary = style.Primary
	colPositive = style.Positive
	colNegative = style.Negative
}

// Corner radii for the custom canvas widgets.
const (
	radius   = style.Radius
	radiusSm = style.RadiusSm
)
