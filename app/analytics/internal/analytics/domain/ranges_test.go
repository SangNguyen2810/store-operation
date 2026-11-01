package domain

import (
	"strings"
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestRelativeRangesFollowAsOf(t *testing.T) {
	cases := []struct {
		name     string
		asOf     string
		kind     RangeKind
		days     int
		from, to string
	}{
		{"7 days", "2022-07-27", LastDays, 7, "2022-07-21", "2022-07-27"},
		{"30 days crosses a month", "2022-07-27", LastDays, 30, "2022-06-28", "2022-07-27"},
		{"1 day is as-of itself", "2022-07-27", LastDays, 1, "2022-07-27", "2022-07-27"},
		{"month to date", "2022-07-27", MonthToDate, 0, "2022-07-01", "2022-07-27"},
		{"month to date on the 1st", "2022-07-01", MonthToDate, 0, "2022-07-01", "2022-07-01"},
		{"last month", "2022-07-27", LastMonth, 0, "2022-06-01", "2022-06-30"},
		{"last month across a year", "2022-01-10", LastMonth, 0, "2021-12-01", "2021-12-31"},
		{"last month in a leap year", "2024-03-10", LastMonth, 0, "2024-02-01", "2024-02-29"},
		{"quarter to date, third quarter", "2022-07-27", QuarterToDate, 0, "2022-07-01", "2022-07-27"},
		{"quarter to date, first quarter", "2022-03-31", QuarterToDate, 0, "2022-01-01", "2022-03-31"},
		{"quarter to date, second quarter", "2022-05-15", QuarterToDate, 0, "2022-04-01", "2022-05-15"},
		{"quarter to date, fourth quarter", "2022-12-31", QuarterToDate, 0, "2022-10-01", "2022-12-31"},
		{"last quarter", "2022-07-27", LastQuarter, 0, "2022-04-01", "2022-06-30"},
		{"last quarter across a year", "2022-02-15", LastQuarter, 0, "2021-10-01", "2021-12-31"},
		{"last quarter from the second quarter", "2022-05-15", LastQuarter, 0, "2022-01-01", "2022-03-31"},
		{"last quarter from the fourth quarter", "2022-11-02", LastQuarter, 0, "2022-07-01", "2022-09-30"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defs := []RangeDef{{ID: "x", Label: "X", Group: "G", Kind: c.kind, Days: c.days, Default: true}}
			got, err := ResolveRanges(defs, d(c.asOf))
			if err != nil {
				t.Fatal(err)
			}
			if !got[0].From.Equal(d(c.from)) || !got[0].To.Equal(d(c.to)) {
				t.Errorf("got %s..%s, want %s..%s", got[0].From.Format(dateLayout), got[0].To.Format(dateLayout), c.from, c.to)
			}
		})
	}
}

func TestFixedRangesDoNotMove(t *testing.T) {
	defs := []RangeDef{{ID: "q", Label: "Q", Group: "G", Kind: FixedDateRange, From: "2022-04-01", To: "2022-06-30", Default: true}}
	for _, asOf := range []string{"2022-07-27", "2030-01-01"} {
		got, err := ResolveRanges(defs, d(asOf))
		if err != nil {
			t.Fatal(err)
		}
		if !got[0].From.Equal(d("2022-04-01")) || !got[0].To.Equal(d("2022-06-30")) {
			t.Errorf("as-of %s moved the fixed range to %v..%v", asOf, got[0].From, got[0].To)
		}
	}
}

func TestBuiltInListIsValidAndHasTheAgreedRanges(t *testing.T) {
	defs := DefaultRangeDefs()
	if err := ValidateRangeDefs(defs); err != nil {
		t.Fatal(err)
	}
	ranges, err := ResolveRanges(defs, d("2022-07-27"))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, r := range ranges {
		ids = append(ids, r.ID)
	}
	want := "last-7 last-14 last-30 last-60 last-90 last-180 month-to-date last-month quarter-to-date last-quarter 2022-02 2022-03 2022-04 2022-05 2022-06 2022-q1 2022-q2 all-data"
	if strings.Join(ids, " ") != want {
		t.Errorf("ids = %v", ids)
	}
	def, ok := DefaultRange(ranges)
	if !ok || def.ID != "last-30" || def.Label != "Last 30 days" {
		t.Errorf("default = %+v", def)
	}
	groups := map[string]int{}
	for _, r := range ranges {
		groups[r.Group]++
	}
	if groups["Recent"] != 6 || groups["Periods"] != 4 || groups["Fixed"] != 8 {
		t.Errorf("groups = %v", groups)
	}
	all, _ := FindRange(ranges, d("2022-01-23"), d("2022-07-27"))
	if all.ID != "all-data" && all.ID != "last-180" {
		t.Errorf("all data window found as %q", all.ID)
	}
}

