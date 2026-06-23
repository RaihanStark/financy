package view

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// TestDateEntryPicker covers the calendar-picker date field end to end: the field
// validates ISO text, exposes a calendar button, and selecting a day writes the
// chosen date back as YYYY-MM-DD.
func TestDateEntryPicker(t *testing.T) {
	test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	win = w // the popup attaches to this window's canvas

	de := newDateEntry(0)
	w.SetContent(de)
	w.Resize(fyne.NewSize(300, 80))

	// Keeps the app's ISO convention: junk is rejected, YYYY-MM-DD accepted.
	if de.Validate() == nil {
		t.Fatal("empty field should be invalid")
	}
	de.SetText("2026-03-15")
	if err := de.Validate(); err != nil {
		t.Fatalf("ISO date should validate, got %v", err)
	}
	de.SetText("15/03/2026")
	if de.Validate() == nil {
		t.Fatal("non-ISO date should be rejected")
	}
	de.SetText("")

	// The field carries a calendar button as its action item.
	btn, ok := de.ActionItem.(*widget.Button)
	if !ok {
		t.Fatalf("date field has no calendar button (ActionItem = %T)", de.ActionItem)
	}

	// Tapping it opens a calendar popup.
	test.Tap(btn)
	top := w.Canvas().Overlays().Top()
	pop, ok := top.(*widget.PopUp)
	if !ok {
		t.Fatalf("calendar button did not open a popup (top overlay = %T)", top)
	}
	cal, ok := pop.Content.(*widget.Calendar)
	if !ok {
		t.Fatalf("popup does not contain a calendar (content = %T)", pop.Content)
	}

	// Picking a day writes it back as ISO — and a local-zone, mid-day time must
	// still land on the same calendar day (no timezone drift).
	cal.OnChanged(time.Date(2026, 3, 15, 14, 30, 0, 0, time.FixedZone("UTC+7", 7*3600)))
	if de.Text != "2026-03-15" {
		t.Fatalf("after picking, field = %q, want %q", de.Text, "2026-03-15")
	}
}
