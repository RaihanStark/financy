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
	render func()
	win    fyne.Window
)

// Init wires the view package to the running application.
func Init(s *core.Store, rerender func(), w fyne.Window) {
	store, render, win = s, rerender, w
}

// ---- domain type aliases ----

type (
	Account     = core.Account
	Posting     = core.Posting
	Transaction = core.Transaction
	RegisterRow = core.RegisterRow
	AcctType    = core.AcctType

	AnalyticsSummary = core.AnalyticsSummary
	MonthFlow        = core.MonthFlow
	NetWorthPoint    = core.NetWorthPoint
	CategorySlice    = core.CategorySlice
	Recurring        = core.Recurring
	DueItem          = core.DueItem
	IncomeStatement  = core.IncomeStatement
	BalanceSheet     = core.BalanceSheet
	CashFlow         = core.CashFlow
	StmtLine         = core.StmtLine
	CashLine         = core.CashLine
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
	fmtMoneyPlain   = core.FmtMoneyPlain
	fmtMoneyShort   = core.FmtMoneyShort
	fmtSerialDate   = core.FmtSerialDate
	fmtSerialMonth  = core.FmtSerialMonth
	serialToTime    = core.SerialToTime
	timeToSerial    = core.TimeToSerial
	parseAmount     = core.ParseAmount
	fmtMoneyInput      = core.FmtMoneyInput
	groupTyped         = core.GroupTyped
	numberFormatStyles = core.NumberFormatStyles
	parseDateSerial = core.ParseDateSerial
	slugify         = core.Slugify
	namesOf         = core.NamesOf
	p               = core.P
	postingsFor     = core.PostingsFor
	frequencies     = core.Frequencies
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
	hbar            = component.Hbar
	divider         = component.Divider
	spacerH         = component.SpacerH
	spacerW         = component.SpacerW
	sectionTitle    = component.SectionTitle
	moneyColor      = component.MoneyColor
	withAlpha       = component.WithAlpha
	alignRight      = component.AlignRight
	appBar          = component.AppBar
	primaryButton   = component.PrimaryButton
	secondaryButton = component.SecondaryButton
	detailField     = component.DetailField
	emptyState      = component.EmptyState
	padCell         = component.PadCell
	newTappableRow  = component.NewTappableRow
	barPairChart    = component.BarPairChart
	lineChart       = component.LineChart
	chartGutter     = component.ChartGutter
	tc              = component.Tc
	tcc             = component.Tcc
	tcb             = component.Tcb
	tco             = component.Tco
)

// ---- palette ----

var (
	colSurface   = style.Surface
	colSurfaceHi = style.SurfaceHi
	colBorder    = style.Border
	colText      = style.Text
	colTextDim   = style.TextDim
	colPrimary   = style.Primary
	colPositive  = style.Positive
	colNegative  = style.Negative
	colWarning   = style.Warning
)

const (
	radius   = style.Radius
	radiusSm = style.RadiusSm
)
