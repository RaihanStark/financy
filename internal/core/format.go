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

// sepThousand / sepDecimal are the active grouping and decimal separators. They
// default to the active currency's, but a document's number-format setting can
// override them (see ApplyNumberFormat) without changing the symbol or decimals.
var sepThousand = active.thousand
var sepDecimal = active.decimal

// SetCurrencySymbol switches the active currency formatting and resets the
// separators to that currency's defaults (re-apply a number format afterwards).
func SetCurrencySymbol(sym string) {
	if c, ok := currencies[sym]; ok {
		active = c
	} else if sym != "" {
		active = currencyFmt{symbol: sym, decimals: 2, thousand: ",", decimal: "."}
	}
	sepThousand, sepDecimal = active.thousand, active.decimal
}

// numberFormats maps a style key to its (thousand, decimal) separators.
var numberFormats = map[string][2]string{
	"1,234.56": {",", "."},
	"1.234,56": {".", ","},
	"1 234,56": {" ", ","},
	"1 234.56": {" ", "."},
	"1234.56":  {"", "."},
	"1234,56":  {"", ","},
}

// NumberFormatStyles lists the selectable separator styles for the settings UI,
// in display order. The empty string means "use the currency default".
func NumberFormatStyles() []string {
	return []string{"1,234.56", "1.234,56", "1 234,56", "1 234.56", "1234.56", "1234,56"}
}

// ApplyNumberFormat overrides the grouping/decimal separators by style key. An
// unknown/empty style reverts to the active currency's defaults.
func ApplyNumberFormat(style string) {
	if f, ok := numberFormats[style]; ok {
		sepThousand, sepDecimal = f[0], f[1]
		return
	}
	sepThousand, sepDecimal = active.thousand, active.decimal
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
	s := groupInt(minor/base, sepThousand)
	if active.decimals > 0 {
		s += sepDecimal + fmt.Sprintf("%0*d", active.decimals, minor%base)
	}
	return s
}

// FmtMoneyInput formats minor units for an editable field: grouped digits with
// the active separators, no currency symbol, leading '-' for negatives.
func FmtMoneyInput(minor int) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	out := fmtMinor(minor)
	if neg {
		out = "-" + out
	}
	return out
}

// GroupTyped reformats a partially-typed amount with thousands grouping,
// preserving a trailing decimal separator and capping decimals at the currency's
// places. Used for live formatting in money entry fields as the user types.
func GroupTyped(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	neg := strings.HasPrefix(strings.TrimSpace(s), "-")

	// Keep only digits and the active decimal separator.
	var cleaned strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			cleaned.WriteRune(r)
		case sepDecimal != "" && string(r) == sepDecimal:
			cleaned.WriteRune(r)
		}
	}
	cs := cleaned.String()

	whole, frac, hasDec := cs, "", false
	if sepDecimal != "" {
		if i := strings.LastIndex(cs, sepDecimal); i >= 0 {
			whole, frac, hasDec = cs[:i], cs[i+len(sepDecimal):], true
		}
	}
	whole, frac = onlyDigits(whole), onlyDigits(frac)
	if active.decimals == 0 {
		hasDec, frac = false, ""
	}
	if len(frac) > active.decimals {
		frac = frac[:active.decimals]
	}

	grouped := groupDigitsStr(whole, sepThousand)
	if !hasDec {
		if neg && grouped != "" {
			return "-" + grouped
		}
		return grouped
	}
	if grouped == "" {
		grouped = "0"
	}
	out := grouped + sepDecimal + frac
	if neg {
		out = "-" + out
	}
	return out
}

// groupDigitsStr groups a digit string in threes (string-based, no overflow);
// leading zeros are trimmed. An empty separator disables grouping.
func groupDigitsStr(s, sep string) string {
	s = strings.TrimLeft(s, "0")
	if s == "" {
		return ""
	}
	if sep == "" {
		return s
	}
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

// FmtMoneyShort renders minor units compactly for chart axes, e.g. "$0",
// "$1.2k", "$26k", "Rp1.5M". Uses the active currency symbol with k/M/B
// abbreviations; integer math only (no float money).
func FmtMoneyShort(minor int) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	major := minor / pow10(active.decimals) // whole currency units
	out := active.symbol + abbrev(major)
	if neg {
		out = "-" + out
	}
	return out
}

// abbrev shortens a non-negative integer with k/M/B suffixes, keeping one
// decimal only for small magnitudes (e.g. 1234 -> "1.2k", 26206 -> "26k").
func abbrev(n int) string {
	switch {
	case n < 1_000:
		return strconv.Itoa(n)
	case n < 1_000_000:
		return abbrevUnit(n, 1_000, "k")
	case n < 1_000_000_000:
		return abbrevUnit(n, 1_000_000, "M")
	default:
		return abbrevUnit(n, 1_000_000_000, "B")
	}
}

func abbrevUnit(n, div int, suffix string) string {
	whole := n / div
	if whole >= 10 { // 10k+ : drop the decimal to stay short
		return strconv.Itoa(whole) + suffix
	}
	frac := (n % div) / (div / 10)
	if frac == 0 {
		return strconv.Itoa(whole) + suffix
	}
	return strconv.Itoa(whole) + "." + strconv.Itoa(frac) + suffix
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

// ParseAPRBps parses a user-entered annual percentage rate into basis points:
// "19.99" (or "19,99%") → 1999. Tolerates a % sign, spaces, and either decimal
// separator; extra decimals are truncated. Returns 0 on empty/invalid input.
func ParseAPRBps(s string) int {
	var t strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' {
			t.WriteByte(byte(r))
		}
	}
	str := t.String()
	whole, frac := str, ""
	if i := strings.LastIndexAny(str, ".,"); i >= 0 {
		whole, frac = onlyDigits(str[:i]), onlyDigits(str[i+1:])
	}
	for len(frac) < 2 {
		frac += "0"
	}
	return atoiSafe(onlyDigits(whole))*100 + atoiSafe(frac[:2])
}

// FmtAPRBps renders basis points as a percentage: 1999 → "19.99%", 1200 → "12%".
func FmtAPRBps(bps int) string {
	whole, frac := bps/100, bps%100
	switch {
	case frac == 0:
		return fmt.Sprintf("%d%%", whole)
	case frac%10 == 0:
		return fmt.Sprintf("%d.%d%%", whole, frac/10)
	default:
		return fmt.Sprintf("%d.%02d%%", whole, frac)
	}
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
