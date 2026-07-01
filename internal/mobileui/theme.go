package mobileui

// A clean, touch-first design system for the mobile app. This package is
// completely independent of internal/ui (the desktop shell) — it shares only
// internal/core. The look is intentionally not the desktop's: a light canvas,
// rounded white cards, a deep-indigo net-worth hero, and a bottom tab bar.

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// Palette.
var (
	colBg       = color.NRGBA{R: 0xF3, G: 0xF4, B: 0xF7, A: 0xFF}
	colCard     = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	colInk      = color.NRGBA{R: 0x11, G: 0x18, B: 0x2A, A: 0xFF}
	colInkDim   = color.NRGBA{R: 0x66, G: 0x70, B: 0x82, A: 0xFF}
	colInkFaint = color.NRGBA{R: 0x9A, G: 0xA1, B: 0xAD, A: 0xFF}
	colPrimary  = color.NRGBA{R: 0x4F, G: 0x46, B: 0xE5, A: 0xFF}
	colHero     = color.NRGBA{R: 0x27, G: 0x21, B: 0x63, A: 0xFF}
	colPos      = color.NRGBA{R: 0x0F, G: 0x9D, B: 0x58, A: 0xFF}
	colNeg      = color.NRGBA{R: 0xDC, G: 0x45, B: 0x45, A: 0xFF}
	colLine     = color.NRGBA{R: 0xE8, G: 0xEA, B: 0xEF, A: 0xFF}
	white       = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

func withAlpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

// appTheme tunes Fyne's built-in widgets (entries, buttons, progress bars,
// dialogs) so they match the palette and are comfortable on touch.
type appTheme struct{}

func (appTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return colBg
	case theme.ColorNameForeground:
		return colInk
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return colPrimary
	case theme.ColorNameButton:
		return colCard
	case theme.ColorNameInputBackground:
		// A hair darker than a card so entries/selects read as fields on a modal.
		return colBg
	case theme.ColorNamePlaceHolder, theme.ColorNameDisabled:
		return colInkFaint
	case theme.ColorNameSeparator:
		return colLine
	}
	return theme.DefaultTheme().Color(n, theme.VariantLight)
}

func (appTheme) Font(s fyne.TextStyle) fyne.Resource     { return theme.DefaultTheme().Font(s) }
func (appTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }

func (appTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameText:
		return 14
	case theme.SizeNamePadding:
		return 5
	case theme.SizeNameInnerPadding:
		return 8
	}
	return theme.DefaultTheme().Size(n)
}

// ---- primitive view helpers ----

func newText(s string, col color.Color, size float32, bold bool) *canvas.Text {
	t := canvas.NewText(s, col)
	t.TextSize = size
	t.TextStyle = fyne.TextStyle{Bold: bold}
	return t
}

// gap is a fixed vertical spacer.
func gap(h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(0, h))
	return r
}

// insets pads an object by explicit top/bottom/left/right amounts.
func insets(o fyne.CanvasObject, t, b, l, r float32) *fyne.Container {
	return container.New(layout.NewCustomPaddedLayout(t, b, l, r), o)
}

func rounded(fill color.Color, radius float32) *canvas.Rectangle {
	r := canvas.NewRectangle(fill)
	r.CornerRadius = radius
	return r
}

// card wraps content in a white rounded surface with comfortable padding.
func card(inner fyne.CanvasObject) fyne.CanvasObject {
	return container.NewStack(rounded(colCard, 16), insets(inner, 16, 16, 16, 16))
}
