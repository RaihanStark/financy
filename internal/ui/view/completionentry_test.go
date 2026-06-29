package view

import (
	"testing"

	"fyne.io/fyne/v2"
)

var (
	keyDown   = fyne.KeyEvent{Name: fyne.KeyDown}
	keyReturn = fyne.KeyEvent{Name: fyne.KeyReturn}
)

// Typing filters the suggestion pool and opens the inline dropdown; an exact
// match or empty text closes it; choosing a suggestion fills the entry.
func TestCompletionEntryLiveFilter(t *testing.T) {
	_, w := newViewTest(t)

	c := newCompletionEntry([]string{"Netflix", "Spotify", "Net Provider", "Acme"})
	mount(w, c)

	// Typing a substring shows only the matching suggestions.
	c.input.SetText("net")
	if got := c.matches; len(got) != 2 {
		t.Fatalf("after %q: matches = %v, want 2 (Netflix, Net Provider)", "net", got)
	}
	if !c.open {
		t.Fatalf("dropdown should be visible while typing a partial match")
	}

	// Narrowing the query narrows the matches.
	c.input.SetText("netf")
	if got := c.matches; len(got) != 1 || got[0] != "Netflix" {
		t.Fatalf("after %q: matches = %v, want [Netflix]", "netf", got)
	}

	// A no-match query hides the dropdown but keeps the typed text (new payee).
	c.input.SetText("brand-new payee")
	if len(c.matches) != 0 {
		t.Fatalf("unexpected matches for novel text: %v", c.matches)
	}
	if c.open {
		t.Fatalf("dropdown should be hidden when nothing matches")
	}
	if c.Text() != "brand-new payee" {
		t.Fatalf("free text not preserved: Text() = %q", c.Text())
	}

	// Selecting a suggestion fills the entry and closes the dropdown.
	c.input.SetText("net")
	want := c.matches[0]
	c.commit(want)
	if c.Text() != want {
		t.Fatalf("after selecting suggestion: Text() = %q, want %q", c.Text(), want)
	}
	if c.open {
		t.Fatalf("dropdown should close after selecting a suggestion")
	}
}

// Keyboard navigation moves the highlight and Enter commits the highlighted row.
func TestCompletionEntryKeyboardSelect(t *testing.T) {
	_, w := newViewTest(t)

	c := newCompletionEntry([]string{"Netflix", "Net Provider"})
	mount(w, c)

	c.input.SetText("net") // matches: [Netflix, Net Provider]
	if !c.open {
		t.Fatalf("dropdown should be open")
	}
	c.input.TypedKey(&keyDown)   // highlight first
	c.input.TypedKey(&keyDown)   // highlight second
	c.input.TypedKey(&keyReturn) // commit highlighted

	if c.Text() != "Net Provider" {
		t.Fatalf("Enter should commit highlighted row; Text() = %q", c.Text())
	}
	if c.open {
		t.Fatalf("dropdown should close after Enter selection")
	}
}

// SetText (programmatic prefill) must not pop the dropdown open.
func TestCompletionEntrySetTextDoesNotOpen(t *testing.T) {
	_, w := newViewTest(t)

	c := newCompletionEntry([]string{"Netflix"})
	mount(w, c)

	c.SetText("Net")
	if c.open {
		t.Fatalf("programmatic SetText must not open the dropdown")
	}
}
