package ui

import (
	"image/color"

	"github.com/raihanstark/financy/internal/core"
	"github.com/raihanstark/financy/internal/ui/component"
	"github.com/raihanstark/financy/internal/ui/style"
)

// Bridges from the sub-packages so the shell/toolbar code stays concise.

type (
	Store       = core.Store
	Account     = core.Account
	Posting     = core.Posting
	Transaction = core.Transaction
)

const (
	Asset = core.Asset
)

var (
	newStore = core.NewStore
	fmtMoney = core.FmtMoney
)

var (
	txt             = component.Txt
	padCell         = component.PadCell
	withAlpha       = component.WithAlpha
	panel           = component.Panel
	sectionTitle    = component.SectionTitle
	spacerH         = component.SpacerH
	primaryButton   = component.PrimaryButton
	secondaryButton = component.SecondaryButton
)

var (
	colBG        color.NRGBA
	colSurface   color.NRGBA
	colSurfaceHi color.NRGBA
	colBorder    color.NRGBA
	colText      color.NRGBA
	colTextDim   color.NRGBA
	colPrimary   color.NRGBA
)

func init() { syncPalette() }

// syncPalette refreshes this package's palette aliases from the active style
// palette. applyTheme calls it (plus the sub-packages' SyncPalette) on a switch.
func syncPalette() {
	colBG = style.BG
	colSurface = style.Surface
	colSurfaceHi = style.SurfaceHi
	colBorder = style.Border
	colText = style.Text
	colTextDim = style.TextDim
	colPrimary = style.Primary
}

// store is the single live data store, set up in Run.
var store *Store
