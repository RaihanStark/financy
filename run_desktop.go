//go:build !android && !ios

package main

import (
	"os"

	"fyne.io/fyne/v2"

	"github.com/raihanstark/financy/internal/mobileui"
	"github.com/raihanstark/financy/internal/ui"
)

// run launches the desktop shell. Setting FINANCY_MOBILE=1 previews the mobile
// UI in a phone-sized window on the desktop (handy for development).
func run(icon fyne.Resource) {
	if os.Getenv("FINANCY_MOBILE") == "1" {
		mobileui.Run(icon)
		return
	}
	ui.Run(icon)
}
