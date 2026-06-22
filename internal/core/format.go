package core

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Itoa is a small re-export of strconv.Itoa used across the app.
func Itoa(n int) string { return strconv.Itoa(n) }

// excelEpoch is the base date Excel uses for its serial numbers (accounting for
// the historical 1900 leap-year bug, the practical epoch is 1899-12-30).
var excelEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

// SerialToTime converts an Excel serial date number into a time.Time.
func SerialToTime(serial int) time.Time {
	return excelEpoch.AddDate(0, 0, serial)
}

// TimeToSerial converts a time.Time into an Excel serial date number.
func TimeToSerial(t time.Time) int {
	return int(t.Sub(excelEpoch).Hours() / 24)
}

// FmtSerialDate formats an Excel serial date as YYYY-MM-DD.
func FmtSerialDate(serial int) string {
	return SerialToTime(serial).Format("2006-01-02")
}

// FmtSerialMonth formats an Excel serial date as YYYY-MM.
func FmtSerialMonth(serial int) string {
	return SerialToTime(serial).Format("2006-01")
}

// ParseDateSerial parses a YYYY-MM-DD string into an Excel serial; 0 on failure.
func ParseDateSerial(s string) int {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return TimeToSerial(t)
}

// ---- money ----
//
// Amounts are stored everywhere as integer MINOR UNITS (e.g. cents for USD,
// whole rupiah for Rp). The active currency determines how many decimals to
// show and which separators to use. This avoids floating-point money entirely.

type currencyFmt struct {
	symbol   string
	decimals int
	thousand string
	decimal  string
}

var currencies = map[string]currencyFmt{
	"Rp": {"Rp", 0, ".", ","},
	"$":  {"$", 2, ",", "."},
	"£":  {"£", 2, ",", "."},
	"€":  {"€", 2, ".", ","},
}

var active = currencies["Rp"]

// SetCurrencySymbol switches the active currency formatting.
func SetCurrencySymbol(sym string) {
	if c, ok := currencies[sym]; ok {
		active = c
	} else if sym != "" {
		active = currencyFmt{symbol: sym, decimals: 2, thousand: ",", decimal: "."}
	}
}

// CurrencyDecimals reports the active currency's number of decimal places.
func CurrencyDecimals() int { return active.decimals }

func pow10(n int) int {
	p := 1
	for i := 0; i < n; i++ {
		p *= 10
	}
	return p
}

func onlyDigits(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func atoiSafe(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// groupInt formats a non-negative integer with the given thousands separator.
func groupInt(n int, sep string) string {
	s := strconv.Itoa(n)
	var out strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		out.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if out.Len() > 0 {
			out.WriteString(sep)
		}
		out.WriteString(s[i : i+3])
	}
	return out.String()
}

// fmtMinor renders minor units with grouping and decimals (no symbol/sign).
func fmtMinor(minor int) string {
	base := pow10(active.decimals)
	s := groupInt(minor/base, active.thousand)
	if active.decimals > 0 {
		s += active.decimal + fmt.Sprintf("%0*d", active.decimals, minor%base)
	}
	return s
}

// FmtMoney formats minor units in the active currency, e.g. "$ 12,345.67" or
// "Rp 6.784.157". Negatives are wrapped in parentheses.
func FmtMoney(minor int) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	out := active.symbol + " " + fmtMinor(minor)
	if neg {
		return "(" + out + ")"
	}
	return out
}

// FmtMoneyPlain formats minor units with a leading minus instead of parentheses.
func FmtMoneyPlain(minor int) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	out := active.symbol + " " + fmtMinor(minor)
	if neg {
		return "-" + out
	}
	return out
}

// AmountToInput renders minor units as a plain editable string ("12.50"),
// without currency symbol or grouping — for pre-filling entry fields.
func AmountToInput(minor int) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	base := pow10(active.decimals)
	out := strconv.Itoa(minor / base)
	if active.decimals > 0 {
		out += "." + fmt.Sprintf("%0*d", active.decimals, minor%base)
	}
	if neg {
		out = "-" + out
	}
	return out
}

// ParseAmount parses a user-entered amount into minor units, honoring the active
// currency's decimals. Tolerates symbols, spaces and grouping; returns 0 on
// empty/invalid input.
func ParseAmount(s string) int {
	neg := strings.ContainsRune(s, '-')

	// Keep only digits and separators.
	var t strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' {
			t.WriteByte(byte(r))
		}
	}
	str := t.String()

	if active.decimals == 0 {
		n := atoiSafe(onlyDigits(str))
		if neg {
			n = -n
		}
		return n
	}

	wholeStr, fracStr := onlyDigits(str), ""
	if i := strings.LastIndexAny(str, ".,"); i >= 0 {
		after := onlyDigits(str[i+1:])
		// Treat the last separator as a decimal point only when 1..decimals
		// digits follow it; otherwise it's a thousands separator.
		if len(after) >= 1 && len(after) <= active.decimals {
			wholeStr, fracStr = onlyDigits(str[:i]), after
		}
	}
	for len(fracStr) < active.decimals {
		fracStr += "0"
	}
	if len(fracStr) > active.decimals {
		fracStr = fracStr[:active.decimals]
	}
	minor := atoiSafe(wholeStr)*pow10(active.decimals) + atoiSafe(fracStr)
	if neg {
		minor = -minor
	}
	return minor
}

// FmtPercent formats a 0..1 ratio as a percentage, e.g. 0.6698 -> "66.98%".
func FmtPercent(ratio float64) string {
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", ratio*100)
}

// FmtRatio formats a multiplier, e.g. 11.97 -> "11.97×".
func FmtRatio(r float64) string {
	return fmt.Sprintf("%.2f×", r)
}

// FmtUSD formats a USD amount with thousands separators, e.g. "$43,200".
func FmtUSD(amount float64) string {
	s := fmt.Sprintf("%.0f", amount)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var out strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		out.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if out.Len() > 0 {
			out.WriteString(",")
		}
		out.WriteString(s[i : i+3])
	}
	prefix := "$"
	if neg {
		prefix = "-$"
	}
	return prefix + out.String()
}
