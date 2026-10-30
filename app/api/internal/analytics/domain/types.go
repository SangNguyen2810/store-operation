// Package domain holds the analytics types and rules. It imports nothing but
// the standard library: no database, no HTTP.
package domain

import "time"

// Filter selects a window of days (both ends inclusive, UTC) and optionally one store.
type Filter struct {
	From    time.Time
	To      time.Time
	StoreID *int
}

type Granularity string

const (
	Day   Granularity = "day"
	Week  Granularity = "week"
	Month Granularity = "month"
)

func (g Granularity) Valid() bool { return g == Day || g == Week || g == Month }

type Store struct {
	ID    int
	Label string
}

// Totals are the raw figures for one window.
type Totals struct {
	Revenue         float64
	Payments        int
	Rentals         int
	ActiveCustomers int
}

// Metric is a value with the same figure for the preceding window.
// DeltaPct is nil when there is no previous value to compare against.
type Metric struct {
	Value    float64
	Previous float64
	DeltaPct *float64
}

type KPIs struct {
	Revenue         Metric
	Rentals         Metric
	ActiveCustomers Metric
	AvgPayment      Metric
}

type RevenuePoint struct {
	Bucket  string // YYYY-MM-DD, start of the bucket
	StoreID int
	Revenue float64
}

type FilmStat struct {
	FilmID  int
	Title   string
	Rentals int
	Revenue float64
}
