package component

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// Native, dependency-free charts drawn with Fyne canvas primitives so they theme
// cleanly and stay crisp at any size. Each has a left Y-axis gutter with value
// tick labels + gridlines, an X axis owned by the caller, and is hoverable:
// moving over a bucket highlights it and shows a tooltip (via the tooltip sink).

const (
	chartHeight = 150
	// ChartGutter is the left space reserved for Y-axis tick labels. Callers
	// align their X-axis labels by insetting them this far from the left.
	ChartGutter float32 = 54
	chartTopPad float32  = 10 // headroom so the top tick label isn't clipped
)

// tick is one Y-axis gridline plus its right-aligned value label.
type tick struct {
	line  *canvas.Line
	label *canvas.Text
}

func newTicks(vals []int, yfmt func(int) string) ([]tick, []int) {
	ticks := make([]tick, len(vals))
	for i, v := range vals {
		ln := canvas.NewLine(withAlpha(colText, 0x12))
		ln.StrokeWidth = 1
		lb := canvas.NewText(yfmt(v), colTextDim)
		lb.TextSize = 9.5
		ticks[i] = tick{line: ln, label: lb}
	}
	return ticks, vals
}

// placeTicks positions gridlines across the plot and right-aligns labels in the
// gutter, vertically centered on each gridline.
func placeTicks(ticks []tick, vals []int, size fyne.Size, yOf func(int) float32) {
	for i, tk := range ticks {
		y := yOf(vals[i])
		tk.line.Position1 = fyne.NewPos(ChartGutter, y)
		tk.line.Position2 = fyne.NewPos(size.Width, y)
		w := tk.label.MinSize().Width
		tk.label.Move(fyne.NewPos(ChartGutter-6-w, y-tk.label.MinSize().Height/2))
	}
}

func tooltipAt(w fyne.CanvasObject, tips []string, i int, evPos fyne.Position) {
	if i < 0 || i >= len(tips) {
		HideTooltip()
		return
	}
	abs := fyne.CurrentApp().Driver().AbsolutePositionForObject(w)
	ShowTooltip(tips[i], abs.Add(evPos).Add(fyne.NewPos(14, 16)))
}

// ---- grouped bar chart (two series over the same buckets) ----

type barChart struct {
	widget.BaseWidget
	a, b       []int
	max        int
	colA, colB color.Color
	tips       []string

	highlight *canvas.Rectangle
	base      *canvas.Rectangle
	bars      []*canvas.Rectangle
	ticks     []tick
	tickVals  []int
	hover     int
}

func barPairChart(a, b []int, colA, colB color.Color, tips []string, yfmt func(int) string) fyne.CanvasObject {
	max := 1
	for i := range a {
		if a[i] > max {
			max = a[i]
		}
		if i < len(b) && b[i] > max {
			max = b[i]
		}
	}
	c := &barChart{a: a, b: b, max: max, colA: colA, colB: colB, tips: tips, hover: -1}
	c.highlight = canvas.NewRectangle(withAlpha(colPrimary, 0x14))
	c.highlight.Hide()
	c.base = canvas.NewRectangle(colBorder)
	for range a {
		c.bars = append(c.bars, canvas.NewRectangle(colA), canvas.NewRectangle(colB))
	}
	c.ticks, c.tickVals = newTicks([]int{0, max / 2, max}, yfmt)
	c.ExtendBaseWidget(c)
	return c
}

func (c *barChart) CreateRenderer() fyne.WidgetRenderer {
	objs := []fyne.CanvasObject{c.highlight}
	for _, tk := range c.ticks {
		objs = append(objs, tk.line)
	}
	objs = append(objs, c.base)
	for _, r := range c.bars {
		objs = append(objs, r)
	}
	for _, tk := range c.ticks {
		objs = append(objs, tk.label)
	}
	return &chartRenderer{objs: objs, layout: c.layout, min: func() fyne.Size {
		return fyne.NewSize(ChartGutter+float32(len(c.a))*28, chartHeight)
	}}
}

func (c *barChart) plotWidth(w float32) float32 { return w - ChartGutter }
func (c *barChart) groupWidth(w float32) float32 {
	if len(c.a) == 0 {
		return c.plotWidth(w)
	}
	return c.plotWidth(w) / float32(len(c.a))
}

