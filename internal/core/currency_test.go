package core

import "testing"

func TestCurrencyFormatting(t *testing.T) {
	// Rp has 0 decimals: minor units == whole rupiah.
	SetCurrencySymbol("Rp")
	if got := FmtMoney(6784157); got != "Rp 6.784.157" {
		t.Fatalf("Rp format = %q", got)
	}
	if got := FmtMoney(-74407015); got != "(Rp 74.407.015)" {
		t.Fatalf("Rp negative = %q", got)
	}

	// USD has 2 decimals: 1234567 minor == $12,345.67.
	SetCurrencySymbol("$")
	if got := FmtMoney(1234567); got != "$ 12,345.67" {
		t.Fatalf("USD format = %q", got)
	}
	if got := AmountToInput(1234567); got != "12345.67" {
		t.Fatalf("USD input form = %q", got)
	}
	// Parsing is the inverse and tolerates grouping.
	if got := ParseAmount("$12,345.67"); got != 1234567 {
		t.Fatalf("ParseAmount = %d, want 1234567", got)
	}
	if got := ParseAmount("12.5"); got != 1250 {
		t.Fatalf("ParseAmount(12.5) = %d, want 1250", got)
	}
	if got := ParseAmount("1,000"); got != 100000 {
		t.Fatalf("ParseAmount(1,000) = %d, want 100000", got)
	}

	// EUR uses "." thousands and "," decimal.
	s := NewStore()
	s.SetSettings("", "€", 2026)
	if got := FmtMoney(100000); got != "€ 1.000,00" {
		t.Fatalf("EUR via SetSettings = %q", got)
	}

	SetCurrencySymbol("Rp") // restore for other tests
}

func TestFmtMoneyShort(t *testing.T) {
	SetCurrencySymbol("$") // 2 decimals: minor units are cents
	cases := map[int]string{
		0:         "$0",
		50000:     "$500",
		120000:    "$1.2k",
		2620618:   "$26k",
		5400000:   "$54k",
		100000000: "$1M",
		-2620618:  "-$26k",
	}
	for minor, want := range cases {
		if got := FmtMoneyShort(minor); got != want {
			t.Errorf("FmtMoneyShort(%d) = %q, want %q", minor, got, want)
		}
	}
	SetCurrencySymbol("Rp")
}
