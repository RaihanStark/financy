package component

import "fyne.io/fyne/v2"

// Tooltip sink. Interactive components (e.g. charts) surface hover tooltips
// through these hooks; the host app (package ui) wires them to its floating
// tooltip layer. They default to no-ops so the component package works
// standalone (and in headless screenshot rendering, where there's no hover).
var (
	ShowTooltip = func(text string, pos fyne.Position) {}
	HideTooltip = func() {}
)
