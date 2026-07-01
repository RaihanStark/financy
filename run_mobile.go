//go:build android || ios

package main

import (
	"fyne.io/fyne/v2"

	"github.com/raihanstark/financy/internal/mobileui"
)

// run launches the touch UI on mobile builds.
func run(icon fyne.Resource) {
	mobileui.Run(icon)
}
