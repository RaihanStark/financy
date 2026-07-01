package mobileui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// openSettings is the full-screen settings page: open/export a .financy file,
// load demo data, or start fresh, plus the live-sync status.
func (m *mobileApp) openSettings() {
	m.pushPage(func(close func()) fyne.CanvasObject {
		openBtn := widget.NewButton("Open .financy file…", func() { close(); m.importFile() })
		openBtn.Importance = widget.HighImportance
		exportBtn := widget.NewButton("Export a copy…", func() { close(); m.exportFile() })
		demoBtn := widget.NewButton("Load demo data", func() {
			close()
			m.confirmReplace("Replace everything with demo data?", true)
		})
		freshBtn := widget.NewButton("Erase & start fresh", func() {
			close()
			m.confirmReplace("Erase all data and start with an empty document?", false)
		})

		box := container.NewVBox(
			newText("Document", colInk, 14, true),
			wrapText("This device keeps one document, saved automatically. Open a "+
				".financy file to bring it here, or export a copy to share it."),
			gap(10),
			openBtn,
			gap(6),
			exportBtn,
			gap(6),
			demoBtn,
			gap(6),
			freshBtn,
		)

		if m.sourceURI != nil {
			unlink := widget.NewButton("Stop syncing", func() {
				m.sourceURI = nil
				close()
			})
			box.Add(gap(14))
			box.Add(newText("Syncing to", colInk, 14, true))
			box.Add(newText(m.sourceURI.Name(), colPos, 12, true))
			box.Add(wrapText("Edits are written back to this file while the app is open. " +
				"Re-open it after restarting to resume syncing."))
			box.Add(gap(6))
			box.Add(unlink)
		}

		box.Add(gap(16))
		box.Add(newText("About", colInk, 14, true))
		box.Add(newText("Financy v"+core.Version, colInkDim, 12, false))

		return m.formPage("Settings", "", box, nil, nil, close)
	})
}

// confirmReplace is a full-screen confirmation before discarding the current
// document (load demo / start fresh).
func (m *mobileApp) confirmReplace(msg string, demo bool) {
	m.pushPage(func(close func()) fyne.CanvasObject {
		replace := widget.NewButton("Replace", func() { close(); m.replaceDocument(demo) })
		replace.Importance = widget.HighImportance
		cancel := widget.NewButton("Cancel", close)

		body := container.NewVBox(
			wrapText(msg),
			gap(18),
			replace,
			gap(6),
			cancel,
		)
		return m.formPage("Are you sure?", "", body, nil, nil, close)
	})
}
