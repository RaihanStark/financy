package component

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
)

// ColumnsLayout lays children out horizontally, each column sized by a weight.
// Gap is 0 so adjacent cell borders merge into continuous grid lines.
type ColumnsLayout struct {
	Weights []float32
	Gap     float32
}

func (c *ColumnsLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var h, w float32
	for _, o := range objs {
		m := o.MinSize()
		if m.Height > h {
			h = m.Height
		}
		w += m.Width
	}
	return fyne.NewSize(w+c.Gap*float32(len(objs)-1), h)
}

func (c *ColumnsLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	var total float32
	for _, wt := range c.Weights {
		total += wt
	}
	if total == 0 {
		total = 1
	}
	gaps := c.Gap * float32(len(objs)-1)
	avail := size.Width - gaps
	var x float32
	for i, o := range objs {
		var wt float32 = 1
		if i < len(c.Weights) {
			wt = c.Weights[i]
		}
		cw := avail * wt / total
		o.Resize(fyne.NewSize(cw, size.Height))
		o.Move(fyne.NewPos(x, 0))
		x += cw + c.Gap
	}
}

// TableSpec describes a data grid.
type TableSpec struct {
	Headers []string
	Weights []float32
	Aligns  []fyne.TextAlign // per-column text alignment
	Rows    [][]Cell         // each cell carries text + optional color/widget
}

// Cell is one table cell. If obj is set it is rendered instead of text.
type Cell struct {
	text string
	col  color.Color
	bold bool
	obj  fyne.CanvasObject
}

// Cell constructors.
func Tc(text string) Cell                 { return Cell{text: text, col: colText} }
func Tcc(text string, c color.Color) Cell { return Cell{text: text, col: c} }
func Tcb(text string, c color.Color) Cell { return Cell{text: text, col: c, bold: true} }
func Tco(o fyne.CanvasObject) Cell        { return Cell{obj: o} }

func (s TableSpec) align(i int) fyne.TextAlign {
	if i < len(s.Aligns) {
		return s.Aligns[i]
	}
	return fyne.TextAlignLeading
}

// GridCell wraps content in a bordered, padded cell with the given fill.
func GridCell(content fyne.CanvasObject, fill color.Color) fyne.CanvasObject {
	bg := canvas.NewRectangle(fill)
	bg.StrokeColor = colBorder
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(3, 3, 7, 7), content))
}

// Build renders the table as a stack of bordered rows with a gray header row.
func (s TableSpec) Build() fyne.CanvasObject {
	lay := &ColumnsLayout{Weights: s.Weights, Gap: 0}

	// Header row.
	var hcells []fyne.CanvasObject
	for i, h := range s.Headers {
		t := txt(h, colText, 11.5, true)
		t.Alignment = s.align(i)
		hcells = append(hcells, GridCell(t, colSurfaceHi))
	}
	header := container.New(lay, hcells...)

	rowsBox := container.NewVBox(header)
	for ri, r := range s.Rows {
		fill := colSurface
		if ri%2 == 1 {
			fill = colAltRow
		}
		var rcells []fyne.CanvasObject
		for ci, c := range r {
			if c.obj != nil {
				rcells = append(rcells, GridCell(c.obj, fill))
				continue
			}
			col := c.col
			if col == nil {
				col = colText
			}
			var t *canvas.Text
			// Right-aligned columns are numeric → monospace for tidy alignment.
			if s.align(ci) == fyne.TextAlignTrailing {
				t = mono(c.text, col, 12, c.bold)
			} else {
				t = txt(c.text, col, 12, c.bold)
			}
			t.Alignment = s.align(ci)
			rcells = append(rcells, GridCell(t, fill))
		}
		rowsBox.Add(container.New(lay, rcells...))
	}

	// VBox adds spacing between rows; wrap tightly to keep grid lines flush.
	return container.New(layout.NewCustomPaddedVBoxLayout(0), rowsBox.Objects...)
}
