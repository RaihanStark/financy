package ui

import (
	"image"
	"image/png"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/raihanstark/financy/internal/core"
	"github.com/raihanstark/financy/internal/ui/style"
	"github.com/raihanstark/financy/internal/ui/view"
)

// runShots renders the app headlessly (via the software renderer) and writes
// PNGs used for docs/screenshots. It seeds demo data so the screens look real.
func runShots(outDir string) {
	_ = os.MkdirAll(outDir, 0o755)

	a := test.NewApp()
	a.Settings().SetTheme(style.Theme{})

	c := &appController{}
	ctl = c
	root := assembleShell(c)
	w := test.NewWindow(root)
	c.win = w
	view.Init(store, c.show, c.refresh, w)
	w.Resize(fyne.NewSize(1320, 860))

	capture := func(name string) { writePNG(outDir+"/"+name+".png", w.Canvas().Capture()) }
	dropOverlay := func() { w.Canvas().Overlays().Remove(w.Canvas().Overlays().Top()) }

	// First-run setup dialog (before any data).
	c.renderWelcome()
	showSetup()
	capture("setup")
	dropOverlay()

	// Populate demo data so the screens are illustrative.
	core.SeedDemo(store, "$")

	c.show("accounts")
	capture("accounts")
	c.show("transactions")
	capture("transactions")
	c.show("analytics")
	capture("analytics")
	c.show("recurring")
	capture("recurring")
	c.show("reports")
	capture("reports")

	// Preferences tabs.
	c.show("accounts")
	view.OpenConfig(1)
	capture("categories")
	dropOverlay()
	view.OpenConfig(2)
	capture("data-summary")
	dropOverlay()

	w.Close()
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_ = png.Encode(f, img)
}
