package mobileui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// screenHeader is the large title (with optional subtitle) at the top of a tab.
func screenHeader(title, subtitle string) fyne.CanvasObject {
	t := newText(title, colInk, 24, true)
	if subtitle == "" {
		return t
	}
	return container.NewVBox(t, newText(subtitle, colInkDim, 12, false))
}

// sectionHeader labels a group; a trailing action link is optional.
func sectionHeader(title, action string, tap func()) fyne.CanvasObject {
	t := newText(title, colInk, 16, true)
	if action == "" || tap == nil {
		return t
	}
	link := widget.NewButton(action, tap)
	link.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, t, link)
}

// mutedCard is an empty-state card with a centered message.
func mutedCard(msg string) fyne.CanvasObject {
	return container.NewStack(rounded(colCard, 16),
		insets(container.NewCenter(newText(msg, colInkDim, 13, false)), 26, 26, 16, 16))
}

func iconButton(res fyne.Resource, tap func()) *widget.Button {
	b := widget.NewButtonWithIcon("", res, tap)
	b.Importance = widget.LowImportance
	return b
}

// thinLine is a 1px horizontal hairline.
func thinLine() fyne.CanvasObject {
	l := canvas.NewRectangle(colLine)
	l.SetMinSize(fyne.NewSize(0, 1))
	return l
}

// field stacks a caption above its control — the mobile-native form row.
func field(label string, control fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(newText(label, colInkDim, 12, false), control, gap(8))
}

// wrapText is a dim, word-wrapping paragraph. Unlike canvas.Text (which never
// wraps and so forces a container as wide as the whole line), a wrapping label
// reports a small min width, so it can't cause horizontal overflow.
func wrapText(s string) fyne.CanvasObject {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	l.Importance = widget.LowImportance
	return l
}

// heroStat is a small white-on-indigo label/value pair for the net-worth card.
func heroStat(label, val string) fyne.CanvasObject {
	return container.NewVBox(
		newText(label, withAlpha(white, 0x99), 11, false),
		newText(val, white, 15, true),
	)
}

// centerInitial centers a single-character badge glyph. canvas.Text renders a
// lone glyph a few pixels right of its box centre (a constant, glyph-independent
// offset), so we pad the right to bring the ink back onto the geometric centre.
func centerInitial(glyph fyne.CanvasObject, size float32) fyne.CanvasObject {
	return container.NewCenter(insets(glyph, 0, 0, 0, size*0.22))
}

// initialCircle is a round, color-tinted badge showing a single initial.
func initialCircle(letter string, c color.NRGBA, size float32) fyne.CanvasObject {
	circle := canvas.NewCircle(withAlpha(c, 0x22))
	txt := newText(letter, c, size*0.4, true)
	// Only lone glyphs need re-centering; multi-character badges like "#12" are
	// wide enough that the constant offset is negligible.
	var glyph fyne.CanvasObject = container.NewCenter(txt)
	if len([]rune(letter)) == 1 {
		glyph = centerInitial(txt, size)
	}
	return container.NewGridWrap(fyne.NewSize(size, size),
		container.NewStack(circle, glyph))
}

// avatar is the badge shown at the start of a transaction row.
func avatar(kind, title string) fyne.CanvasObject {
	return initialCircle(initialOf(title), accentFor(kind), 36)
}

// rowDivider is a hairline indented past the avatar, separating list rows.
func rowDivider() fyne.CanvasObject {
	l := canvas.NewRectangle(colLine)
	l.SetMinSize(fyne.NewSize(0, 1))
	return insets(l, 0, 0, 52, 8)
}

// listCard wraps a set of rows in one rounded surface.
func listCard(rows ...fyne.CanvasObject) fyne.CanvasObject {
	return container.NewStack(rounded(colCard, 16), insets(container.NewVBox(rows...), 6, 6, 4, 4))
}

// flexWidth reports zero minimum width (height from its child) and lays the
// child out to whatever width it's given. Wrapping a text column in it stops a
// long line from forcing the whole row — and thus a list — wider than the
// screen, which is what caused sideways scrolling.
type flexWidth struct{}

func (flexWidth) MinSize(objs []fyne.CanvasObject) fyne.Size {
	h := float32(0)
	for _, o := range objs {
		if s := o.MinSize(); s.Height > h {
			h = s.Height
		}
	}
	return fyne.NewSize(0, h)
}

func (flexWidth) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

func flexBox(o fyne.CanvasObject) *fyne.Container { return container.New(flexWidth{}, o) }

// ratioLayout arranges children horizontally, sizing each to its weight — used
// for proportional fills (progress bars) where a percentage width is needed.
type ratioLayout struct{ weights []int }

func newRatioLayout(w ...int) *ratioLayout { return &ratioLayout{weights: w} }

func (r *ratioLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	h := float32(0)
	for _, o := range objs {
		if s := o.MinSize(); s.Height > h {
			h = s.Height
		}
	}
	return fyne.NewSize(0, h)
}

func (r *ratioLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	total := 0
	for _, w := range r.weights {
		total += w
	}
	if total == 0 {
		total = 1
	}
	x := float32(0)
	for i, o := range objs {
		w := 0
		if i < len(r.weights) {
			w = r.weights[i]
		}
		cw := size.Width * float32(w) / float32(total)
		o.Resize(fyne.NewSize(cw, size.Height))
		o.Move(fyne.NewPos(x, 0))
		x += cw
	}
}
