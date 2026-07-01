package mobileui

import (
	"image/png"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/raihanstark/financy/internal/core"
)

// TestRenderShots renders each mobile screen headlessly (software renderer) to
// PNGs for visual review. It only runs when MOBILE_SHOTS points at an output
// directory: MOBILE_SHOTS=/tmp/x go test -run TestRenderShots ./internal/mobileui/
func TestRenderShots(t *testing.T) {
	outDir := os.Getenv("MOBILE_SHOTS")
	if outDir == "" {
		t.Skip("set MOBILE_SHOTS=<dir> to render screenshots")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	a := test.NewApp()
	a.Settings().SetTheme(appTheme{})
	w := test.NewWindow(nil)
	w.Resize(fyne.NewSize(390, 844))

	m := &mobileApp{fyneApp: a, win: w}
	ctl = m
	m.build()

	s := core.NewStore()
	core.SeedDemo(s, "$")
	m.store = s

	shot := func(name string) {
		w.Content().Refresh()
		f, err := os.Create(outDir + "/m-" + name + ".png")
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, w.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}

	for i, name := range []string{"home", "transactions", "budget", "debts"} {
		m.selectTab(i)
		shot(name)
	}

	// The debt detail page.
	m.selectTab(3)
	if debts := m.store.Debts(); len(debts) > 0 {
		m.openDebt(debts[0])
		shot("debt")
	}

	// The add-transaction full-screen form.
	m.selectTab(0)
	m.openAdd(nil)
	shot("addmodal")

	// The account detail page.
	m.selectTab(0)
	if accts := m.store.MoneyAccounts(); len(accts) > 0 {
		m.openAccount(accts[0])
		shot("account")
	}

	// The transaction detail page.
	m.selectTab(0)
	if txns := sortedTxns(m.store); len(txns) > 0 {
		m.openTransaction(txns[0])
		shot("transaction")
	}

	// The full-screen unlock page.
	m.showUnlockPage("Unlock document",
		"This document is password-protected. Enter its passphrase to open it.",
		func(string) (*core.Store, error) { return nil, nil }, func() {})
	shot("unlock")

	// The welcome / no-document screen.
	m.showWelcome()
	shot("welcome")

	// The full-screen settings page.
	m.openSettings()
	shot("settings")
}
