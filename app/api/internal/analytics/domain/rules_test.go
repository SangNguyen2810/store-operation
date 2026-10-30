package domain

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestPreviousWindow(t *testing.T) {
	cases := []struct {
		name             string
		from, to         string
		wantFrom, wantTo string
	}{
		{"30 days", "2022-06-28", "2022-07-27", "2022-05-29", "2022-06-27"},
		{"single day", "2022-07-27", "2022-07-27", "2022-07-26", "2022-07-26"},
		{"across year", "2022-01-05", "2022-01-11", "2021-12-29", "2022-01-04"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Filter{From: day(c.from), To: day(c.to)}.PreviousWindow()
			if !got.From.Equal(day(c.wantFrom)) || !got.To.Equal(day(c.wantTo)) {
				t.Fatalf("got %s..%s, want %s..%s", got.From.Format("2006-01-02"), got.To.Format("2006-01-02"), c.wantFrom, c.wantTo)
			}
		})
	}
}

func TestEndExclusive(t *testing.T) {
	got := Filter{From: day("2022-07-01"), To: day("2022-07-31")}.EndExclusive()
	if !got.Equal(day("2022-08-01")) {
		t.Errorf("got %s, want 2022-08-01", got.Format("2006-01-02"))
	}
}

func TestGranularityValid(t *testing.T) {
	for _, g := range []Granularity{Day, Week, Month} {
		if !g.Valid() {
			t.Errorf("%q should be valid", g)
		}
	}
	for _, g := range []Granularity{"", "year", "DAY"} {
		if g.Valid() {
			t.Errorf("%q should be invalid", g)
		}
	}
}

func TestAvgPayment(t *testing.T) {
	cases := []struct {
		name string
		in   Totals
		want float64
	}{
		{"no payments", Totals{Revenue: 50}, 0},
		{"exact", Totals{Revenue: 100, Payments: 20}, 5},
		{"rounds to cents", Totals{Revenue: 10, Payments: 3}, 3.33},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.in.AvgPayment(); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestNewMetric(t *testing.T) {
	cases := []struct {
		name        string
		value, prev float64
		wantDelta   *float64
	}{
		{"increase", 150, 100, ptr(50)},
		{"decrease", 50, 100, ptr(-50)},
		{"unchanged", 100, 100, ptr(0)},
		{"rounds to 2 decimals", 100, 3, ptr(3233.33)},
		{"negative previous (refunds)", 50, -100, ptr(-150)},
		{"no previous", 10, 0, nil},
		{"both zero", 0, 0, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := NewMetric(c.value, c.prev)
			if m.Value != c.value || m.Previous != c.prev {
				t.Errorf("value/previous not kept: %+v", m)
			}
			switch {
			case c.wantDelta == nil && m.DeltaPct != nil:
				t.Errorf("delta = %v, want nil", *m.DeltaPct)
			case c.wantDelta != nil && (m.DeltaPct == nil || *m.DeltaPct != *c.wantDelta):
				t.Errorf("delta = %v, want %v", m.DeltaPct, *c.wantDelta)
			}
		})
	}
}

func TestCompareKPIsWiresEachMetric(t *testing.T) {
	cur := Totals{Revenue: 200, Payments: 20, Rentals: 30, ActiveCustomers: 8}
	prev := Totals{Revenue: 100, Payments: 20, Rentals: 60, ActiveCustomers: 4}
	k := CompareKPIs(cur, prev)
	want := map[string]struct{ v, p, d float64 }{
		"revenue":    {200, 100, 100},
		"rentals":    {30, 60, -50},
		"customers":  {8, 4, 100},
		"avgPayment": {10, 5, 100},
	}
	got := map[string]Metric{"revenue": k.Revenue, "rentals": k.Rentals, "customers": k.ActiveCustomers, "avgPayment": k.AvgPayment}
	for name, w := range want {
		m := got[name]
		if m.Value != w.v || m.Previous != w.p || m.DeltaPct == nil || *m.DeltaPct != w.d {
			t.Errorf("%s = %+v, want value %v previous %v delta %v", name, m, w.v, w.p, w.d)
		}
	}
}

func ptr(v float64) *float64 { return &v }