func (c *barChart) layout(size fyne.Size) {
	n := len(c.a)
	if n == 0 || c.max <= 0 {
		return
	}
	baselineY := size.Height - 1
	plotH := baselineY - chartTopPad
	yOf := func(v int) float32 { return baselineY - plotH*float32(v)/float32(c.max) }

	placeTicks(c.ticks, c.tickVals, size, yOf)
	c.base.Resize(fyne.NewSize(c.plotWidth(size.Width), 1))
	c.base.Move(fyne.NewPos(ChartGutter, baselineY))

	groupW := c.groupWidth(size.Width)
	barW := groupW * 0.28
	gap := groupW * 0.06
	for i := range n {
		cx := ChartGutter + groupW*float32(i) + groupW/2
		ta, tb := yOf(c.a[i]), yOf(c.b[i])
		ra, rb := c.bars[2*i], c.bars[2*i+1]
		ra.Resize(fyne.NewSize(barW, baselineY-ta))
		ra.Move(fyne.NewPos(cx-barW-gap/2, ta))
		rb.Resize(fyne.NewSize(barW, baselineY-tb))
		rb.Move(fyne.NewPos(cx+gap/2, tb))
	}
	if c.hover >= 0 && c.hover < n {
		c.highlight.Resize(fyne.NewSize(groupW, size.Height))
		c.highlight.Move(fyne.NewPos(ChartGutter+groupW*float32(c.hover), 0))
	}
}

