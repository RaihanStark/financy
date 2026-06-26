package component

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	gchart "github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

// Charts render their data series with go-chart (anti-aliased, area-filled,
// rounded) into a transparent raster image, then layer a native Fyne overlay on
// top: a left Y-axis gutter with themed tick labels + gridlines, plus hover
// interaction (crosshair / highlight band + a tooltip via the tooltip sink).
// go-chart owns the look of the data; Fyne owns the chrome and the interactivity,
// so the charts stay crisp, theme cleanly, and react to the cursor.

const (
	chartHeight = 150
	// ChartGutter is the left space reserved for Y-axis tick labels. Callers
	// align their X-axis labels by insetting them this far from the left.
	ChartGutter float32 = 54
	chartTopPad float32 = 10 // headroom so the top tick label isn't clipped
	linePad     float32 = 6  // inset so edge points/dots aren't clipped
)

// dcolor converts a Fyne/std color to go-chart's drawing.Color.
func dcolor(c color.Color) drawing.Color {
	r, g, b, a := c.RGBA()
	return drawing.Color{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// dalpha is dcolor with an explicit alpha — for translucent area fills.
func dalpha(c color.Color, a uint8) drawing.Color {
	d := dcolor(c)
	d.A = a
	return d
}

// canvasScale returns the device pixel ratio so go-chart can render at native
// resolution and stay crisp on HiDPI displays. Defaults to 1 (e.g. headless tests).
func canvasScale(o fyne.CanvasObject) float32 {
	if app := fyne.CurrentApp(); app != nil {
		if c := app.Driver().CanvasForObject(o); c != nil {
			if s := c.Scale(); s > 0 {
				return s
			}
		}
	}
	return 1
}

// rasterImage runs a draw function against a fresh go-chart raster renderer and
// returns the resulting (transparent-background) image.
func rasterImage(w, h int, draw func(r gchart.Renderer)) image.Image {
	if w < 1 || h < 1 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	r, err := gchart.PNG(w, h)
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	draw(r)
	iw := &gchart.ImageWriter{}
	_ = r.Save(iw)
	img, err := iw.Image()
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	return img
}

// newPlotImage builds the canvas.Image that holds a chart's rendered series. It
// stretches to the plot box and smooths when scaled.
func newPlotImage() *canvas.Image {
	img := canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	img.FillMode = canvas.ImageFillStretch
	img.ScaleMode = canvas.ImageScaleSmooth
	return img
}

// ---- ticks (themed Y-axis gutter, drawn natively so they stay crisp) ----

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

// tooltipAt shows tip i just below-right of the cursor. cursor is the event's
// AbsolutePosition (already canvas-relative), so this is robust to scrolling — no
// need to reconstruct the widget's position.
func tooltipAt(tips []string, i int, cursor fyne.Position) {
	if i < 0 || i >= len(tips) {
		HideTooltip()
		return
	}
	ShowTooltip(tips[i], cursor.Add(fyne.NewPos(12, 14)))
}

// ---- grouped bar chart (two series over the same buckets) ----

type barChart struct {
	widget.BaseWidget
	a, b       []int
	max        int
	colA, colB color.Color
	tips       []string

	img       *canvas.Image
	highlight *canvas.Rectangle
	base      *canvas.Rectangle
	ticks     []tick
	tickVals  []int
	hover     int

	renderedW, renderedH, renderedScale float32
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
	c.img = newPlotImage()
	c.highlight = canvas.NewRectangle(withAlpha(colPrimary, 0x1c))
	c.highlight.CornerRadius = radiusSm
	c.highlight.Hide()
	c.base = canvas.NewRectangle(colBorder)
	c.ticks, c.tickVals = newTicks([]int{0, max / 2, max}, yfmt)
	c.ExtendBaseWidget(c)
	return c
}

func (c *barChart) CreateRenderer() fyne.WidgetRenderer {
	objs := []fyne.CanvasObject{c.highlight}
	for _, tk := range c.ticks {
		objs = append(objs, tk.line)
	}
	objs = append(objs, c.base, c.img)
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

// renderBars draws the grouped bars with go-chart at native resolution. The plot
// box fills the image edge to edge, so the Fyne overlay maps cursor → bucket with
// the same plotWidth/n geometry used here.
func (c *barChart) renderBars(plotW, h, scale float32) image.Image {
	pw, ph := int(plotW*scale), int(h*scale)
	n := len(c.a)
	return rasterImage(pw, ph, func(r gchart.Renderer) {
		if n == 0 || c.max <= 0 {
			return
		}
		baseline := float64(h*scale) - 1
		plotH := baseline - float64(chartTopPad*scale)
		groupW := float64(pw) / float64(n)
		barW := groupW * 0.30
		gap := groupW * 0.06
		rad := 5 * float64(scale)
		bar := func(left, val float64, col color.Color) {
			if val <= 0 {
				return
			}
			top := baseline - plotH*val/float64(c.max)
			drawRoundedBar(r, left, top, left+barW, baseline, rad, dcolor(col))
		}
		for i := 0; i < n; i++ {
			cx := groupW*float64(i) + groupW/2
			bar(cx-barW-gap/2, float64(c.a[i]), c.colA)
			if i < len(c.b) {
				bar(cx+gap/2, float64(c.b[i]), c.colB)
			}
		}
	})
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

	scale := canvasScale(c)
	if size.Width != c.renderedW || size.Height != c.renderedH || scale != c.renderedScale {
		c.img.Image = c.renderBars(c.plotWidth(size.Width), size.Height, scale)
		c.img.Refresh()
		c.renderedW, c.renderedH, c.renderedScale = size.Width, size.Height, scale
	}
	c.img.Move(fyne.NewPos(ChartGutter, 0))
	c.img.Resize(fyne.NewSize(c.plotWidth(size.Width), size.Height))

	if c.hover >= 0 && c.hover < n {
		groupW := c.groupWidth(size.Width)
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

func (c *barChart) MouseIn(ev *desktop.MouseEvent) { c.MouseMoved(ev) }
func (c *barChart) MouseMoved(ev *desktop.MouseEvent) {
	c.update(c.bucketAt(ev.Position.X), ev.AbsolutePosition)
}
func (c *barChart) MouseOut() {
	c.hover = -1
	c.highlight.Hide()
	canvas.Refresh(c)
	HideTooltip()
}

func (c *barChart) update(i int, cursor fyne.Position) {
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
	tooltipAt(c.tips, i, cursor)
}

// drawRoundedBar fills a bar with rounded top corners using the go-chart renderer.
func drawRoundedBar(r gchart.Renderer, left, top, right, bottom, rad float64, fill drawing.Color) {
	if w := (right - left) / 2; rad > w {
		rad = w
	}
	if h := bottom - top; rad > h {
		rad = h
	}
	if rad < 0 {
		rad = 0
	}
	r.SetFillColor(fill)
	r.SetStrokeColor(fill)
	r.MoveTo(int(left), int(bottom))
	r.LineTo(int(left), int(top+rad))
	r.QuadCurveTo(int(left), int(top), int(left+rad), int(top))
	r.LineTo(int(right-rad), int(top))
	r.QuadCurveTo(int(right), int(top), int(right), int(top+rad))
	r.LineTo(int(right), int(bottom))
	r.Close()
	r.Fill()
}

// ---- net-worth line chart ----

type lineChart struct {
	widget.BaseWidget
	vals       []int
	pmin, pmax int // padded range used for plotting
	col        color.Color
	tips       []string

	img      *canvas.Image
	guide    *canvas.Line
	marker   *canvas.Circle // focus dot snapped to the hovered point
	base     *canvas.Rectangle
	ticks    []tick
	tickVals []int
	hover    int

	renderedW, renderedH, renderedScale float32
}

func lineChartWidget(vals []int, col color.Color, tips []string, yfmt func(int) string) fyne.CanvasObject {
	c := &lineChart{vals: vals, col: col, tips: tips, hover: -1}
	dataMin, dataMax := dataRange(vals)
	c.pmin, c.pmax = padRange(dataMin, dataMax)
	c.img = newPlotImage()
	c.guide = canvas.NewLine(withAlpha(col, 0x55))
	c.guide.StrokeWidth = 1.5
	c.guide.Hide()
	c.marker = canvas.NewCircle(col)
	c.marker.StrokeColor = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	c.marker.StrokeWidth = 2.5
	c.marker.Hide()
	c.base = canvas.NewRectangle(colBorder)
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

func (c *lineChart) at(i int, size fyne.Size) fyne.Position {
	n := len(c.vals)
	plotW := size.Width - ChartGutter
	x := ChartGutter + linePad
	if n > 1 {
		x = ChartGutter + linePad + (plotW-2*linePad)*float32(i)/float32(n-1)
	}
	return fyne.NewPos(x, c.yOf(c.vals[i], size))
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
	objs := []fyne.CanvasObject{}
	for _, tk := range c.ticks {
		objs = append(objs, tk.line)
	}
	objs = append(objs, c.base, c.img, c.guide, c.marker)
	for _, tk := range c.ticks {
		objs = append(objs, tk.label)
	}
	return &chartRenderer{objs: objs, layout: c.layout, min: func() fyne.Size {
		return fyne.NewSize(ChartGutter+float32(len(c.vals))*28, chartHeight)
	}}
}

// renderLine draws the net-worth line + area fill + point markers with go-chart
// at native resolution. The padded Y range and edge inset match the Fyne overlay
// so the hover crosshair lands on the rendered points.
func (c *lineChart) renderLine(plotW, h, scale float32) image.Image {
	pw, ph := int(plotW*scale), int(h*scale)
	n := len(c.vals)
	if n == 0 || pw < 1 || ph < 1 {
		return image.NewRGBA(image.Rect(0, 0, maxInt(pw, 1), maxInt(ph, 1)))
	}
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i, v := range c.vals {
		xs[i] = float64(i)
		ys[i] = float64(v)
	}
	s := float64(scale)
	// go-chart draws the line + point markers (no fill — we paint a richer
	// vertical gradient underneath ourselves for a modern, dynamic look).
	series := gchart.ContinuousSeries{
		XValues: xs,
		YValues: ys,
		Style: gchart.Style{
			StrokeColor: dcolor(c.col),
			StrokeWidth: 3 * s,
			DotColor:    dcolor(c.col),
			DotWidth:    3.5 * s,
		},
	}
	padL := int(linePad * scale)
	padTop := int(chartTopPad * scale)
	graph := gchart.Chart{
		Width:  pw,
		Height: ph,
		DPI:    gchart.DefaultDPI,
		Background: gchart.Style{
			FillColor:   drawing.ColorTransparent,
			StrokeColor: drawing.ColorTransparent,
			Padding:     gchart.Box{Top: padTop, Left: padL, Right: padL, Bottom: 1},
		},
		Canvas: gchart.Style{
			FillColor:   drawing.ColorTransparent,
			StrokeColor: drawing.ColorTransparent,
		},
		XAxis: gchart.XAxis{
			Style: gchart.Hidden(),
			Range: &gchart.ContinuousRange{Min: 0, Max: float64(maxInt(n-1, 1))},
		},
		YAxis: gchart.YAxis{
			Style: gchart.Hidden(),
			Range: &gchart.ContinuousRange{Min: float64(c.pmin), Max: float64(c.pmax)},
		},
		Series: []gchart.Series{series},
	}
	iw := &gchart.ImageWriter{}
	if err := graph.Render(gchart.PNG, iw); err != nil {
		return image.NewRGBA(image.Rect(0, 0, pw, ph))
	}
	lineImg, err := iw.Image()
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, pw, ph))
	}

	// Pixel geometry of each point, matching go-chart's canvas box, so the
	// gradient hugs the line exactly.
	left, right := float64(padL), float64(pw-padL)
	top, bottom := float64(padTop), float64(ph-1)
	px := make([]float64, n)
	py := make([]float64, n)
	rng := float64(c.pmax - c.pmin)
	if rng <= 0 {
		rng = 1
	}
	for i := 0; i < n; i++ {
		if n > 1 {
			px[i] = left + (right-left)*float64(i)/float64(n-1)
		} else {
			px[i] = (left + right) / 2
		}
		py[i] = bottom - (bottom-top)*(ys[i]-float64(c.pmin))/rng
	}

	final := image.NewRGBA(image.Rect(0, 0, pw, ph))
	drawGradientFill(final, px, py, bottom, c.col)
	draw.Draw(final, final.Bounds(), lineImg, image.Point{}, draw.Over)
	return final
}

// drawGradientFill paints a vertical gradient below the polyline (px,py): opaque
// near the line, fading to transparent at the baseline. This is the hallmark
// "area chart" look that reads as modern and lively.
func drawGradientFill(dst *image.RGBA, px, py []float64, baseline float64, col color.Color) {
	n := len(px)
	if n < 2 {
		return
	}
	const maxAlpha = 0x60 // ~38% near the line
	r16, g16, b16, _ := col.RGBA()
	cr, cg, cb := float64(r16>>8), float64(g16>>8), float64(b16>>8)
	b := dst.Bounds()
	bot := int(baseline)
	if bot > b.Max.Y-1 {
		bot = b.Max.Y - 1
	}
	for i := 0; i < n-1; i++ {
		x0, x1 := px[i], px[i+1]
		y0, y1 := py[i], py[i+1]
		for x := int(math.Ceil(x0)); x <= int(math.Floor(x1)) && x < b.Max.X; x++ {
			if x < 0 {
				continue
			}
			t := 0.0
			if x1 > x0 {
				t = (float64(x) - x0) / (x1 - x0)
			}
			lineY := y0 + (y1-y0)*t
			top := int(math.Ceil(lineY))
			if top < 0 {
				top = 0
			}
			span := float64(bot - top)
			if span <= 0 {
				continue
			}
			for y := top; y <= bot; y++ {
				frac := float64(y-top) / span // 0 at line → 1 at baseline
				a := maxAlpha * (1 - frac)
				if a <= 0 {
					continue
				}
				af := a / 255
				// Source-over onto a transparent canvas, premultiplied.
				dst.SetRGBA(x, y, color.RGBA{
					R: uint8(cr * af),
					G: uint8(cg * af),
					B: uint8(cb * af),
					A: uint8(a),
				})
			}
		}
	}
}

func (c *lineChart) layout(size fyne.Size) {
	n := len(c.vals)
	if n == 0 {
		return
	}
	placeTicks(c.ticks, c.tickVals, size, func(v int) float32 { return c.yOf(v, size) })
	c.base.Resize(fyne.NewSize(size.Width-ChartGutter, 1))
	c.base.Move(fyne.NewPos(ChartGutter, size.Height-1))

	scale := canvasScale(c)
	if size.Width != c.renderedW || size.Height != c.renderedH || scale != c.renderedScale {
		c.img.Image = c.renderLine(size.Width-ChartGutter, size.Height, scale)
		c.img.Refresh()
		c.renderedW, c.renderedH, c.renderedScale = size.Width, size.Height, scale
	}
	c.img.Move(fyne.NewPos(ChartGutter, 0))
	c.img.Resize(fyne.NewSize(size.Width-ChartGutter, size.Height))

	if c.hover >= 0 && c.hover < n {
		c.placeFocus(c.hover, size)
	}
}

// placeFocus snaps the crosshair guide and the focus marker to point i.
func (c *lineChart) placeFocus(i int, size fyne.Size) {
	p := c.at(i, size)
	c.guide.Position1 = fyne.NewPos(p.X, 0)
	c.guide.Position2 = fyne.NewPos(p.X, size.Height)
	const r = 5
	c.marker.Move(fyne.NewPos(p.X-r, p.Y-r))
	c.marker.Resize(fyne.NewSize(2*r, 2*r))
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

func (c *lineChart) MouseIn(ev *desktop.MouseEvent) { c.MouseMoved(ev) }
func (c *lineChart) MouseMoved(ev *desktop.MouseEvent) {
	c.update(c.nearest(ev.Position.X), ev.AbsolutePosition)
}
func (c *lineChart) MouseOut() {
	c.hover = -1
	c.guide.Hide()
	c.marker.Hide()
	canvas.Refresh(c)
	HideTooltip()
}

func (c *lineChart) update(i int, cursor fyne.Position) {
	if i != c.hover {
		c.hover = i
		if i >= 0 {
			c.placeFocus(i, c.Size())
			c.guide.Show()
			c.marker.Show()
		} else {
			c.guide.Hide()
			c.marker.Hide()
		}
		canvas.Refresh(c)
	}
	tooltipAt(c.tips, i, cursor)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
