package mobileui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// showWelcome is the no-document landing screen: a friendly header and a set of
// tappable option cards (unlock an existing locked file, open a .financy file,
// try the demo, or start empty). Shown on first run and whenever there's no open
// document (e.g. after cancelling the unlock screen).
func (m *mobileApp) showWelcome() {
	header := container.NewVBox(
		gap(8),
		newText("Financy", colInk, 30, true),
		newText("Personal finance with double-entry clarity.", colInkDim, 13, false),
	)

	opts := container.NewVBox()
	if enc, err := core.IsEncrypted(docPath()); err == nil && enc {
		opts.Add(welcomeOption(theme.LoginIcon(), colPrimary,
			"Unlock your document", "Enter the passphrase for your saved file",
			func() { m.openEncryptedSandbox(docPath()) }))
		opts.Add(gap(12))
	}
	opts.Add(welcomeOption(theme.FolderOpenIcon(), colPrimary,
		"Open a .financy file", "Load one from your device or cloud",
		func() { m.importFile() }))
	opts.Add(gap(12))
	opts.Add(welcomeOption(theme.MediaPlayIcon(), colPos,
		"Try the demo", "Explore with sample data",
		func() { m.replaceDocument(true) }))
	opts.Add(gap(12))
	opts.Add(welcomeOption(theme.DocumentCreateIcon(), colInkDim,
		"Start from empty", "Begin with a fresh, blank document",
		func() { m.replaceDocument(false) }))

	body := container.NewVBox(gap(36), header, gap(24), opts, gap(24))
	page := container.NewStack(
		canvas.NewRectangle(colBg),
		container.NewVScroll(insets(body, 8, 8, 20, 20)),
	)
	m.navStack = nil // welcome is a root screen
	m.win.SetContent(page)
}

// welcomeOption is a tappable card: a tinted icon, a title, and a subtitle.
func welcomeOption(icon fyne.Resource, accent color.NRGBA, title, subtitle string, tap func()) fyne.CanvasObject {
	ic := iconCircle(icon, accent)
	texts := container.NewVBox(newText(title, colInk, 15, true), wrapText(subtitle))
	row := container.NewBorder(nil, nil, ic, nil, insets(texts, 0, 0, 12, 8))
	card := container.NewStack(rounded(colCard, 16), insets(row, 14, 14, 14, 14))
	return newTappableCard(card, tap)
}

// iconCircle renders a theme icon inside a tinted round badge.
func iconCircle(res fyne.Resource, c color.NRGBA) fyne.CanvasObject {
	circle := canvas.NewCircle(withAlpha(c, 0x22))
	ic := widget.NewIcon(theme.NewThemedResource(res))
	return container.NewGridWrap(fyne.NewSize(42, 42),
		container.NewStack(circle, container.NewCenter(ic)))
}

// tappableCard makes an arbitrary object tappable (for option/list cards).
type tappableCard struct {
	widget.BaseWidget
	content fyne.CanvasObject
	tap     func()
}

func newTappableCard(content fyne.CanvasObject, tap func()) *tappableCard {
	c := &tappableCard{content: content, tap: tap}
	c.ExtendBaseWidget(c)
	return c
}

func (c *tappableCard) Tapped(_ *fyne.PointEvent) {
	if c.tap != nil {
		c.tap()
	}
}

func (c *tappableCard) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
}
