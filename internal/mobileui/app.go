// Package mobileui is Financy's touch-first mobile UI. It is a ground-up
// redesign that reuses only internal/core (the domain/data layer) and shares no
// code with the desktop shell in internal/ui. The screens — a dashboard Home, a
// recycled Transactions list, Budget and Debts — are built for small screens
// and one-handed use: a bottom tab bar, a floating add button, and a single
// auto-saved document in the app sandbox.
package mobileui

import (
	"sync"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	mobiledriver "fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"

	"github.com/raihanstark/financy/internal/core"
)

type mobileApp struct {
	fyneApp fyne.App
	win     fyne.Window
	store   *core.Store

	shell       fyne.CanvasObject // the persistent nav+content+fab; shown once a doc is open
	content     *fyne.Container   // swappable screen host
	fabWrap     *fyne.Container    // the FAB layer; rebuilt to expand/collapse the speed dial
	fabOpen     bool               // whether the Transactions speed dial is expanded
	nav         *navBar
	current     int
	budgetMonth string              // "YYYY-MM" shown on the Budget tab (empty = current month)
	navStack    []fyne.CanvasObject // previous full-screen views, for the Back button

	// Session write-back: when a document was opened from (or exported to) an
	// external file, sourceURI is that file and edits are copied back to it so
	// it stays in sync (e.g. a file in a cloud-synced location). Android grants
	// this access only for the session, so it resets on restart.
	sourceURI   fyne.URI
	syncMu      sync.Mutex
	syncPending bool
}

// ctl is the single live controller (the app is a singleton window).
var ctl *mobileApp

// Run builds the mobile UI and starts the event loop. icon may be nil.
func Run(icon fyne.Resource) {
	a := fyneapp.NewWithID("app.financy")
	a.Settings().SetTheme(appTheme{})
	if icon != nil {
		a.SetIcon(icon)
	}

	w := a.NewWindow("Financy")
	if icon != nil {
		w.SetIcon(icon)
	}
	w.Resize(fyne.NewSize(390, 844)) // ignored on device; sizes the desktop preview

	ctl = &mobileApp{fyneApp: a, win: w}
	ctl.build()

	// Wire the Android Back button to in-app navigation. Fyne routes Back to the
	// canvas's typed-key handler when nothing is focused; without this it would
	// finish the activity (close the app) even on a drill-down page.
	w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name != mobiledriver.KeyBack {
			return
		}
		// If a popup is open (a Select dropdown, a dialog, the calendar picker),
		// Back should dismiss it — not navigate the page — matching Android's
		// expectation. Overlays sit on the canvas's overlay stack.
		if ov := w.Canvas().Overlays().Top(); ov != nil {
			w.Canvas().Overlays().Remove(ov)
			return
		}
		ctl.back()
	})

	ctl.startup()
	w.ShowAndRun()
}

// pushView shows page as a new full-screen level, remembering the current view
// so back() can return to it (via the ✕/← control or the Android Back button).
func (m *mobileApp) pushView(page fyne.CanvasObject) {
	m.navStack = append(m.navStack, m.win.Content())
	m.win.SetContent(page)
}

// replaceView swaps the current level's content without changing the stack
// (used to rebuild a detail page in place after a mutation).
func (m *mobileApp) replaceView(page fyne.CanvasObject) { m.win.SetContent(page) }

// back pops one full-screen level. At the root it asks the OS to background the
// app (the normal Android Back-at-home behavior) — NOT app.Quit(), which the
// mobile driver refuses and which leaves the UI frozen.
func (m *mobileApp) back() {
	n := len(m.navStack)
	if n == 0 {
		if drv, ok := m.fyneApp.Driver().(mobiledriver.Driver); ok {
			drv.GoBack()
		}
		return
	}
	prev := m.navStack[n-1]
	m.navStack = m.navStack[:n-1]
	m.win.SetContent(prev)
}

// showShell returns to the tabbed shell and clears the navigation stack (used
// after opening a document, so Back from a tab exits rather than replays pages).
func (m *mobileApp) showShell() {
	m.navStack = nil
	if m.shell != nil {
		m.win.SetContent(m.shell)
	}
}

