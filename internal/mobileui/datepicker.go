package mobileui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/raihanstark/financy/internal/core"
)

// dateField builds a tappable, calendar-backed date input styled like the other
// form controls. It shows the current date and opens a month calendar on tap
// (no keyboard, no typing YYYY-MM-DD by hand). The returned getter reports the
// chosen date as an Excel serial, so callers read it the same way they read a
// parsed text entry.
func (m *mobileApp) dateField(label string, initial int) (fyne.CanvasObject, func() int) {
	if initial == 0 {
		initial = core.TodaySerial
	}
	sel := initial

	val := newText(core.FmtSerialDate(sel), colInk, 15, false)
	chevron := widget.NewIcon(theme.NewThemedResource(theme.MenuDropDownIcon()))

	box := canvas.NewRectangle(colCard)
	box.CornerRadius = 10
	box.StrokeColor = colLine
	box.StrokeWidth = 1

	inner := container.NewBorder(nil, nil, nil, chevron, container.NewVBox(val))
	card := newTappableCard(container.NewStack(box, insets(inner, 12, 12, 12, 10)), func() {
		m.showDatePicker(sel, func(n int) {
			sel = n
			val.Text = core.FmtSerialDate(n)
			val.Refresh()
		})
	})
	return field(label, card), func() int { return sel }
}

// showDatePicker pops a modal month calendar seeded at `current`; picking a day
// calls onPick with its serial and dismisses the sheet.
func (m *mobileApp) showDatePicker(current int, onPick func(int)) {
	if current == 0 {
		current = core.TodaySerial
	}
	var d dialog.Dialog
	cal := widget.NewCalendar(core.SerialToTime(current), func(t time.Time) {
		onPick(core.TimeToSerial(t))
		if d != nil {
			d.Hide()
		}
	})
	d = dialog.NewCustom("Select date", "Cancel", cal, m.win)
	d.Resize(fyne.NewSize(340, 400))
	d.Show()
}
