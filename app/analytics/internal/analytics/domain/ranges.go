package domain

import (
	"errors"
	"fmt"
	"time"
)

// RangeKind says how a RangeDef turns into a window of days.
type RangeKind string

const (
	LastDays       RangeKind = "days"            // the N days ending at as-of
	MonthToDate    RangeKind = "month_to_date"   // the 1st of as-of's month to as-of
	LastMonth      RangeKind = "last_month"      // the whole month before as-of's month
	QuarterToDate  RangeKind = "quarter_to_date" // the 1st of as-of's quarter to as-of
	LastQuarter    RangeKind = "last_quarter"    // the whole quarter before as-of's quarter
	FixedDateRange RangeKind = "fixed"           // two given dates
)

const (
	dateLayout = "2006-01-02"
	// MaxRangeDays is the longest window a range may cover.
	MaxRangeDays = 366
	// MaxRangeDefs bounds the list, so a bad config cannot flood the UI.
	MaxRangeDefs = 50
	maxTextLen   = 60
)

// RangeDef is one entry of the configured list of date ranges. Relative kinds
// move with as-of; a fixed one does not. Days is used by LastDays, From and To
// (YYYY-MM-DD) by FixedDateRange.
type RangeDef struct {
	ID      string
	Label   string
	Group   string
	Kind    RangeKind
	Days    int
	From    string
	To      string
	Default bool
}

// Range is a RangeDef resolved to concrete days (both ends inclusive, UTC).
type Range struct {
	ID      string
	Label   string
	Group   string
	From    time.Time
	To      time.Time
	Default bool
}

// DefaultRangeDefs is the built-in list, used when no configured list is available.
func DefaultRangeDefs() []RangeDef {
	defs := []RangeDef{
		{ID: "last-7", Label: "Last 7 days", Group: "Recent", Kind: LastDays, Days: 7},
		{ID: "last-14", Label: "Last 14 days", Group: "Recent", Kind: LastDays, Days: 14},
		{ID: "last-30", Label: "Last 30 days", Group: "Recent", Kind: LastDays, Days: 30, Default: true},
		{ID: "last-60", Label: "Last 60 days", Group: "Recent", Kind: LastDays, Days: 60},
		{ID: "last-90", Label: "Last 90 days", Group: "Recent", Kind: LastDays, Days: 90},
		{ID: "last-180", Label: "Last 180 days", Group: "Recent", Kind: LastDays, Days: 180},
		{ID: "month-to-date", Label: "Month to date", Group: "Periods", Kind: MonthToDate},
		{ID: "last-month", Label: "Last month", Group: "Periods", Kind: LastMonth},
		{ID: "quarter-to-date", Label: "Quarter to date", Group: "Periods", Kind: QuarterToDate},
		{ID: "last-quarter", Label: "Last quarter", Group: "Periods", Kind: LastQuarter},
	}
	fixed := []struct {
		id, label, from, to string
	}{
		{"2022-02", "February 2022", "2022-02-01", "2022-02-28"},
		{"2022-03", "March 2022", "2022-03-01", "2022-03-31"},
		{"2022-04", "April 2022", "2022-04-01", "2022-04-30"},
		{"2022-05", "May 2022", "2022-05-01", "2022-05-31"},
		{"2022-06", "June 2022", "2022-06-01", "2022-06-30"},
		{"2022-q1", "Q1 2022", "2022-01-01", "2022-03-31"},
		{"2022-q2", "Q2 2022", "2022-04-01", "2022-06-30"},
		{"all-data", "All data", "2022-01-23", "2022-07-27"},
	}
	for _, m := range fixed {
		defs = append(defs, RangeDef{ID: m.id, Label: m.label, Group: "Fixed", Kind: FixedDateRange, From: m.from, To: m.to})
	}
	return defs
}

