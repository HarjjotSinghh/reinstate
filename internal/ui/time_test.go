// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package ui

import (
	"testing"
	"time"
)

// Every time assertion runs against a fixed reference instant. Nothing here may
// call time.Now(): a bucket boundary that only misbehaves near midnight is
// exactly the kind of defect a clock-reading test hides.
//
// The reference sits in mid-June, which no time zone uses for a daylight-saving
// transition, and it is built in time.Local because SectionFor buckets by local
// calendar day.
func referenceNow() time.Time {
	return time.Date(2026, time.June, 15, 12, 0, 0, 0, time.Local)
}

func localTime(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.Local)
}

func TestSectionFor(t *testing.T) {
	now := referenceNow()

	cases := []struct {
		name string
		when time.Time
		// now overrides the reference for the rows that need a specific
		// wall-clock position within the day.
		nowOverride time.Time
		want        Section
	}{
		{
			name: "the same instant is today",
			when: now,
			want: SectionToday,
		},
		{
			name: "earlier the same day is today",
			when: localTime(2026, time.June, 15, 0, 1),
			want: SectionToday,
		},
		{
			name: "later the same day is today",
			when: localTime(2026, time.June, 15, 23, 59),
			want: SectionToday,
		},
		{
			name: "twenty hours that stay inside one day is today",
			when: localTime(2026, time.June, 15, 3, 0),
			// A twenty-hour gap is nearly a day, but the calendar day never
			// changed, so bucketing by elapsed hours would be wrong here.
			nowOverride: localTime(2026, time.June, 15, 23, 0),
			want:        SectionToday,
		},
		{
			name: "a future timestamp from clock skew is today",
			when: localTime(2026, time.June, 16, 12, 0),
			want: SectionToday,
		},
		{
			name: "the previous calendar day is yesterday",
			when: localTime(2026, time.June, 14, 12, 0),
			want: SectionYesterday,
		},
		{
			name: "two hours across midnight is yesterday",
			when: localTime(2026, time.June, 14, 23, 30),
			// Only two hours elapsed, but the calendar day turned, which is how
			// people describe their own work.
			nowOverride: localTime(2026, time.June, 15, 1, 30),
			want:        SectionYesterday,
		},
		{
			name: "two days is this week",
			when: localTime(2026, time.June, 13, 12, 0),
			want: SectionThisWeek,
		},
		{
			name: "three days is this week",
			when: localTime(2026, time.June, 12, 9, 0),
			want: SectionThisWeek,
		},
		{
			name: "six days is the last day of this week",
			when: localTime(2026, time.June, 9, 12, 0),
			want: SectionThisWeek,
		},
		{
			name: "seven days is this month",
			when: localTime(2026, time.June, 8, 12, 0),
			want: SectionThisMonth,
		},
		{
			name: "ten days is this month",
			when: localTime(2026, time.June, 5, 12, 0),
			want: SectionThisMonth,
		},
		{
			name: "thirty days is the last day of this month",
			when: localTime(2026, time.May, 16, 12, 0),
			want: SectionThisMonth,
		},
		{
			name: "thirty-one days is older",
			when: localTime(2026, time.May, 15, 12, 0),
			want: SectionOlder,
		},
		{
			name: "sixty days is older",
			when: localTime(2026, time.April, 16, 12, 0),
			want: SectionOlder,
		},
		{
			name: "years back is older",
			when: localTime(2023, time.January, 2, 12, 0),
			want: SectionOlder,
		},
		{
			name: "the zero time is older",
			when: time.Time{},
			want: SectionOlder,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reference := now
			if !tc.nowOverride.IsZero() {
				reference = tc.nowOverride
			}
			if got := SectionFor(tc.when, reference); got != tc.want {
				t.Fatalf("SectionFor(%s, %s) = %s, want %s",
					tc.when.Format(time.RFC3339), reference.Format(time.RFC3339), got.Title(), tc.want.Title())
			}
		})
	}
}

