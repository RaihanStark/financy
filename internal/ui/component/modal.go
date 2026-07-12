package component

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/ui/style"
)

// Modal is the app's replacement for Fyne's stock dialogs: a full-canvas
// overlay with a dimming scrim and a centered rounded card that matches the
// design system (surface fill, hairline border, soft halo). Content and
// footer buttons are supplied by the caller; Show/Hide manage the overlay.
type Modal struct {
	widget.BaseWidget
	cv       fyne.Canvas
	title    string
	content  fyne.CanvasObject
	buttons  []fyne.CanvasObject
	onClosed func()
	cardSize fyne.Size // requested card size; zero → content-driven
}

// NewModal builds a modal card titled title around content, shown on cv when
// Show is called. Footer buttons (right-aligned) are optional via SetButtons —
// content that carries its own action row simply doesn't set any.
func NewModal(cv fyne.Canvas, title string, content fyne.CanvasObject) *Modal {
	m := &Modal{cv: cv, title: title, content: content}
	m.ExtendBaseWidget(m)
	return m
}

// SetButtons puts actions in the card footer, right-aligned in given order.
func (m *Modal) SetButtons(buttons ...fyne.CanvasObject) { m.buttons = buttons }

// SetCardSize requests a card size (clamped to the canvas and the content's
// minimum). Zero leaves the card sized to its content.
func (m *Modal) SetCardSize(s fyne.Size) {
	m.cardSize = s
	m.Refresh()
}

// SetOnClosed registers fn to run whenever the modal is hidden.
func (m *Modal) SetOnClosed(fn func()) { m.onClosed = fn }

// Show puts the modal on the canvas overlay stack, sized to the full canvas.
func (m *Modal) Show() {
	if m.cv == nil {
		return
	}
	m.BaseWidget.Show()
	m.cv.Overlays().Add(m)
	m.Resize(m.cv.Size())
}

// Hide removes the modal from the overlay stack and fires onClosed.
func (m *Modal) Hide() {
	if m.cv != nil {
		m.cv.Overlays().Remove(m)
	}
	m.BaseWidget.Hide()
	if m.onClosed != nil {
		m.onClosed()
	}
}

// Tapped swallows clicks on the scrim so the UI beneath stays inert. The modal
// intentionally does not dismiss on outside taps — same as the dialogs it
// replaces.
func (m *Modal) Tapped(_ *fyne.PointEvent) {}

func (m *Modal) CreateRenderer() fyne.WidgetRenderer {
	r := &modalRenderer{m: m}
	r.scrim = canvas.NewRectangle(color.Transparent)
	r.halo = canvas.NewRectangle(color.Transparent)
	r.halo.CornerRadius = style.Radius + 6
	r.card = canvas.NewRectangle(colSurface)
	r.card.StrokeWidth = 1
	r.card.CornerRadius = style.Radius + 2

	r.titleTxt = txt(m.title, colText, 15, true)
	header := container.New(padCell(14, 18), r.titleTxt)
	body := container.New(layout.NewCustomPaddedLayout(0, 14, 18, 18), m.content)

	var footer fyne.CanvasObject
	if len(m.buttons) > 0 {
		row := container.NewHBox(layout.NewSpacer())
		for _, b := range m.buttons {
			row.Add(b)
		}
		footer = container.New(layout.NewCustomPaddedLayout(0, 14, 18, 18), row)
	}
	r.inner = container.NewStack(r.card, container.NewBorder(header, footer, nil, nil, body))

	r.applyColors()
	return r
}

type modalRenderer struct {
	m        *Modal
	scrim    *canvas.Rectangle
	halo     *canvas.Rectangle
	card     *canvas.Rectangle
	titleTxt *canvas.Text
	inner    *fyne.Container
}

func (r *modalRenderer) applyColors() {
	r.scrim.FillColor = color.NRGBA{A: 0x5c}
	haloA := uint8(0x16)
	if style.Dark {
		r.scrim.FillColor = color.NRGBA{A: 0x84}
		haloA = 0x42
	}
	r.halo.FillColor = color.NRGBA{A: haloA}
	r.card.FillColor = colSurface
	r.card.StrokeColor = colBorder
	r.titleTxt.Color = colText
}

// cardBounds computes the centered card rectangle for the given canvas size.
func (r *modalRenderer) cardBounds(size fyne.Size) (fyne.Position, fyne.Size) {
	const margin = 24
	min := r.inner.MinSize()
	w, h := min.Width, min.Height
	if r.m.cardSize.Width > w {
		w = r.m.cardSize.Width
	}
	if r.m.cardSize.Height > h {
		h = r.m.cardSize.Height
	}
	if maxW := size.Width - 2*margin; w > maxW {
		w = maxW
	}
	if maxH := size.Height - 2*margin; h > maxH {
		h = maxH
	}
	pos := fyne.NewPos((size.Width-w)/2, (size.Height-h)/2)
	return pos, fyne.NewSize(w, h)
}

func (r *modalRenderer) Layout(size fyne.Size) {
	r.scrim.Resize(size)
	r.scrim.Move(fyne.NewPos(0, 0))
	pos, cs := r.cardBounds(size)
	r.halo.Resize(fyne.NewSize(cs.Width+8, cs.Height+8))
	r.halo.Move(pos.SubtractXY(4, 4))
	r.inner.Resize(cs)
	r.inner.Move(pos)
}

func (r *modalRenderer) MinSize() fyne.Size { return r.inner.MinSize() }

func (r *modalRenderer) Refresh() {
	r.applyColors()
	r.scrim.Refresh()
	r.halo.Refresh()
	r.card.Refresh()
	r.titleTxt.Refresh()
	r.Layout(r.m.Size())
}

func (r *modalRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.scrim, r.halo, r.inner}
}

func (r *modalRenderer) Destroy() {}
