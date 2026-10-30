package httpadapter

import "github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"

// Response types are the API contract. They are separate from the domain so
// the domain can change without breaking clients.

type storeDTO struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

type metricDTO struct {
	Value    float64  `json:"value"`
	Previous float64  `json:"previous"`
	DeltaPct *float64 `json:"deltaPct"`
}

type kpisDTO struct {
	Revenue         metricDTO `json:"revenue"`
	Rentals         metricDTO `json:"rentals"`
	ActiveCustomers metricDTO `json:"activeCustomers"`
	AvgPayment      metricDTO `json:"avgPayment"`
}

type revenuePointDTO struct {
	Bucket  string  `json:"bucket"` // YYYY-MM-DD, start of the bucket
	StoreID int     `json:"storeId"`
	Revenue float64 `json:"revenue"`
}

type filmDTO struct {
	FilmID  int     `json:"filmId"`
	Title   string  `json:"title"`
	Rentals int     `json:"rentals"`
	Revenue float64 `json:"revenue"`
}

func toStores(in []domain.Store) []storeDTO {
	out := make([]storeDTO, 0, len(in))
	for _, s := range in {
		out = append(out, storeDTO{ID: s.ID, Label: s.Label})
	}
	return out
}

func toMetric(m domain.Metric) metricDTO {
	return metricDTO{Value: m.Value, Previous: m.Previous, DeltaPct: m.DeltaPct}
}

func toKPIs(k domain.KPIs) kpisDTO {
	return kpisDTO{
		Revenue:         toMetric(k.Revenue),
		Rentals:         toMetric(k.Rentals),
		ActiveCustomers: toMetric(k.ActiveCustomers),
		AvgPayment:      toMetric(k.AvgPayment),
	}
}

func toRevenue(in []domain.RevenuePoint) []revenuePointDTO {
	out := make([]revenuePointDTO, 0, len(in))
	for _, p := range in {
		out = append(out, revenuePointDTO{Bucket: p.Bucket, StoreID: p.StoreID, Revenue: p.Revenue})
	}
	return out
}

func toFilms(in []domain.FilmStat) []filmDTO {
	out := make([]filmDTO, 0, len(in))
	for _, f := range in {
		out = append(out, filmDTO{FilmID: f.FilmID, Title: f.Title, Rentals: f.Rentals, Revenue: f.Revenue})
	}
	return out
}
