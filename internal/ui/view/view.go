// Package view holds the application's screens (Accounts, Transactions) and the
// Preferences dialog, plus their entry forms. It depends on core (data) and the
// component/style packages (presentation); the running app injects the data
// store and navigation callbacks via Init so there's no import cycle.
package view

import (
	"image/color"

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

	AnalyticsSummary = core.AnalyticsSummary
	MonthFlow        = core.MonthFlow
	NetWorthPoint    = core.NetWorthPoint
	CategorySlice    = core.CategorySlice
	Recurring        = core.Recurring
	DueItem          = core.DueItem
	Debt             = core.Debt
	Installment      = core.Installment
	DebtStatus       = core.DebtStatus
	PayoffProjection = core.PayoffProjection
	StatementDue     = core.StatementDue
	StrategyResult   = core.StrategyResult
	DebtPayoff       = core.DebtPayoff
	DebtOverview     = core.DebtOverview
	DebtOverviewRow  = core.DebtOverviewRow
	BudgetMonth      = core.BudgetMonth
	BudgetCategory   = core.BudgetCategory
)

const (
	Asset     = core.Asset
	Liability = core.Liability
	Income    = core.Income
	Expense   = core.Expense
	Equity    = core.Equity

	DebtBNPL      = core.DebtBNPL
	DebtLoan      = core.DebtLoan
	DebtRevolving = core.DebtRevolving
	DebtInformal  = core.DebtInformal
)

// ---- core helpers ----

var (
	itoa                 = core.Itoa
	fmtMoney             = core.FmtMoney
	fmtMoneyPlain        = core.FmtMoneyPlain
	fmtMoneyShort        = core.FmtMoneyShort
	fmtSerialDate        = core.FmtSerialDate
	fmtSerialMonth       = core.FmtSerialMonth
	serialToTime         = core.SerialToTime
	timeToSerial         = core.TimeToSerial
	parseAmount          = core.ParseAmount
	fmtMoneyInput        = core.FmtMoneyInput
	groupTyped           = core.GroupTyped
	numberFormatStyles   = core.NumberFormatStyles
	parseDateSerial      = core.ParseDateSerial
	slugify              = core.Slugify
	groupedLabels        = core.GroupedLabels
	firstSelectable      = core.FirstSelectable
	isGroupHeader        = core.IsGroupHeader
	accountLabel         = core.AccountLabel
	p                    = core.P
	postingsFor          = core.PostingsFor
	frequencies          = core.Frequencies
	debtTypes            = core.DebtTypes
	generateInstallments = core.GenerateInstallments
	generateLoanSchedule = core.GenerateLoanSchedule
	amortizedPayment     = core.AmortizedPayment
	scheduleInterest     = core.ScheduleInterestTotal
	periodInterest       = core.PeriodInterest
	parseAPRBps          = core.ParseAPRBps
	fmtAPRBps            = core.FmtAPRBps
	todaySerial          = core.TodaySerial
	version              = core.Version

	currentMonthKey = core.CurrentMonthKey
	shiftMonth      = core.ShiftMonth
	fmtMonthLong    = core.FmtMonthLong
	monthEditable   = core.MonthEditable
)

// ---- component helpers ----

type (
	tableSpec     = component.TableSpec
	cell          = component.Cell
	columnsLayout = component.ColumnsLayout
	modal         = component.Modal
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
	badge           = component.Badge
	alertBanner     = component.AlertBanner
	progressBar     = component.ProgressBar
	pillStat        = component.PillStat
	padCell         = component.PadCell
	newTappableRow  = component.NewTappableRow
	barPairChart    = component.BarPairChart
	lineChart       = component.LineChart
	chartGutter     = component.ChartGutter
	tc              = component.Tc
	tcc             = component.Tcc
	tco             = component.Tco
)

// ---- palette ----
//
// Local copies of the active palette kept concise. SyncPalette refreshes them
// after a theme switch so freshly built screens pick up the new colors.

var (
	colSurface   color.NRGBA
	colSurfaceHi color.NRGBA
	colAltRow    color.NRGBA
	colBorder    color.NRGBA
	colText      color.NRGBA
	colTextDim   color.NRGBA
	colPrimary   color.NRGBA
	colPositive  color.NRGBA
	colNegative  color.NRGBA
	colWarning   color.NRGBA
)

func init() { SyncPalette() }

// SyncPalette refreshes the local palette aliases from the active style palette.
func SyncPalette() {
	colSurface = style.Surface
	colSurfaceHi = style.SurfaceHi
	colAltRow = style.AltRow
	colBorder = style.Border
	colText = style.Text
	colTextDim = style.TextDim
	colPrimary = style.Primary
	colPositive = style.Positive
	colNegative = style.Negative
	colWarning = style.Warning
}

const (
	radius   = style.Radius
	radiusSm = style.RadiusSm
)
