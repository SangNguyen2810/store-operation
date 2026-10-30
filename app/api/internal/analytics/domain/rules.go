package domain

import (
	"math"
	"time"
)

// PreviousWindow is the window of equal length that ends the day before f starts.
func (f Filter) PreviousWindow() Filter {
	days := int(f.To.Sub(f.From).Hours()/24) + 1
	prevTo := f.From.AddDate(0, 0, -1)
	return Filter{From: prevTo.AddDate(0, 0, -(days - 1)), To: prevTo, StoreID: f.StoreID}
}

// EndExclusive is the instant after the last day of the window.
func (f Filter) EndExclusive() time.Time { return f.To.AddDate(0, 0, 1) }

// AvgPayment is revenue per payment, or 0 when there were no payments.
func (t Totals) AvgPayment() float64 {
	if t.Payments == 0 {
		return 0
	}
	return round2(t.Revenue / float64(t.Payments))
}

// NewMetric pairs a value with the previous one. DeltaPct stays nil when
// previous is 0, because there is nothing to compare against.
func NewMetric(value, previous float64) Metric {
	m := Metric{Value: value, Previous: previous}
	if previous != 0 {
		d := round2((value - previous) / previous * 100)
		m.DeltaPct = &d
	}
	return m
}

// CompareKPIs builds the four KPIs from the current and previous window totals.
func CompareKPIs(cur, prev Totals) KPIs {
	return KPIs{
		Revenue:         NewMetric(cur.Revenue, prev.Revenue),
		Rentals:         NewMetric(float64(cur.Rentals), float64(prev.Rentals)),
		ActiveCustomers: NewMetric(float64(cur.ActiveCustomers), float64(prev.ActiveCustomers)),
		AvgPayment:      NewMetric(cur.AvgPayment(), prev.AvgPayment()),
	}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
