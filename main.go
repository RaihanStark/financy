package main

import (
	_ "embed"

	"fyne.io/fyne/v2"

	"github.com/raihanstark/financy/internal/ui"
)

//go:embed Icon.png
var iconPNG []byte

func main() {
	ui.Run(fyne.NewStaticResource("Icon.png", iconPNG))
}
