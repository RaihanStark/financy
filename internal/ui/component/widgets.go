package component

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// panel wraps content in a clean white card with a hairline border and soft
// rounded corners.
func panel(content fyne.CanvasObject) *fyne.Container {
	bg := canvas.NewRectangle(colSurface)
	bg.StrokeColor = colBorder
	bg.StrokeWidth = 1
	bg.CornerRadius = radius
	return container.NewStack(bg, container.NewPadded(content))
}

// groupBox is a titled card group with a tinted header strip.
func groupBox(title string, content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(colSurface)
	bg.StrokeColor = colBorder
	bg.StrokeWidth = 1
	bg.CornerRadius = radius

	titleBar := canvas.NewRectangle(colSurfaceHi)
	titleBar.CornerRadius = radiusSm
	header := container.NewStack(titleBar, container.NewPadded(txt(title, colText, 12.5, true)))

	body := container.NewPadded(content)
	inner := container.NewBorder(header, nil, nil, nil, body)
	return container.NewStack(bg, inner)
}

// txt builds a canvas.Text with explicit color/size/weight.
func txt(s string, col color.Color, size float32, bold bool) *canvas.Text {
	t := canvas.NewText(s, col)
	t.TextSize = size
	t.TextStyle = fyne.TextStyle{Bold: bold}
	return t
}

// mono builds monospaced text — used for right-aligned figures in grids.
func mono(s string, col color.Color, size float32, bold bool) *canvas.Text {
	t := canvas.NewText(s, col)
	t.TextSize = size
	t.TextStyle = fyne.TextStyle{Bold: bold, Monospace: true}
	return t
}

// moneyColor picks green/red/neutral based on the sign of an amount.
func moneyColor(amount int) color.Color {
	switch {
	case amount > 0:
		return colPositive
	case amount < 0:
		return colNegative
	default:
		return colText
	}
}

// statCard is a compact bordered summary tile: caption above, value below.
func statCard(caption, value string, valueCol color.Color, sub string) fyne.CanvasObject {
	cap := txt(caption, colTextDim, 10.5, true)
	val := txt(value, valueCol, 19, true)
	items := []fyne.CanvasObject{cap, spacerH(3), val}
	if sub != "" {
		items = append(items, spacerH(2), txt(sub, colTextDim, 10.5, false))
	}
	bg := canvas.NewRectangle(colSurface)
	bg.StrokeColor = colBorder
	bg.StrokeWidth = 1
	bg.CornerRadius = radius
	return container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(12, 12, 14, 14), container.NewVBox(items...)))
}

// sectionTitle is a heading used above a block of content.
func sectionTitle(s string) fyne.CanvasObject {
	return txt(s, colText, 13, true)
}

// pageHeader renders a compact screen title plus subtitle.
func pageHeader(title, subtitle string) fyne.CanvasObject {
	t := txt(title, colText, 16, true)
	if subtitle == "" {
		return t
	}
	return container.NewVBox(t, txt(subtitle, colTextDim, 11, false))
}

// badge renders a small lightly tinted pill-shaped status chip.
func badge(label string, col color.Color) fyne.CanvasObject {
	bg := canvas.NewRectangle(withAlpha(col, 0x1f))
	bg.StrokeColor = withAlpha(col, 0x70)
	bg.StrokeWidth = 1
	bg.CornerRadius = 8
	t := txt(label, col, 10.5, true)
	cell := container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(2, 2, 8, 8), t))
	// Keep the chip from stretching across the whole grid column.
	return container.NewHBox(cell, layout.NewSpacer())
}

// progressBar draws a flat square-filled track for a 0..1 ratio.
func progressBar(ratio float64, col color.Color) fyne.CanvasObject {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	track := canvas.NewRectangle(colSurfaceHi)
	track.CornerRadius = 5
	track.SetMinSize(fyne.NewSize(0, 10))
	fill := canvas.NewRectangle(col)
	fill.CornerRadius = 5
	var filler fyne.CanvasObject
	if ratio <= 0 {
		filler = layout.NewSpacer()
	} else if ratio >= 1 {
		filler = fill
	} else {
		filler = container.New(&ratioLayout{leftW: float32(ratio * 1000), rightW: float32((1 - ratio) * 1000)},
			fill, layout.NewSpacer())
	}
	return container.NewStack(track, filler)
}

// ratioLayout splits horizontal space between two children by weight.
type ratioLayout struct{ leftW, rightW float32 }

func (r *ratioLayout) MinSize(_ []fyne.CanvasObject) fyne.Size { return fyne.NewSize(0, 10) }
func (r *ratioLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if len(objs) != 2 {
		return
	}
	total := r.leftW + r.rightW
	lw := size.Width * r.leftW / total
	objs[0].Resize(fyne.NewSize(lw, size.Height))
	objs[0].Move(fyne.NewPos(0, 0))
	objs[1].Resize(fyne.NewSize(size.Width-lw, size.Height))
	objs[1].Move(fyne.NewPos(lw, 0))
}

// hbar is a labeled horizontal bar used in mini charts.
func hbar(label, valueLabel string, ratio float64, col color.Color) fyne.CanvasObject {
	return container.NewBorder(nil, nil,
		container.NewGridWrap(fyne.NewSize(150, 16), txt(label, colText, 11.5, false)),
		container.NewGridWrap(fyne.NewSize(120, 16), alignRight(mono(valueLabel, colTextDim, 11.5, false))),
		progressBar(ratio, col),
	)
}

func alignRight(t *canvas.Text) fyne.CanvasObject {
	t.Alignment = fyne.TextAlignTrailing
	return t
}

func withAlpha(c color.Color, a uint8) color.Color {
	r, g, b, _ := c.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: a}
}

// divider is a thin horizontal separator.
func divider() fyne.CanvasObject {
	return widget.NewSeparator()
}

// spacerH adds fixed vertical space.
func spacerH(h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(0, h))
	return r
}

// canvasTransparent returns a fresh transparent rectangle.
func canvasTransparent() *canvas.Rectangle {
	return canvas.NewRectangle(color.Transparent)
}

// spacerW adds fixed horizontal space.
func spacerW(w float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(w, 0))
	return r
}
