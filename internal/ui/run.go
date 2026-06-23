package ui

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/raihanstark/financy/internal/ui/style"
	"github.com/raihanstark/financy/internal/ui/view"
)

// node describes one entry in the navigation registry.
type node struct {
	label string
	icon  fyne.Resource
	build func() fyne.CanvasObject
}

var nav map[string]node

func initNav() {
	nav = map[string]node{
		"accounts":     {label: "Accounts", icon: iconAccounts(), build: view.ScreenAccounts},
		"transactions": {label: "Transactions", icon: iconTransactions(), build: view.ScreenTransactions},
		"analytics":    {label: "Analytics", icon: iconAnalytics(), build: view.ScreenAnalytics},
	}
}

// Run is the application entry point, invoked from package main. icon is the
// application/window icon (may be nil).
func Run(icon fyne.Resource) {
	initNav()

	if len(os.Args) > 1 && os.Args[1] == "shot" {
		out := "."
		if len(os.Args) > 2 {
			out = os.Args[2]
		}
		store = newStore()
		runShots(out)
		return
	}

	a := app.NewWithID("app.financy")
	a.Settings().SetTheme(style.Theme{})
	if icon != nil {
		a.SetIcon(icon)
	}

	w := a.NewWindow(appTitle(""))
	if icon != nil {
		w.SetIcon(icon)
	}
	w.Resize(fyne.NewSize(1320, 860))

	ctl = &appController{win: w}
	w.SetContent(assembleShell(ctl))
	w.SetMainMenu(buildMainMenu())

	startup() // opens last-used or a default document, then renders

	w.ShowAndRun()
}

// buildMainMenu builds the single File menu (no Edit/View/Tools/Help).
func buildMainMenu() *fyne.MainMenu {
	var recentItems []*fyne.MenuItem
	if prefs != nil {
		for _, r := range prefs.Recent {
			r := r
			recentItems = append(recentItems, fyne.NewMenuItem(filepath.Base(r), func() { openDocumentAt(r) }))
		}
	}
	if len(recentItems) == 0 {
		none := fyne.NewMenuItem("No recent files", func() {})
		none.Disabled = true
		recentItems = []*fyne.MenuItem{none}
	}
	openRecent := fyne.NewMenuItem("Open Recent", nil)
	openRecent.ChildMenu = fyne.NewMenu("", recentItems...)

	file := fyne.NewMenu("File",
		fyne.NewMenuItem("New…", doNew),
		fyne.NewMenuItem("Open…", doOpen),
		openRecent,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Setup Wizard…", showSetup),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Save a Copy…", doSaveCopy),
		fyne.NewMenuItem("Close", doClose),
	)
	return fyne.NewMainMenu(file)
}

func refreshMenu() {
	if ctl != nil && ctl.win != nil {
		ctl.win.SetMainMenu(buildMainMenu())
	}
}
