package component

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func hoverCapture(t *testing.T) (*string, *bool) {
	t.Helper()
	var got string
	var shown bool
	ShowTooltip = func(s string, _ fyne.Position) { got, shown = s, true }
	HideTooltip = func() { shown = false }
	t.Cleanup(func() {
		ShowTooltip = func(string, fyne.Position) {}
		HideTooltip = func() {}
	})
	return &got, &shown
}

func moveTo(o fyne.CanvasObject, x, y float32) {
	o.(desktop.Hoverable).MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(x, y)}})
}

func TestBarChartHoverTooltip(t *testing.T) {
	test.NewApp()
	got, shown := hoverCapture(t)

	tips := []string{"Jan", "Feb", "Mar", "Apr"}
	idfmt := func(v int) string { return "" }
	obj := barPairChart([]int{10, 20, 30, 40}, []int{5, 5, 5, 5}, colPositive, colNegative, tips, idfmt)
	c := obj.(*barChart)
	c.Resize(fyne.NewSize(454, 150)) // gutter 54 + plot 400 → 4 buckets, groupW 100

	moveTo(obj, 54+250, 75) // bucket 2 (plot x in [200,300))
	if !*shown || *got != "Mar" {
		t.Fatalf("hover tip = %q shown=%v, want Mar", *got, *shown)
	}
	if c.hover != 2 {
		t.Fatalf("hover index = %d, want 2", c.hover)
	}
	if !c.highlight.Visible() {
		t.Fatal("highlight should show on hover")
	}

	c.MouseOut()
	if c.hover != -1 || c.highlight.Visible() || *shown {
		t.Fatal("MouseOut should clear hover, hide highlight, hide tooltip")
	}
}

func TestLineChartHoverPicksNearestPoint(t *testing.T) {
	test.NewApp()
	got, shown := hoverCapture(t)

	tips := []string{"p0", "p1", "p2", "p3"}
	idfmt := func(v int) string { return "" }
	obj := lineChartWidget([]int{100, 200, 150, 300}, colPrimary, tips, idfmt)
	c := obj.(*lineChart)
	c.Resize(fyne.NewSize(454, 150)) // gutter 54 + plot 400; points at plot x ≈ 6,135,264,394

	moveTo(obj, 54+260, 40) // closest to p2 (plot x≈264)
	if !*shown || *got != "p2" {
		t.Fatalf("hover tip = %q shown=%v, want p2", *got, *shown)
	}
	if !c.guide.Visible() {
		t.Fatal("guide line should show on hover")
	}

	moveTo(obj, 54+8, 40) // closest to p0
	if *got != "p0" {
		t.Fatalf("hover tip = %q, want p0", *got)
	}
}
