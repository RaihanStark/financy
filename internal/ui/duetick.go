package ui

import (
	"time"

	"fyne.io/fyne/v2"

	"github.com/raihanstark/financy/internal/core"
	"github.com/raihanstark/financy/internal/ui/view"
)

// startDueTicker re-evaluates the date and due recurring entries once a minute,
// so an app left running notices midnight rollover and newly due items without
// the document being reopened. It only ever *shows* the existing review prompt;
// nothing posts without confirmation.
func startDueTicker() {
	go func() {
		for range time.NewTicker(time.Minute).C { // app-lifetime; never stopped
			fyne.Do(func() {
				if store == nil || ctl == nil || ctl.win == nil {
					return
				}
				if core.RefreshToday() {
					view.SyncToday()
					ctl.refresh() // due tints and time buckets are date-derived
				}
				view.RecheckRecurringDue()
			})
		}
	}()
}
