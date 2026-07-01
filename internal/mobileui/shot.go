//go:build !android && !ios

package mobileui

import (
	"image"
	"image/png"
	"os"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	"github.com/raihanstark/financy/internal/core"
)

// RunShots launches the real, GL-rendered mobile UI, navigates straight to a
// single target screen, captures a device-accurate PNG (m-<target>.png) into
// outDir, and quits. It captures ONE screen per process on purpose: the glfw
// back-buffer only reads back cleanly on the first Capture() of a session (later
// captures invert alpha on this driver), so the harness relaunches the binary
// once per screen. Needs a display (won't run on a headless CI box).
func RunShots(icon fyne.Resource, outDir, target string) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}

	a := fyneapp.NewWithID("app.financy.shots")
	a.Settings().SetTheme(appTheme{})
	if icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("Financy")
	if icon != nil {
		w.SetIcon(icon)
	}
	w.Resize(fyne.NewSize(390, 844))

	m := &mobileApp{fyneApp: a, win: w}
	ctl = m
	m.build()
	s := core.NewStore()
	core.SeedDemo(s, "$")
	m.store = s

	nav := map[string]func(){
		"home":         func() { m.selectTab(0) },
		"transactions": func() { m.selectTab(1) },
		"budget":       func() { m.selectTab(2) },
		"debts":        func() { m.selectTab(3) },
		"debt": func() {
			m.selectTab(3)
			if d := m.store.Debts(); len(d) > 0 {
				m.openDebt(d[0])
			}
		},
		"account": func() {
			m.selectTab(0)
			if ac := m.store.MoneyAccounts(); len(ac) > 0 {
				m.openAccount(ac[0])
			}
		},
		"addmodal": func() { m.selectTab(0); m.openAdd(nil) },
		"transaction": func() {
			m.selectTab(0)
			if t := sortedTxns(m.store); len(t) > 0 {
				m.openTransaction(t[0])
			}
		},
		"settings": func() { m.selectTab(0); m.openSettings() },
	}
	navFn, ok := nav[target]
	if !ok {
		target, navFn = "home", func() { m.selectTab(0) }
	}

	go func() {
		time.Sleep(900 * time.Millisecond) // window mapped + first frame drawn
		fyne.DoAndWait(navFn)
		time.Sleep(500 * time.Millisecond) // let the target screen render fully

		var img image.Image
		fyne.DoAndWait(func() { img = w.Canvas().Capture() }) // first (clean) grab
		if f, err := os.Create(outDir + "/m-" + target + ".png"); err == nil {
			_ = png.Encode(f, img)
			_ = f.Close()
		}
		fyne.Do(func() { a.Quit() })
	}()

	w.ShowAndRun()
}
