package view

import (
	"errors"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// newDateEntry returns a YYYY-MM-DD text field with a calendar-picker button
// docked at its right edge. It keeps the app's ISO date convention (so it reads
// the same as dates everywhere else in the UI) while letting the user pick a day
// from a popup calendar instead of typing. Pass initialSerial to seed the field
// and the calendar's starting month, or 0 for an empty field (callers often
// SetText afterwards).
func newDateEntry(initialSerial int) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder("YYYY-MM-DD")
	e.Validator = func(s string) error {
		if parseDateSerial(s) == 0 {
			return errors.New("use YYYY-MM-DD")
		}
		return nil
	}
	if initialSerial != 0 {
		e.SetText(fmtSerialDate(initialSerial))
	}

	var pop *widget.PopUp
	btn := widget.NewButtonWithIcon("", theme.CalendarIcon(), func() {
		if win == nil {
			return
		}
		// Seed the calendar from the field's current value, falling back to today.
		seed := parseDateSerial(e.Text)
		if seed == 0 {
			seed = todaySerial
		}
		cal := widget.NewCalendar(serialToTime(seed), func(t time.Time) {
			// Normalise the picked day to the app's UTC-midnight convention
			// (matching TodaySerial and ParseDateSerial) so the chosen date can't
			// drift by a day from the calendar's local timezone / time-of-day.
			day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			e.SetText(fmtSerialDate(timeToSerial(day)))
			if pop != nil {
				pop.Hide()
			}
		})
		pop = widget.NewPopUp(cal, win.Canvas())
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(e)
		pop.ShowAtPosition(pos.Add(fyne.NewPos(0, e.Size().Height)))
		pop.Resize(cal.MinSize())
	})
	btn.Importance = widget.LowImportance
	e.ActionItem = btn
	return e
}
