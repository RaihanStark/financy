package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed Icon.png
var iconPNG []byte

func main() {
	// run() is selected by build tags: the desktop shell (internal/ui) on
	// desktop, the touch UI (internal/mobileui) on Android/iOS.
	run(fyne.NewStaticResource("Icon.png", iconPNG))
}
