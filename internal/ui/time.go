// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package ui

import (
	"fmt"
	"time"
)

// Section is a human time bucket. Sessions are grouped by these because people
// navigate their own work by when they did it, not by identifier.
type Section int

const (
	SectionToday Section = iota
	SectionYesterday
	SectionThisWeek
	SectionThisMonth
	SectionOlder
)

// Title returns the uppercase section heading.
func (s Section) Title() string {
	switch s {
	case SectionToday:
		return "TODAY"
	case SectionYesterday:
		return "YESTERDAY"
	case SectionThisWeek:
		return "THIS WEEK"
	case SectionThisMonth:
		return "THIS MONTH"
	default:
		return "OLDER"
	}
}

// SectionFor buckets when relative to now, using local calendar days so
// "yesterday" means the previous calendar day rather than 24 hours ago.
func SectionFor(when, now time.Time) Section {
	if when.IsZero() {
		return SectionOlder
	}
	whenDay := startOfDay(when.Local())
	nowDay := startOfDay(now.Local())
	days := int(nowDay.Sub(whenDay).Hours() / 24)
	switch {
	case days <= 0:
		return SectionToday
	case days == 1:
		return SectionYesterday
	case days < 7:
		return SectionThisWeek
	case days < 31:
		return SectionThisMonth
	default:
		return SectionOlder
	}
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// Relative renders a compact age such as "4m ago" or "2d ago". It never shows
// more than one unit, because the column is three or four cells wide and a
// precise duration is not what the reader is asking.
func Relative(when, now time.Time) string {
	if when.IsZero() {
		return "unknown"
	}
	delta := now.Sub(when)
	if delta < 0 {
		// Clock skew between the vendor's timestamp and this host. Reporting a
		// negative age would look like a bug; "just now" is the honest reading.
		return "just now"
	}
	switch {
	case delta < time.Minute:
		return "just now"
	case delta < time.Hour:
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	case delta < 31*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(delta.Hours()/24))
	case delta < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(delta.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(delta.Hours()/(24*365)))
	}
}
