// Package style holds the application palette and Fyne theme — the single place
// that defines how the app looks.
package style

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Palette — a clean, modern light look: soft off-white chrome, crisp white
// surfaces, hairline borders and a confident blue accent. Generous rounding and
// spacing (see Size) keep it airy and approachable rather than dense.
var (
	BG        = color.NRGBA{R: 0xf5, G: 0xf6, B: 0xf8, A: 0xff} // window chrome / app background
	Surface   = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff} // cards / panels / inputs
	SurfaceHi = color.NRGBA{R: 0xed, G: 0xf0, B: 0xf4, A: 0xff} // header fill / hover / toolbar
	AltRow    = color.NRGBA{R: 0xf7, G: 0xf9, B: 0xfb, A: 0xff} // alternating grid row
	Border    = color.NRGBA{R: 0xe2, G: 0xe6, B: 0xec, A: 0xff} // hairline borders / grid lines
	Text      = color.NRGBA{R: 0x16, G: 0x19, B: 0x1d, A: 0xff} // near-black slate
	TextDim   = color.NRGBA{R: 0x6b, G: 0x72, B: 0x80, A: 0xff} // muted captions
	Primary   = color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff} // modern blue accent

	Positive = color.NRGBA{R: 0x15, G: 0x80, B: 0x3d, A: 0xff} // emerald green
	Negative = color.NRGBA{R: 0xdc, G: 0x26, B: 0x26, A: 0xff} // clear red
	Warning  = color.NRGBA{R: 0xb4, G: 0x53, B: 0x09, A: 0xff} // amber
)

// Corner radii used across the custom canvas widgets (cards, chips, banners).
const (
	Radius   float32 = 10 // cards / panels
	RadiusSm float32 = 6  // chips / badges / small controls
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
		// A translucent darken overlay — Fyne blends this over the button's base
		// color, so it tints (darker blue on primary, darker gray on others)
		// instead of replacing it with a flat gray.
		return color.NRGBA{R: 0, G: 0, B: 0, A: 0x1a}
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return Border
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0x29}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0xc2, G: 0xc7, B: 0xce, A: 0xff}
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
		return 6
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameText:
		return 13
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameSubHeadingText:
		return 15
	case theme.SizeNameInputRadius:
		return 8
	case theme.SizeNameSelectionRadius:
		return 6
	case theme.SizeNameScrollBar:
		return 12
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameScrollBarRadius:
		return 4
	}
	return theme.DefaultTheme().Size(name)
}