func (c *barChart) bucketAt(x float32) int {
	n := len(c.a)
	w := c.Size().Width
	if n == 0 || w <= 0 {
		return -1
	}
	i := int((x - ChartGutter) / c.groupWidth(w))
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

func (c *barChart) MouseIn(ev *desktop.MouseEvent)    { c.MouseMoved(ev) }
func (c *barChart) MouseMoved(ev *desktop.MouseEvent) { c.update(c.bucketAt(ev.Position.X), ev.Position) }
func (c *barChart) MouseOut() {
	c.hover = -1
	c.highlight.Hide()
	canvas.Refresh(c)
	HideTooltip()
}

func (c *barChart) update(i int, evPos fyne.Position) {
	if i != c.hover {
		c.hover = i
		if i >= 0 {
			gw := c.groupWidth(c.Size().Width)
			c.highlight.Resize(fyne.NewSize(gw, c.Size().Height))
			c.highlight.Move(fyne.NewPos(ChartGutter+gw*float32(i), 0))
			c.highlight.Show()
		} else {
			c.highlight.Hide()
		}
		canvas.Refresh(c)
	}
	tooltipAt(c, c.tips, i, evPos)
}

// ---- net-worth line chart ----

type lineChart struct {
	widget.BaseWidget
	vals       []int
	pmin, pmax int // padded range used for plotting
	col        color.Color
	tips       []string

	guide    *canvas.Line
	base     *canvas.Rectangle
	lines    []*canvas.Line
	dots     []*canvas.Circle
	ticks    []tick
	tickVals []int
	hover    int
}

func lineChartWidget(vals []int, col color.Color, tips []string, yfmt func(int) string) fyne.CanvasObject {
	c := &lineChart{vals: vals, col: col, tips: tips, hover: -1}
	dataMin, dataMax := dataRange(vals)
	c.pmin, c.pmax = padRange(dataMin, dataMax)
	c.guide = canvas.NewLine(withAlpha(colText, 0x40))
	c.guide.StrokeWidth = 1
	c.guide.Hide()
	c.base = canvas.NewRectangle(colBorder)
	for i := 0; i < len(vals)-1; i++ {
		ln := canvas.NewLine(col)
		ln.StrokeWidth = 2
		c.lines = append(c.lines, ln)
	}
	for range vals {
		c.dots = append(c.dots, canvas.NewCircle(col))
	}
	// Ticks at the real data min / midpoint / max, positioned on the padded scale.
	c.ticks, c.tickVals = newTicks([]int{dataMin, (dataMin + dataMax) / 2, dataMax}, yfmt)
	c.ExtendBaseWidget(c)
	return c
}

func dataRange(vals []int) (int, int) {
	if len(vals) == 0 {
		return 0, 1
	}
	min, max := vals[0], vals[0]
	for _, v := range vals {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

// padRange brackets [min,max] with ~10% headroom so the line isn't glued to the
// top or bottom edge.
func padRange(min, max int) (int, int) {
	span := max - min
	if span == 0 {
		if span = max; span == 0 {
			span = 1
		}
	}
	return min - span/10, max + span/10
}

const linePad float32 = 6

func (c *lineChart) at(i int, size fyne.Size) fyne.Position {
	n := len(c.vals)
	baselineY := size.Height - 1
	plotH := baselineY - chartTopPad
	rng := c.pmax - c.pmin
	if rng <= 0 {
		rng = 1
	}
	plotW := size.Width - ChartGutter
	x := ChartGutter + linePad
	if n > 1 {
		x = ChartGutter + linePad + (plotW-2*linePad)*float32(i)/float32(n-1)
	}
	y := baselineY - plotH*float32(c.vals[i]-c.pmin)/float32(rng)
	return fyne.NewPos(x, y)
}

func (c *lineChart) yOf(v int, size fyne.Size) float32 {
	baselineY := size.Height - 1
	plotH := baselineY - chartTopPad
	rng := c.pmax - c.pmin
	if rng <= 0 {
		rng = 1
	}
	return baselineY - plotH*float32(v-c.pmin)/float32(rng)
}

func (c *lineChart) CreateRenderer() fyne.WidgetRenderer {
	objs := []fyne.CanvasObject{c.guide}
	for _, tk := range c.ticks {
		objs = append(objs, tk.line)
	}
	objs = append(objs, c.base)
	for _, ln := range c.lines {
		objs = append(objs, ln)
	}
	for _, d := range c.dots {
		objs = append(objs, d)
	}
	for _, tk := range c.ticks {
		objs = append(objs, tk.label)
	}
	return &chartRenderer{objs: objs, layout: c.layout, min: func() fyne.Size {
		return fyne.NewSize(ChartGutter+float32(len(c.vals))*28, chartHeight)
	}}
}

func (c *lineChart) layout(size fyne.Size) {
	n := len(c.vals)
	if n == 0 {
		return
	}
	placeTicks(c.ticks, c.tickVals, size, func(v int) float32 { return c.yOf(v, size) })
	c.base.Resize(fyne.NewSize(size.Width-ChartGutter, 1))
	c.base.Move(fyne.NewPos(ChartGutter, size.Height-1))

	for i := 0; i < n-1; i++ {
		c.lines[i].Position1, c.lines[i].Position2 = c.at(i, size), c.at(i+1, size)
	}
	const r = 3
	for i := range n {
		p := c.at(i, size)
		c.dots[i].Move(fyne.NewPos(p.X-r, p.Y-r))
		c.dots[i].Resize(fyne.NewSize(2*r, 2*r))
	}
	if c.hover >= 0 && c.hover < n {
		x := c.at(c.hover, size).X
		c.guide.Position1 = fyne.NewPos(x, 0)
		c.guide.Position2 = fyne.NewPos(x, size.Height)
	}
}

func (c *lineChart) nearest(x float32) int {
	n := len(c.vals)
	w := c.Size().Width
	if n == 0 || w <= 0 {
		return -1
	}
	if n == 1 {
		return 0
	}
	step := (w - ChartGutter - 2*linePad) / float32(n-1)
	i := int((x - ChartGutter - linePad + step/2) / step)
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

func (c *lineChart) MouseIn(ev *desktop.MouseEvent)    { c.MouseMoved(ev) }
func (c *lineChart) MouseMoved(ev *desktop.MouseEvent) { c.update(c.nearest(ev.Position.X), ev.Position) }
func (c *lineChart) MouseOut() {
	c.hover = -1
	c.guide.Hide()
	canvas.Refresh(c)
	HideTooltip()
}

func (c *lineChart) update(i int, evPos fyne.Position) {
	if i != c.hover {
		c.hover = i
		if i >= 0 {
			x := c.at(i, c.Size()).X
			c.guide.Position1 = fyne.NewPos(x, 0)
			c.guide.Position2 = fyne.NewPos(x, c.Size().Height)
			c.guide.Show()
		} else {
			c.guide.Hide()
		}
		canvas.Refresh(c)
	}
	tooltipAt(c, c.tips, i, evPos)
}

// ---- shared renderer ----

type chartRenderer struct {
	objs   []fyne.CanvasObject
	layout func(fyne.Size)
	min    func() fyne.Size
}

func (r *chartRenderer) Layout(size fyne.Size)        { r.layout(size) }
func (r *chartRenderer) MinSize() fyne.Size           { return r.min() }
func (r *chartRenderer) Refresh()                     {}
func (r *chartRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *chartRenderer) Destroy()                     {}
