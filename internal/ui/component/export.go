package component

// Exported API of the component package. The implementations stay lowercase so
// intra-package code reads cleanly; these aliases expose them to view/ui.

var (
	Txt               = txt
	Mono              = mono
	Panel             = panel
	GroupBox          = groupBox
	StatCard          = statCard
	Badge             = badge
	ProgressBar       = progressBar
	Hbar              = hbar
	Divider           = divider
	SpacerH           = spacerH
	SpacerW           = spacerW
	SectionTitle      = sectionTitle
	PageHeader        = pageHeader
	MoneyColor        = moneyColor
	WithAlpha         = withAlpha
	AlignRight        = alignRight
	CanvasTransparent = canvasTransparent

	AppBar          = appBar
	PrimaryButton   = primaryButton
	SecondaryButton = secondaryButton
	DetailField     = detailField
	AlertBanner     = alertBanner
	EmptyState      = emptyState
	PillStat        = pillStat
	PadCell         = padCell
	NewTappableRow  = newTappableRow
)