func TestValidateRangeDefsRejectsBadLists(t *testing.T) {
	ok := func() RangeDef {
		return RangeDef{ID: "a", Label: "A", Group: "G", Kind: LastDays, Days: 7, Default: true}
	}
	long := strings.Repeat("x", maxTextLen+1)
	many := make([]RangeDef, MaxRangeDefs+1)
	for i := range many {
		many[i] = ok()
	}
	cases := map[string]struct {
		defs []RangeDef
		want string
	}{
		"empty":                    {nil, "no ranges"},
		"too many":                 {many, "at most"},
		"empty id":                 {[]RangeDef{func() RangeDef { r := ok(); r.ID = ""; return r }()}, "id must be"},
		"long id":                  {[]RangeDef{func() RangeDef { r := ok(); r.ID = long; return r }()}, "id must be"},
		"empty label":              {[]RangeDef{func() RangeDef { r := ok(); r.Label = ""; return r }()}, "label must be"},
		"long label":               {[]RangeDef{func() RangeDef { r := ok(); r.Label = long; return r }()}, "label must be"},
		"empty group":              {[]RangeDef{func() RangeDef { r := ok(); r.Group = ""; return r }()}, "group must be"},
		"long group":               {[]RangeDef{func() RangeDef { r := ok(); r.Group = long; return r }()}, "group must be"},
		"duplicate id":             {[]RangeDef{ok(), ok()}, "duplicate id"},
		"no default":               {[]RangeDef{func() RangeDef { r := ok(); r.Default = false; return r }()}, "0 default"},
		"two defaults":             {[]RangeDef{ok(), func() RangeDef { r := ok(); r.ID = "b"; return r }()}, "2 default"},
		"unknown kind":             {[]RangeDef{func() RangeDef { r := ok(); r.Kind = "weekly"; return r }()}, "unknown kind"},
		"days zero":                {[]RangeDef{func() RangeDef { r := ok(); r.Days = 0; return r }()}, "days"},
		"days too many":            {[]RangeDef{func() RangeDef { r := ok(); r.Days = MaxRangeDays + 1; return r }()}, "days"},
		"fixed bad from":           {[]RangeDef{{ID: "f", Label: "F", Group: "G", Kind: FixedDateRange, From: "x", To: "2022-01-02", Default: true}}, "from: want"},
		"fixed bad to":             {[]RangeDef{{ID: "f", Label: "F", Group: "G", Kind: FixedDateRange, From: "2022-01-01", To: "2022-1-2", Default: true}}, "to: want"},
		"fixed reversed":           {[]RangeDef{{ID: "f", Label: "F", Group: "G", Kind: FixedDateRange, From: "2022-02-01", To: "2022-01-01", Default: true}}, "must not be after"},
		"fixed longer than a year": {[]RangeDef{{ID: "f", Label: "F", Group: "G", Kind: FixedDateRange, From: "2021-01-01", To: "2022-01-02", Default: true}}, "must not exceed"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := ValidateRangeDefs(c.defs)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want it to mention %q", err, c.want)
			}
		})
	}
}

func TestValidateRangeDefsAcceptsTheBoundaries(t *testing.T) {
	defs := []RangeDef{
		{ID: strings.Repeat("x", maxTextLen), Label: "A", Group: "G", Kind: LastDays, Days: MaxRangeDays, Default: true},
		{ID: "one", Label: "B", Group: "G", Kind: LastDays, Days: 1},
		{ID: "year", Label: "C", Group: "G", Kind: FixedDateRange, From: "2021-01-02", To: "2022-01-02"}, // 366 days
	}
	if err := ValidateRangeDefs(defs); err != nil {
		t.Fatal(err)
	}
	max := make([]RangeDef, MaxRangeDefs)
	for i := range max {
		max[i] = RangeDef{ID: string(rune('a'+i%26)) + string(rune('a'+i/26)), Label: "L", Group: "G", Kind: MonthToDate, Default: i == 0}
	}
	if err := ValidateRangeDefs(max); err != nil {
		t.Fatalf("%d ranges: %v", MaxRangeDefs, err)
	}
}

func TestResolveRangesRejectsAnInvalidList(t *testing.T) {
	if _, err := ResolveRanges(nil, d("2022-07-27")); err == nil {
		t.Fatal("an empty list must not resolve")
	}
}

func TestFindRangeMatchesBothEndsExactly(t *testing.T) {
	ranges, _ := ResolveRanges(DefaultRangeDefs(), d("2022-07-27"))
	if r, ok := FindRange(ranges, d("2022-06-28"), d("2022-07-27")); !ok || r.ID != "last-30" {
		t.Errorf("last 30 days = %+v, %v", r, ok)
	}
	for name, c := range map[string][2]string{
		"from off by a day": {"2022-06-29", "2022-07-27"},
		"to off by a day":   {"2022-06-28", "2022-07-26"},
		"unlisted":          {"2022-03-05", "2022-03-20"},
	} {
		if _, ok := FindRange(ranges, d(c[0]), d(c[1])); ok {
			t.Errorf("%s should not match", name)
		}
	}
	if _, ok := FindRange(nil, d("2022-06-28"), d("2022-07-27")); ok {
		t.Error("nothing to find in an empty list")
	}
}

func TestDefaultRange(t *testing.T) {
	if _, ok := DefaultRange([]Range{{ID: "a"}}); ok {
		t.Error("no default was marked")
	}
	if r, ok := DefaultRange([]Range{{ID: "a"}, {ID: "b", Default: true}}); !ok || r.ID != "b" {
		t.Errorf("got %+v, %v", r, ok)
	}
}
