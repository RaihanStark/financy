package view

import "testing"

// Typing filters the suggestion pool and opens the dropdown; an exact match or
// empty text closes it; choosing a suggestion fills the entry and dismisses it.
func TestCompletionEntryLiveFilter(t *testing.T) {
	_, w := newViewTest(t)

	c := newCompletionEntry([]string{"Netflix", "Spotify", "Net Provider", "Acme"})
	mount(w, c)

	// Typing a substring shows only the matching suggestions.
	c.SetText("net")
	if got := c.matches; len(got) != 2 {
		t.Fatalf("after %q: matches = %v, want 2 (Netflix, Net Provider)", "net", got)
	}
	if c.popup == nil || !c.popup.Visible() {
		t.Fatalf("dropdown should be visible while typing a partial match")
	}

	// Narrowing the query narrows the matches.
	c.SetText("netf")
	if got := c.matches; len(got) != 1 || got[0] != "Netflix" {
		t.Fatalf("after %q: matches = %v, want [Netflix]", "netf", got)
	}

	// A no-match query hides the dropdown but keeps the typed text (new payee).
	c.SetText("brand-new payee")
	if len(c.matches) != 0 {
		t.Fatalf("unexpected matches for novel text: %v", c.matches)
	}
	if c.popup != nil && c.popup.Visible() {
		t.Fatalf("dropdown should be hidden when nothing matches")
	}

	// Selecting a suggestion fills the entry and closes the dropdown.
	c.SetText("net")
	want := c.matches[0]
	c.list.OnSelected(0)
	if c.Text != want {
		t.Fatalf("after selecting suggestion: Text = %q, want %q", c.Text, want)
	}
	if c.popup.Visible() {
		t.Fatalf("dropdown should close after selecting a suggestion")
	}
}