// build assembles the persistent shell: a content area, a bottom tab bar, and a
// floating add button layered on top.
func (m *mobileApp) build() {
	m.content = container.NewStack()
	m.nav = newNavBar(m.selectTab)

	body := container.NewStack(canvas.NewRectangle(colBg), m.content)
	root := container.NewBorder(nil, m.nav.bar, nil, nil, body)

	m.fabWrap = container.NewStack()
	m.rebuildFab()

	m.shell = container.NewStack(root, m.fabWrap)
	m.win.SetContent(m.shell)
}

// fabTapped handles the floating +. On the Transactions tab it expands a speed
// dial (Add / Bulk add); on the other tabs it runs that tab's single add action.
func (m *mobileApp) fabTapped() {
	if m.store == nil {
		return
	}
	if m.current == 1 {
		m.toggleFab()
		return
	}
	m.fabAction()
}

// fabAction routes the floating + to the "add" that fits the current tab:
// Home → account, Debts → debt, otherwise → transaction.
func (m *mobileApp) fabAction() {
	if m.store == nil {
		return
	}
	switch m.current {
	case 0: // Home
		m.addAccount()
	case 2: // Budget
		m.addCategory()
	case 3: // Debts
		m.addDebt()
	default: // Transactions
		m.openAdd(nil)
	}
}

// toggleFab flips the speed dial open/closed; closeFab only collapses it (called
// before pushing a page so the shell is collapsed again when the user returns).
func (m *mobileApp) toggleFab() { m.fabOpen = !m.fabOpen; m.rebuildFab() }
func (m *mobileApp) closeFab() {
	if m.fabOpen {
		m.fabOpen = false
		m.rebuildFab()
	}
}

// rebuildFab renders the FAB layer: a lone + when collapsed, or a dimming scrim
// plus the stacked "Add / Bulk add" actions when the Transactions speed dial is
// expanded.
func (m *mobileApp) rebuildFab() {
	if m.fabWrap == nil {
		return
	}
	collapsed := insets(
		container.NewVBox(layout.NewSpacer(),
			container.NewHBox(layout.NewSpacer(), newFab(m.fabTapped))),
		0, 104, 0, 18) // clear the tab bar (its top border), inset from the right edge

	if m.current != 1 || !m.fabOpen {
		m.fabWrap.Objects = []fyne.CanvasObject{collapsed}
		m.fabWrap.Refresh()
		return
	}

	scrim := newTappableCard(canvas.NewRectangle(withAlpha(colInk, 0x55)), m.toggleFab)
	actions := insets(
		container.NewVBox(
			layout.NewSpacer(),
			speedDialItem("Bulk add", theme.ListIcon(), 46, func() { m.closeFab(); m.openBulkAdd() }),
			gap(14),
			speedDialItem("Add", theme.ContentAddIcon(), 58, func() { m.closeFab(); m.openAdd(nil) }),
		),
		0, 104, 0, 18)
	m.fabWrap.Objects = []fyne.CanvasObject{scrim, actions}
	m.fabWrap.Refresh()
}

func (m *mobileApp) selectTab(i int) {
	m.current = i
	m.fabOpen = false // never carry an open speed dial across tabs
	if m.nav != nil {
		m.nav.setActive(i)
	}
	m.rebuildFab()
	m.render()
}

// render rebuilds the active tab. Screens are cheap to rebuild; the transaction
// list uses a recycling widget.List so long histories stay smooth.
func (m *mobileApp) render() {
	if m.content == nil {
		return
	}
	var screen fyne.CanvasObject
	switch {
	case m.store == nil:
		screen = container.NewCenter(newText("Loading…", colInkDim, 14, false))
	case m.current == 1:
		screen = m.transactionsScreen()
	case m.current == 2:
		screen = m.budgetScreen()
	case m.current == 3:
		screen = m.debtsScreen()
	default:
		screen = m.homeScreen()
	}
	m.content.Objects = []fyne.CanvasObject{screen}
	m.content.Refresh()
}

func (m *mobileApp) fatal(err error) {
	if m.content == nil || err == nil {
		return
	}
	m.content.Objects = []fyne.CanvasObject{
		container.NewCenter(insets(newText(err.Error(), colNeg, 13, false), 0, 0, 24, 24)),
	}
	m.content.Refresh()
}