// ValidateRangeDefs reports why a list cannot be used: it must be a bounded,
// well-formed list with unique ids and exactly one default.
func ValidateRangeDefs(defs []RangeDef) error {
	if len(defs) == 0 {
		return errors.New("no ranges")
	}
	if len(defs) > MaxRangeDefs {
		return fmt.Errorf("%d ranges, at most %d", len(defs), MaxRangeDefs)
	}
	seen := map[string]bool{}
	defaults := 0
	for i, d := range defs {
		where := fmt.Sprintf("range %d (%q)", i+1, d.ID)
		switch {
		case d.ID == "" || len(d.ID) > maxTextLen:
			return fmt.Errorf("%s: id must be 1 to %d characters", where, maxTextLen)
		case seen[d.ID]:
			return fmt.Errorf("%s: duplicate id", where)
		case d.Label == "" || len(d.Label) > maxTextLen:
			return fmt.Errorf("%s: label must be 1 to %d characters", where, maxTextLen)
		case d.Group == "" || len(d.Group) > maxTextLen:
			return fmt.Errorf("%s: group must be 1 to %d characters", where, maxTextLen)
		}
		seen[d.ID] = true
		if d.Default {
			defaults++
		}
		if err := validateShape(d); err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
	}
	if defaults != 1 {
		return fmt.Errorf("%d default ranges, want exactly 1", defaults)
	}
	return nil
}

func validateShape(d RangeDef) error {
	switch d.Kind {
	case LastDays:
		if d.Days < 1 || d.Days > MaxRangeDays {
			return fmt.Errorf("days: want 1..%d", MaxRangeDays)
		}
	case MonthToDate, LastMonth, QuarterToDate, LastQuarter:
	case FixedDateRange:
		from, err := time.Parse(dateLayout, d.From)
		if err != nil {
			return errors.New("from: want YYYY-MM-DD")
		}
		to, err := time.Parse(dateLayout, d.To)
		if err != nil {
			return errors.New("to: want YYYY-MM-DD")
		}
		if from.After(to) {
			return errors.New("from must not be after to")
		}
		if days(from, to) > MaxRangeDays {
			return fmt.Errorf("must not exceed %d days", MaxRangeDays)
		}
	default:
		return fmt.Errorf("unknown kind %q", d.Kind)
	}
	return nil
}

// ResolveRanges turns a validated list into concrete windows, relative to asOf.
func ResolveRanges(defs []RangeDef, asOf time.Time) ([]Range, error) {
	if err := ValidateRangeDefs(defs); err != nil {
		return nil, err
	}
	out := make([]Range, 0, len(defs))
	for _, d := range defs {
		from, to := d.window(asOf)
		out = append(out, Range{ID: d.ID, Label: d.Label, Group: d.Group, From: from, To: to, Default: d.Default})
	}
	return out, nil
}

// window assumes the def is valid.
func (d RangeDef) window(asOf time.Time) (from, to time.Time) {
	y, m, _ := asOf.Date()
	monthStart := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	quarterStart := time.Date(y, m-(m-1)%3, 1, 0, 0, 0, 0, time.UTC)
	switch d.Kind {
	case LastDays:
		return asOf.AddDate(0, 0, -(d.Days - 1)), asOf
	case MonthToDate:
		return monthStart, asOf
	case LastMonth:
		return monthStart.AddDate(0, -1, 0), monthStart.AddDate(0, 0, -1)
	case QuarterToDate:
		return quarterStart, asOf
	case LastQuarter:
		return quarterStart.AddDate(0, -3, 0), quarterStart.AddDate(0, 0, -1)
	default: // FixedDateRange
		from, _ = time.Parse(dateLayout, d.From)
		to, _ = time.Parse(dateLayout, d.To)
		return from, to
	}
}

func days(from, to time.Time) int { return int(to.Sub(from).Hours()/24) + 1 }

// FindRange returns the range with exactly these days.
func FindRange(ranges []Range, from, to time.Time) (Range, bool) {
	for _, r := range ranges {
		if r.From.Equal(from) && r.To.Equal(to) {
			return r, true
		}
	}
	return Range{}, false
}

// DefaultRange returns the range marked as the default.
func DefaultRange(ranges []Range) (Range, bool) {
	for _, r := range ranges {
		if r.Default {
			return r, true
		}
	}
	return Range{}, false
}
