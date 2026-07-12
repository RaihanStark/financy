package component

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// TappableRow is a clickable, hover-highlighting container — the basis for
// selectable master lists and clickable data rows.
type TappableRow struct {
	widget.BaseWidget
	content     fyne.CanvasObject
	bg          *canvas.Rectangle
	onTap       func()
	onSecondary func(fyne.Position)
	baseFill    color.Color
	selected    bool
}

func newTappableRow(content fyne.CanvasObject, baseFill color.Color, onTap func()) *TappableRow {
	r := &TappableRow{content: content, onTap: onTap, baseFill: baseFill}
	r.bg = canvas.NewRectangle(baseFill)
	r.ExtendBaseWidget(r)
	return r
}

func (r *TappableRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(r.bg, r.content))
}

func (r *TappableRow) Tapped(_ *fyne.PointEvent) {
	if r.onTap != nil {
		r.onTap()
	}
}

// TappedSecondary handles right-click for desktop-style context menus.
func (r *TappableRow) TappedSecondary(e *fyne.PointEvent) {
	if r.onSecondary != nil {
		r.onSecondary(e.AbsolutePosition)
	}
}

func (r *TappableRow) MouseIn(_ *desktop.MouseEvent) {
	if !r.selected {
		r.bg.FillColor = colSurfaceHi
		r.bg.Refresh()
	}
}
func (r *TappableRow) MouseMoved(_ *desktop.MouseEvent) {}
func (r *TappableRow) MouseOut() {
	if !r.selected {
		r.bg.FillColor = r.baseFill
		r.bg.Refresh()
	}
}

func (r *TappableRow) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (r *TappableRow) SetSelected(b bool) {
	r.selected = b
	if b {
		r.bg.FillColor = withAlpha(colPrimary, 0x28)
	} else {
		r.bg.FillColor = r.baseFill
	}
	r.bg.Refresh()
}

// SetOnSecondary sets the right-click handler.
func (r *TappableRow) SetOnSecondary(fn func(fyne.Position)) { r.onSecondary = fn }

// SetBordered draws a hairline border with rounded corners around the row
// (used for card-style rows).
func (r *TappableRow) SetBordered() {
	r.bg.StrokeColor = colBorder
	r.bg.StrokeWidth = 1
	r.bg.CornerRadius = radius
}

// appBar is the in-content header: title + subtitle on the left, action buttons
// on the right. It sits directly on the window background — no chrome strip —
// so screens open with a clean, airy heading.
func appBar(title, subtitle string, actions ...fyne.CanvasObject) fyne.CanvasObject {
	left := container.NewVBox(txt(title, colText, 20, true))
	if subtitle != "" {
		left = container.NewVBox(txt(title, colText, 20, true), spacerH(1), txt(subtitle, colTextDim, 11.5, false))
	}
	right := container.NewHBox()
	for _, a := range actions {
		if a != nil {
			right.Add(a)
		}
	}
	row := container.NewBorder(nil, nil, container.NewPadded(left), container.NewCenter(right))
	return container.New(padCell(4, 0), row)
}

// primaryButton is a filled accent action button.
func primaryButton(label string, icon fyne.Resource, fn func()) *widget.Button {
	b := widget.NewButtonWithIcon(label, icon, fn)
	b.Importance = widget.HighImportance
	return b
}

// secondaryButton is a low-emphasis action button.
func secondaryButton(label string, icon fyne.Resource, fn func()) *widget.Button {
	b := widget.NewButtonWithIcon(label, icon, fn)
	b.Importance = widget.MediumImportance
	return b
}

// detailField renders a "Label  value" pair for detail panels.
func detailField(label, value string, valueCol color.Color) fyne.CanvasObject {
	return container.NewBorder(nil, nil,
		container.NewGridWrap(fyne.NewSize(150, 18), txt(label, colTextDim, 12, false)),
		nil,
		txt(value, valueCol, 12, true),
	)
}

// alertBanner is a colored notice strip used for things needing attention.
func alertBanner(icon fyne.Resource, message string, col color.Color, action fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(withAlpha(col, 0x1e))
	bg.StrokeColor = withAlpha(col, 0x88)
	bg.StrokeWidth = 1
	bg.CornerRadius = radius
	ic := widget.NewIcon(icon)
	left := container.NewHBox(ic, txt(message, col, 12.5, true))
	var row fyne.CanvasObject
	if action != nil {
		row = container.NewBorder(nil, nil, left, action)
	} else {
		row = left
	}
	return container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(7, 7, 10, 10), row))
}

// emptyState shows a centered hint when a list/detail has nothing to show.
func emptyState(message string) fyne.CanvasObject {
	return container.NewCenter(txt(message, colTextDim, 12, false))
}

// padCell returns a layout with vertical/horizontal padding for compact rows.
func padCell(v, h float32) fyne.Layout {
	return layout.NewCustomPaddedLayout(v, v, h, h)
}

// pillStat is a compact inline metric used inside detail headers.
func pillStat(label, value string, valueCol color.Color) fyne.CanvasObject {
	return container.NewVBox(
		txt(label, colTextDim, 10, true),
		txt(value, valueCol, 15, true),
	)
}
