package ui

import (
	"context"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// updateCheckInterval throttles the automatic (startup) update check so we hit
// GitHub at most once per day.
const updateCheckInterval = 24 * time.Hour

// checkForUpdatesAsync queries GitHub for a newer release off the UI thread and,
// when one is found, shows a modal popup.
//
// manual=true means the user explicitly asked (File ▸ Check for Updates…): the
// throttle and skip-version are ignored, errors surface as a dialog, and being
// up to date is confirmed. manual=false is the silent startup check.
func checkForUpdatesAsync(manual bool) {
	if !manual {
		if prefs == nil || prefs.DisableUpdateCheck || !prefs.dueForUpdateCheck(time.Now()) {
			return
		}
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		rel, available, err := core.UpdateAvailable(ctx)

		fyne.Do(func() {
			// Record the automatic-check time regardless of outcome so an offline
			// machine doesn't retry on every launch.
			if !manual && prefs != nil {
				prefs.LastUpdateCheck = time.Now().UTC().Format(time.RFC3339)
				prefs.save()
			}
			if ctl == nil || ctl.win == nil {
				return
			}
			if err != nil {
				if manual {
					dialog.ShowError(err, ctl.win)
				}
				return
			}
			if !available {
				if manual {
					dialog.ShowInformation("Up to date",
						"You're running the latest version of Financy (v"+core.Version+").", ctl.win)
				}
				return
			}
			// Honour a skipped version on the automatic path only.
			if !manual && prefs != nil && prefs.SkipVersion == rel.Version {
				return
			}
			showUpdateDialog(rel)
		})
	}()
}

// showUpdateDialog presents the dedicated "Update available" popup.
func showUpdateDialog(rel core.Release) {
	if ctl == nil || ctl.win == nil {
		return
	}

	body := container.NewVBox(
		txt("Update available", colText, 18, true),
		txt("Financy v"+rel.Version+" is available — you have v"+core.Version+".", colTextDim, 12, false),
	)

	if notes := updateNotesExcerpt(rel.Notes); notes != "" {
		lbl := widget.NewLabel(notes)
		lbl.Wrapping = fyne.TextWrapWord
		scroll := container.NewVScroll(lbl)
		scroll.SetMinSize(fyne.NewSize(420, 150))
		body.Add(spacerH(6))
		body.Add(scroll)
	}

	var d *dialog.CustomDialog
	download := primaryButton("Download", nil, func() {
		if u, err := url.Parse(rel.URL); err == nil && rel.URL != "" {
			_ = fyne.CurrentApp().OpenURL(u)
		}
		d.Hide()
	})
	skip := secondaryButton("Skip This Version", nil, func() {
		if prefs != nil {
			prefs.SkipVersion = rel.Version
			prefs.save()
		}
		d.Hide()
	})
	later := secondaryButton("Later", nil, func() { d.Hide() })

	content := container.NewVBox(
		body,
		spacerH(14),
		container.NewCenter(container.NewHBox(later, skip, download)),
	)
	d = dialog.NewCustomWithoutButtons("Software Update", content, ctl.win)
	d.Resize(fyne.NewSize(500, 380))
	d.Show()
}

// updateNotesExcerpt trims the release body to a short, dialog-friendly excerpt.
func updateNotesExcerpt(notes string) string {
	notes = strings.TrimSpace(notes)
	const maxLen = 700
	if len(notes) > maxLen {
		notes = strings.TrimSpace(notes[:maxLen]) + "…"
	}
	return notes
}
