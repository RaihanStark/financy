// Package style holds the application palette and Fyne theme — the single place
// that defines how the app looks.
package style

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Palette — a light, neutral "classic native enterprise" look
// (think Quicken / MS Money / back-office banking software).
var (
	BG        = color.NRGBA{R: 0xec, G: 0xec, B: 0xec, A: 0xff} // window chrome / app background
	Surface   = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff} // grids / panels / inputs
	SurfaceHi = color.NRGBA{R: 0xe6, G: 0xe8, B: 0xeb, A: 0xff} // header fill / hover / toolbar
	AltRow    = color.NRGBA{R: 0xf5, G: 0xf6, B: 0xf7, A: 0xff} // alternating grid row
	Border    = color.NRGBA{R: 0xbf, G: 0xc4, B: 0xc9, A: 0xff} // grid lines / panel borders
	Text      = color.NRGBA{R: 0x1c, G: 0x20, B: 0x24, A: 0xff}
	TextDim   = color.NRGBA{R: 0x5b, G: 0x62, B: 0x69, A: 0xff}
	Primary   = color.NRGBA{R: 0x0a, G: 0x5f, B: 0xa4, A: 0xff} // corporate blue (selection)

	Positive = color.NRGBA{R: 0x15, G: 0x73, B: 0x47, A: 0xff} // green
	Negative = color.NRGBA{R: 0xb0, G: 0x2a, B: 0x37, A: 0xff} // red
	Warning  = color.NRGBA{R: 0x9a, G: 0x66, B: 0x00, A: 0xff} // amber/brown
)

// Theme implements fyne.Theme with the palette above.
type Theme struct{}

func (Theme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return BG
	case theme.ColorNameForeground:
		return Text
	case theme.ColorNameForegroundOnPrimary:
		return color.White
	case theme.ColorNameDisabled:
		return TextDim
	case theme.ColorNamePlaceHolder:
		return TextDim
	case theme.ColorNameButton:
		return SurfaceHi
	case theme.ColorNameInputBackground:
		return Surface
	case theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return Surface
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return Primary
	case theme.ColorNameHover:
		return SurfaceHi
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return Border
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x0a, G: 0x5f, B: 0xa4, A: 0x33}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0xb0, G: 0xb4, B: 0xb8, A: 0xff}
	case theme.ColorNameError:
		return Negative
	case theme.ColorNameSuccess:
		return Positive
	case theme.ColorNameWarning:
		return Warning
	}
	return theme.DefaultTheme().Color(name, theme.VariantLight)
}

func (Theme) Font(s fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(s) }

func (Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (Theme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 6
	case theme.SizeNameText:
		return 12.5
	case theme.SizeNameHeadingText:
		return 18
	case theme.SizeNameSubHeadingText:
		return 14
	case theme.SizeNameScrollBar:
		return 12
	case theme.SizeNameScrollBarSmall:
		return 4
	}
	return theme.DefaultTheme().Size(name)
}
