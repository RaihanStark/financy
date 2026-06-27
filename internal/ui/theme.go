package ui

import (
	"fyne.io/fyne/v2"

	"github.com/raihanstark/financy/internal/ui/component"
	"github.com/raihanstark/financy/internal/ui/style"
	"github.com/raihanstark/financy/internal/ui/view"
)

// applyPalette switches the active palette to dark (on) or light (off) and
// re-syncs every package's local aliases. It does NOT rebuild the UI — callers
// that need already-rendered canvas objects recolored must do that themselves
// (see toggleTheme). Used at startup before the shell is first built.
func applyPalette(dark bool) {
	style.SetDark(dark)
	syncPalette()
	component.SyncPalette()
	view.SyncPalette()
}

// themeToggleLabel is the tooltip for the toolbar theme button — it names the
// mode the button switches to.
func themeToggleLabel() string {
	if style.Dark {
		return "Switch to light mode"
	}
	return "Switch to dark mode"
}

// toggleTheme flips between the light and dark palettes and rebuilds the window
// so every custom canvas widget — which caches its colors at build time — picks
// up the new look. The choice is persisted in prefs.
func toggleTheme() {
	dark := !style.Dark
	applyPalette(dark)

	if app := fyne.CurrentApp(); app != nil {
		// Refresh standard widgets / overlays that read the theme directly.
		app.Settings().SetTheme(style.Theme{})
	}
	if ctl != nil && ctl.win != nil {
		ctl.win.SetContent(assembleShell(ctl))
		ctl.win.SetMainMenu(buildMainMenu())
		ctl.render()
	}

	if prefs != nil {
		prefs.Dark = dark
		prefs.save()
	}
}
