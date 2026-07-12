// Package style holds the application palette and Fyne theme — the single place
// that defines how the app looks. It ships two palettes (light and dark); the
// active one is swapped at runtime via SetDark.
package style

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Active palette. These vars are the live colors the whole app reads — custom
// canvas widgets included. SetDark mutates them in place so a theme switch is a
// matter of reassigning here and rebuilding the UI. They start on the light
// palette (see init).
var (
	BG        color.NRGBA // window chrome / app background
	Surface   color.NRGBA // cards / panels / inputs
	SurfaceHi color.NRGBA // header fill / hover / toolbar
	AltRow    color.NRGBA // alternating grid row
	Border    color.NRGBA // hairline borders / grid lines
	Text      color.NRGBA // primary text
	TextDim   color.NRGBA // muted captions
	Primary   color.NRGBA // accent

	Positive color.NRGBA // gains / credits
	Negative color.NRGBA // losses / debits
	Warning  color.NRGBA // cautions
)

// Dark reports whether the dark palette is currently active.
var Dark bool

// palette is a full set of the colors above.
type palette struct {
	BG, Surface, SurfaceHi, AltRow, Border, Text, TextDim, Primary color.NRGBA
	Positive, Negative, Warning                                    color.NRGBA
}

// Light — a clean, modern light look: soft cool-gray chrome, crisp white
// surfaces, hairline borders and a confident indigo accent. Money colors are
// emerald/rose rather than raw green/red so figures read calm, not alarming.
var light = palette{
	BG:        color.NRGBA{R: 0xf6, G: 0xf7, B: 0xf9, A: 0xff},
	Surface:   color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	SurfaceHi: color.NRGBA{R: 0xee, G: 0xf0, B: 0xf4, A: 0xff},
	AltRow:    color.NRGBA{R: 0xf8, G: 0xf9, B: 0xfb, A: 0xff},
	Border:    color.NRGBA{R: 0xe4, G: 0xe7, B: 0xec, A: 0xff},
	Text:      color.NRGBA{R: 0x11, G: 0x18, B: 0x27, A: 0xff},
	TextDim:   color.NRGBA{R: 0x6b, G: 0x72, B: 0x80, A: 0xff},
	Primary:   color.NRGBA{R: 0x4f, G: 0x46, B: 0xe5, A: 0xff}, // indigo
	Positive:  color.NRGBA{R: 0x05, G: 0x96, B: 0x69, A: 0xff}, // emerald
	Negative:  color.NRGBA{R: 0xe1, G: 0x1d, B: 0x48, A: 0xff}, // rose
	Warning:   color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}, // amber
}

// Dark — a deep indigo-tinted slate mirroring the light palette: near-black
// chrome, raised slate surfaces, hairline borders and brighter accents tuned to
// read well on a dark ground.
var dark = palette{
	BG:        color.NRGBA{R: 0x0b, G: 0x0d, B: 0x12, A: 0xff}, // window chrome
	Surface:   color.NRGBA{R: 0x15, G: 0x18, B: 0x21, A: 0xff}, // cards / panels / inputs
	SurfaceHi: color.NRGBA{R: 0x1f, G: 0x23, B: 0x30, A: 0xff}, // header / hover / sidebar
	AltRow:    color.NRGBA{R: 0x19, G: 0x1d, B: 0x28, A: 0xff}, // alternating grid row
	Border:    color.NRGBA{R: 0x2a, G: 0x2f, B: 0x3d, A: 0xff}, // hairline borders / grid lines
	Text:      color.NRGBA{R: 0xe7, G: 0xe9, B: 0xee, A: 0xff}, // primary text
	TextDim:   color.NRGBA{R: 0x9a, G: 0xa1, B: 0xb2, A: 0xff}, // muted captions
	Primary:   color.NRGBA{R: 0x81, G: 0x8c, B: 0xf8, A: 0xff}, // light indigo accent
	Positive:  color.NRGBA{R: 0x34, G: 0xd3, B: 0x99, A: 0xff}, // light emerald
	Negative:  color.NRGBA{R: 0xfb, G: 0x71, B: 0x85, A: 0xff}, // light rose
	Warning:   color.NRGBA{R: 0xfb, G: 0xbf, B: 0x24, A: 0xff}, // light amber
}

func init() { apply(light) }

// apply copies p into the active palette vars.
func apply(p palette) {
	BG, Surface, SurfaceHi, AltRow, Border = p.BG, p.Surface, p.SurfaceHi, p.AltRow, p.Border
	Text, TextDim, Primary = p.Text, p.TextDim, p.Primary
	Positive, Negative, Warning = p.Positive, p.Negative, p.Warning
}

// SetDark switches the active palette to dark (on) or light (off). Callers must
// re-sync their local palette aliases and rebuild custom-canvas content
// afterwards — the active canvas objects cache their colors at build time.
func SetDark(on bool) {
	Dark = on
	if on {
		apply(dark)
	} else {
		apply(light)
	}
}

// Corner radii used across the custom canvas widgets (cards, chips, banners).
const (
	Radius   float32 = 12 // cards / panels
	RadiusSm float32 = 7  // chips / badges / small controls
)

// Theme implements fyne.Theme with the active palette above.
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
		// A translucent overlay Fyne blends over the button's base color so it
		// tints rather than replacing it with a flat gray. Darken on light,
		// lighten on dark.
		if Dark {
			return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1f}
		}
		return color.NRGBA{R: 0, G: 0, B: 0, A: 0x1a}
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return Border
	case theme.ColorNameSelection:
		return withAlpha(Primary, 0x29)
	case theme.ColorNameScrollBar:
		if Dark {
			return color.NRGBA{R: 0x4a, G: 0x50, B: 0x5a, A: 0xff}
		}
		return color.NRGBA{R: 0xc2, G: 0xc7, B: 0xce, A: 0xff}
	case theme.ColorNameError:
		return Negative
	case theme.ColorNameSuccess:
		return Positive
	case theme.ColorNameWarning:
		return Warning
	}
	return theme.DefaultTheme().Color(name, variant())
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
		return 9
	case theme.SizeNameSelectionRadius:
		return 7
	case theme.SizeNameScrollBar:
		return 12
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameScrollBarRadius:
		return 4
	}
	return theme.DefaultTheme().Size(name)
}

// variant maps the active palette to the matching Fyne variant, used for the
// default-theme fallback so unhandled colors look right in both modes.
func variant() fyne.ThemeVariant {
	if Dark {
		return theme.VariantDark
	}
	return theme.VariantLight
}

// withAlpha returns c with its alpha replaced by a.
func withAlpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}
