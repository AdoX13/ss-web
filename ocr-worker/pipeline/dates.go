package pipeline

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Romanian date parser.
//
// Supported forms:
//   DD.MM.YYYY    01.06.2026
//   DD/MM/YYYY    01/06/2026
//   DD-MM-YYYY    01-06-2026
//   D Month YYYY  1 iunie 2026          (Romanian month names, with diacritics or without)
//
// Returns the parsed time at 00:00:00 UTC and a normalized canonical
// string (DD.MM.YYYY) for storage. If parsing fails, returns (zero, "").

var (
	// Compact numeric form.
	dateNumericRe = regexp.MustCompile(`(\d{1,2})[./\-](\d{1,2})[./\-](\d{4})`)

	// "1 iunie 2026" — written month form. Match is case-insensitive
	// and tolerates diacritic stripping.
	dateWrittenRe = regexp.MustCompile(`(?i)(\d{1,2})\s+([a-zăâîșțĂÂÎȘȚ]+)\s+(\d{4})`)
)

// romanianMonths maps lowercased, diacritic-stripped Romanian month
// names to their 1..12 month numbers.
var romanianMonths = map[string]time.Month{
	"ianuarie":   time.January,
	"februarie":  time.February,
	"martie":     time.March,
	"aprilie":    time.April,
	"mai":        time.May,
	"iunie":      time.June,
	"iulie":      time.July,
	"august":     time.August,
	"septembrie": time.September,
	"octombrie":  time.October,
	"noiembrie":  time.November,
	"decembrie":  time.December,
}

// ParseRomanianDate tries every supported form against the input.
// The input may contain surrounding noise; the function searches for
// the first match.
func ParseRomanianDate(s string) (time.Time, string, bool) {
	if t, canon, ok := parseNumericDate(s); ok {
		return t, canon, true
	}
	if t, canon, ok := parseWrittenDate(s); ok {
		return t, canon, true
	}
	return time.Time{}, "", false
}

func parseNumericDate(s string) (time.Time, string, bool) {
	m := dateNumericRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, "", false
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	year, _ := strconv.Atoi(m[3])
	return buildDate(day, month, year)
}

func parseWrittenDate(s string) (time.Time, string, bool) {
	m := dateWrittenRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, "", false
	}
	day, _ := strconv.Atoi(m[1])
	monthName := stripDiacritics(strings.ToLower(m[2]))
	month, ok := romanianMonths[monthName]
	if !ok {
		return time.Time{}, "", false
	}
	year, _ := strconv.Atoi(m[3])
	return buildDate(day, int(month), year)
}

// buildDate validates ranges and produces the canonical (DD.MM.YYYY) form.
// We use time.Date's overflow normalization to validate: a date like
// 31 February becomes 3 March, which we then reject by comparing.
func buildDate(day, month, year int) (time.Time, string, bool) {
	if year < 1900 || year > 2100 {
		return time.Time{}, "", false
	}
	if month < 1 || month > 12 {
		return time.Time{}, "", false
	}
	if day < 1 || day > 31 {
		return time.Time{}, "", false
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, "", false
	}
	canon := normalizeDate(day, month, year)
	return t, canon, true
}

func normalizeDate(day, month, year int) string {
	var b strings.Builder
	if day < 10 {
		b.WriteByte('0')
	}
	b.WriteString(strconv.Itoa(day))
	b.WriteByte('.')
	if month < 10 {
		b.WriteByte('0')
	}
	b.WriteString(strconv.Itoa(month))
	b.WriteByte('.')
	b.WriteString(strconv.Itoa(year))
	return b.String()
}

// stripDiacritics maps the Romanian-specific letters to their ASCII
// equivalents so OCR variations like "iunie" vs "iunîe" still match.
// Limited to the diacritics we actually expect; we don't pull in
// golang.org/x/text/unicode/norm for this.
var diacriticMap = map[rune]rune{
	'ă': 'a', 'â': 'a', 'î': 'i', 'ș': 's', 'ț': 't',
	'Ă': 'A', 'Â': 'A', 'Î': 'I', 'Ș': 'S', 'Ț': 'T',
}

func stripDiacritics(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if mapped, ok := diacriticMap[r]; ok {
			b.WriteRune(mapped)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
