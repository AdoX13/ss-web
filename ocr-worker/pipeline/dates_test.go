package pipeline

import (
	"testing"
	"time"
)

func TestParseRomanianDate_NumericForms(t *testing.T) {
	cases := []struct {
		in    string
		year  int
		month time.Month
		day   int
		canon string
	}{
		{"Data: 01.06.2026", 2026, time.June, 1, "01.06.2026"},
		{"Data 1/6/2026", 2026, time.June, 1, "01.06.2026"},
		{"15-12-2025", 2025, time.December, 15, "15.12.2025"},
		{"prefix 31.01.2030 suffix", 2030, time.January, 31, "31.01.2030"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, canon, ok := ParseRomanianDate(c.in)
			if !ok {
				t.Fatalf("ParseRomanianDate(%q) = !ok", c.in)
			}
			if got.Year() != c.year || got.Month() != c.month || got.Day() != c.day {
				t.Errorf("date: want %d-%v-%d got %v", c.year, c.month, c.day, got)
			}
			if canon != c.canon {
				t.Errorf("canon: want %q got %q", c.canon, canon)
			}
		})
	}
}

func TestParseRomanianDate_WrittenMonths(t *testing.T) {
	cases := []struct {
		in    string
		month time.Month
	}{
		{"5 ianuarie 2026", time.January},
		{"5 Februarie 2026", time.February},
		{"15 martie 2026", time.March},
		{"15 mai 2026", time.May},
		{"30 septembrie 2026", time.September},
		{"31 decembrie 2026", time.December},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, _, ok := ParseRomanianDate(c.in)
			if !ok {
				t.Fatalf("ParseRomanianDate(%q) = !ok", c.in)
			}
			if got.Month() != c.month {
				t.Errorf("month: want %v got %v", c.month, got.Month())
			}
		})
	}
}

func TestParseRomanianDate_RejectsInvalid(t *testing.T) {
	bad := []string{
		"",
		"not a date",
		"99.99.9999",
		"31.02.2026",       // Feb 31 — buildDate's normalization rejects this
		"00.06.2026",       // day 0
		"15.13.2026",       // month 13
		"15.06.1700",       // year out of [1900, 2100]
		"15.06.3000",       // year out of [1900, 2100]
		"15 lunaFalsa 2026",
	}
	for _, b := range bad {
		if _, _, ok := ParseRomanianDate(b); ok {
			t.Errorf("expected %q to fail", b)
		}
	}
}

func TestStripDiacritics(t *testing.T) {
	in := "Întâi septembrie cu ț, ș și Ăsta"
	out := stripDiacritics(in)
	// Expected: ASCII-only Romanian letters, case preserved.
	want := "Intai septembrie cu t, s si Asta"
	if out != want {
		t.Errorf("stripDiacritics: want %q got %q", want, out)
	}
}
