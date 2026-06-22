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

// runShots renders the full application headlessly via the software renderer.
func runShots(outDir string) {
	a := test.NewApp()
	a.Settings().SetTheme(style.Theme{})

	c := &appController{}
	ctl = c
	root := assembleShell(c)
	w := test.NewWindow(root)
	c.win = w
	view.Init(store, c.show, c.refresh, w)
	w.Resize(fyne.NewSize(1320, 860))

	// Main screens.
	for _, uid := range []string{"accounts", "transactions"} {
		c.show(uid)
		writePNG(outDir+"/app-"+uid+".png", w.Canvas().Capture())
	}

	// Preferences dialog, one capture per tab.
	c.show("accounts")
	view.OpenConfig(0)
	writePNG(outDir+"/app-config.png", w.Canvas().Capture())
	w.Canvas().Overlays().Remove(w.Canvas().Overlays().Top())
	view.OpenConfig(1)
	writePNG(outDir+"/app-config-categories.png", w.Canvas().Capture())
	w.Canvas().Overlays().Remove(w.Canvas().Overlays().Top())
	view.OpenConfig(2)
	writePNG(outDir+"/app-config-data.png", w.Canvas().Capture())
	w.Canvas().Overlays().Remove(w.Canvas().Overlays().Top())

	// First-run setup dialog.
	c.renderWelcome()
	showSetup()
	writePNG(outDir+"/app-setup.png", w.Canvas().Capture())
	w.Canvas().Overlays().Remove(w.Canvas().Overlays().Top())

	// Demo data (USD) populated.
	core.SeedDemo(store, "$")
	c.show("accounts")
	writePNG(outDir+"/app-demo.png", w.Canvas().Capture())
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
