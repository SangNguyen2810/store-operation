// Package analytics answers read-only revenue and rental questions about
// Pagila. It owns all of its SQL; nothing else may query these tables.
package analytics

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
	ID    int    `json:"id"`
	Label string `json:"label"`
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
	Value    float64  `json:"value"`
	Previous float64  `json:"previous"`
	DeltaPct *float64 `json:"deltaPct"`
}

type KPIs struct {
	Revenue         Metric `json:"revenue"`
	Rentals         Metric `json:"rentals"`
	ActiveCustomers Metric `json:"activeCustomers"`
	AvgPayment      Metric `json:"avgPayment"`
}

type RevenuePoint struct {
	Bucket  string  `json:"bucket"` // YYYY-MM-DD, start of the bucket
	StoreID int     `json:"storeId"`
	Revenue float64 `json:"revenue"`
}

type FilmStat struct {
	FilmID  int     `json:"filmId"`
	Title   string  `json:"title"`
	Rentals int     `json:"rentals"`
	Revenue float64 `json:"revenue"`
}
