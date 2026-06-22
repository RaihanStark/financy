// Package view holds the application's screens (Accounts, Transactions) and the
// Preferences dialog, plus their entry forms. It depends on core (data) and the
// component/style packages (presentation); the running app injects the data
// store and navigation callbacks via Init so there's no import cycle.
package view

import (
	"fyne.io/fyne/v2"

	"github.com/raihanstark/financy/internal/core"
	"github.com/raihanstark/financy/internal/ui/component"
	"github.com/raihanstark/financy/internal/ui/style"
)

// Injected services, set by ui.Run via Init.
var (
	store  *core.Store
	nav    func(string)
	render func()
	win    fyne.Window
)

// Init wires the view package to the running application.
func Init(s *core.Store, navigate func(string), rerender func(), w fyne.Window) {
	store, nav, render, win = s, navigate, rerender, w
}

// ---- domain type aliases ----

type (
	Account     = core.Account
	Posting     = core.Posting
	Transaction = core.Transaction
	RegisterRow = core.RegisterRow
	AcctType    = core.AcctType
)

const (
	Asset     = core.Asset
	Liability = core.Liability
	Income    = core.Income
	Expense   = core.Expense
	Equity    = core.Equity
)

// ---- core helpers ----

var (
	itoa            = core.Itoa
	fmtMoney        = core.FmtMoney
	fmtSerialDate   = core.FmtSerialDate
	fmtSerialMonth  = core.FmtSerialMonth
	serialToTime    = core.SerialToTime
	parseAmount     = core.ParseAmount
	amountToInput   = core.AmountToInput
	parseDateSerial = core.ParseDateSerial
	slugify         = core.Slugify
	namesOf         = core.NamesOf
	p               = core.P
	todaySerial     = core.TodaySerial
	version         = core.Version
)

// ---- component helpers ----

type (
	tableSpec     = component.TableSpec
	cell          = component.Cell
	columnsLayout = component.ColumnsLayout
)

var (
	txt             = component.Txt
	mono            = component.Mono
	panel           = component.Panel
	statCard        = component.StatCard
	badge           = component.Badge
	progressBar     = component.ProgressBar
	hbar            = component.Hbar
	divider         = component.Divider
	spacerH         = component.SpacerH
	spacerW         = component.SpacerW
	sectionTitle    = component.SectionTitle
	pageHeader      = component.PageHeader
	moneyColor      = component.MoneyColor
	withAlpha       = component.WithAlpha
	alignRight      = component.AlignRight
	appBar          = component.AppBar
	primaryButton   = component.PrimaryButton
	secondaryButton = component.SecondaryButton
	detailField     = component.DetailField
	alertBanner     = component.AlertBanner
	emptyState      = component.EmptyState
	pillStat        = component.PillStat
	padCell         = component.PadCell
	newTappableRow  = component.NewTappableRow
	tc              = component.Tc
	tcc             = component.Tcc
	tcb             = component.Tcb
	tco             = component.Tco
)

// ---- palette ----

var (
	colBG        = style.BG
	colSurface   = style.Surface
	colSurfaceHi = style.SurfaceHi
	colAltRow    = style.AltRow
	colBorder    = style.Border
	colText      = style.Text
	colTextDim   = style.TextDim
	colPrimary   = style.Primary
	colPositive  = style.Positive
	colNegative  = style.Negative
	colWarning   = style.Warning
)