func TestSectionTitle(t *testing.T) {
	cases := []struct {
		section Section
		want    string
	}{
		{section: SectionToday, want: "TODAY"},
		{section: SectionYesterday, want: "YESTERDAY"},
		{section: SectionThisWeek, want: "THIS WEEK"},
		{section: SectionThisMonth, want: "THIS MONTH"},
		{section: SectionOlder, want: "OLDER"},
		{section: Section(99), want: "OLDER"},
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.section.Title(); got != tc.want {
				t.Fatalf("Section(%d).Title() = %q, want %q", int(tc.section), got, tc.want)
			}
		})
		seen[tc.want] = true
	}
	if len(seen) != 5 {
		t.Fatalf("section headings collapsed to %d distinct titles, want 5", len(seen))
	}
}

func TestRelative(t *testing.T) {
	// Relative is pure duration arithmetic, so UTC keeps the arithmetic visible.
	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		when time.Time
		want string
	}{
		{name: "zero time is unknown", when: time.Time{}, want: "unknown"},
		{name: "the same instant", when: now, want: "just now"},
		{name: "one second", when: now.Add(-time.Second), want: "just now"},
		{name: "thirty seconds", when: now.Add(-30 * time.Second), want: "just now"},
		{name: "fifty-nine seconds", when: now.Add(-59 * time.Second), want: "just now"},
		{name: "one minute", when: now.Add(-time.Minute), want: "1m ago"},
		{name: "five and a half minutes truncates", when: now.Add(-5*time.Minute - 30*time.Second), want: "5m ago"},
		{name: "fifty-nine minutes", when: now.Add(-59 * time.Minute), want: "59m ago"},
		{name: "one hour", when: now.Add(-time.Hour), want: "1h ago"},
		{name: "just under a day", when: now.Add(-23*time.Hour - 59*time.Minute), want: "23h ago"},
		{name: "one day", when: now.Add(-24 * time.Hour), want: "1d ago"},
		{name: "six days", when: now.Add(-6 * 24 * time.Hour), want: "6d ago"},
		{name: "thirty days", when: now.Add(-30 * 24 * time.Hour), want: "30d ago"},
		{name: "thirty-one days rolls over to months", when: now.Add(-31 * 24 * time.Hour), want: "1mo ago"},
		{name: "ninety days", when: now.Add(-90 * 24 * time.Hour), want: "3mo ago"},
		{name: "just under a year", when: now.Add(-364 * 24 * time.Hour), want: "12mo ago"},
		{name: "one year", when: now.Add(-365 * 24 * time.Hour), want: "1y ago"},
		{name: "two years", when: now.Add(-800 * 24 * time.Hour), want: "2y ago"},
		{
			// Vendor timestamps come from other hosts, so a future reading is a
			// skewed clock rather than a bug worth showing the reader.
			name: "a future timestamp reads as just now",
			when: now.Add(time.Hour),
			want: "just now",
		},
		{
			name: "a far future timestamp still reads as just now",
			when: now.Add(72 * time.Hour),
			want: "just now",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Relative(tc.when, now); got != tc.want {
				t.Fatalf("Relative(%s, %s) = %q, want %q",
					tc.when.Format(time.RFC3339), now.Format(time.RFC3339), got, tc.want)
			}
		})
	}
}

// TestRelativeIsAlwaysOneUnit guards the column budget: the age column is a
// handful of cells wide, so the rendering may never grow a second unit.
func TestRelativeIsAlwaysOneUnit(t *testing.T) {
	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	for _, minutes := range []int{0, 1, 59, 60, 61, 1439, 1440, 10_000, 100_000, 1_000_000, 5_000_000} {
		when := now.Add(-time.Duration(minutes) * time.Minute)
		got := Relative(when, now)
		if Width(got) > 9 {
			t.Fatalf("Relative(-%dm) = %q, %d cells wide: it must fit the age column", minutes, got, Width(got))
		}
	}
}
