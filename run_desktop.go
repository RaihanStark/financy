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
		// `FINANCY_MOBILE=1 financy shot <dir>` captures the mobile screens from the
		// real GL renderer (device-accurate), unlike the headless test harness.
		if len(os.Args) > 1 && os.Args[1] == "shot" {
			out := "."
			if len(os.Args) > 2 {
				out = os.Args[2]
			}
			target := "home" // one screen per process (see RunShots)
			if len(os.Args) > 3 {
				target = os.Args[3]
			}
			mobileui.RunShots(icon, out, target)
			return
		}
		mobileui.Run(icon)
		return
	}
	ui.Run(icon)
}
