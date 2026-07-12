package ui

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"

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
		"budget":       {label: "Budget", icon: iconBudget(), build: view.ScreenBudget},
		"analytics":    {label: "Analytics", icon: iconAnalytics(), build: view.ScreenAnalytics},
		"recurring":    {label: "Recurring", icon: iconRecurring(), build: view.ScreenRecurring},
		"debts":        {label: "Debts", icon: iconDebts(), build: view.ScreenDebts},
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
	// Restore the saved light/dark choice before any UI is built so the first
	// render is already in the right palette.
	prefs = loadPrefs()
	applyPalette(prefs.Dark)
	a.Settings().SetTheme(style.Theme{})
	if icon != nil {
		a.SetIcon(icon)
	}

	w := a.NewWindow(appTitle(""))
	if icon != nil {
		w.SetIcon(icon)
	}
	w.Resize(fyne.NewSize(1420, 880))

	ctl = &appController{win: w}
	w.SetContent(assembleShell(ctl))
	w.SetMainMenu(buildMainMenu())

	// Cmd/Ctrl-S saves an encrypted document; closing with unsaved changes prompts.
	w.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyS,
		Modifier: fyne.KeyModifierShortcutDefault,
	}, func(fyne.Shortcut) { doSave() })
	w.SetCloseIntercept(func() { guardUnsaved(func() { w.Close() }) })

	startup() // opens last-used or a default document, then renders

	startDueTicker()            // date rollover + newly due recurring entries
	checkForUpdatesAsync(false) // silent, throttled background check

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

	// Save is only meaningful for encrypted documents (plain files auto-save).
	saveItem := fyne.NewMenuItem("Save", doSave)
	saveItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault}
	saveItem.Disabled = store == nil || !store.Encrypted()

	pwLabel := "Set Password…"
	if store != nil && store.Encrypted() {
		pwLabel = "Change Password…"
	}
	setPwItem := fyne.NewMenuItem(pwLabel, doSetPassword)
	setPwItem.Disabled = store == nil
	removePwItem := fyne.NewMenuItem("Remove Password…", doRemovePassword)
	removePwItem.Disabled = store == nil || !store.Encrypted()

	file := fyne.NewMenu("File",
		fyne.NewMenuItem("New…", doNew),
		fyne.NewMenuItem("Open…", doOpen),
		openRecent,
		fyne.NewMenuItemSeparator(),
		saveItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Import CSV…", func() {
			if store != nil {
				view.ImportCSV()
			}
		}),
		fyne.NewMenuItem("Export CSV…", doExportCSV),
		fyne.NewMenuItem("Setup Wizard…", showSetup),
		fyne.NewMenuItemSeparator(),
		setPwItem,
		removePwItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Check for Updates…", func() { checkForUpdatesAsync(true) }),
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
