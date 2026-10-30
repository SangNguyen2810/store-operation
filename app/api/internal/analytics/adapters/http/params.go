package httpadapter

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
)

const (
	dateLayout    = "2006-01-02"
	defaultDays   = 30
	maxRangeDays  = 366
	defaultLimit  = 10
	maxLimit      = 50
	defaultGranul = domain.Day
)

// parseFilter reads from, to and store. Missing dates default to the 30 days ending at asOf.
func parseFilter(r *http.Request, asOf time.Time) (domain.Filter, error) {
	q := r.URL.Query()
	f := domain.Filter{To: asOf, From: asOf.AddDate(0, 0, -(defaultDays - 1))}

	var err error
	if v := q.Get("to"); v != "" {
		if f.To, err = time.Parse(dateLayout, v); err != nil {
			return f, fmt.Errorf("to: want YYYY-MM-DD")
		}
		if q.Get("from") == "" {
			f.From = f.To.AddDate(0, 0, -(defaultDays - 1))
		}
	}
	if v := q.Get("from"); v != "" {
		if f.From, err = time.Parse(dateLayout, v); err != nil {
			return f, fmt.Errorf("from: want YYYY-MM-DD")
		}
	}
	if f.From.After(f.To) {
		return f, errors.New("from must not be after to")
	}
	if f.To.Sub(f.From).Hours()/24 >= maxRangeDays {
		return f, fmt.Errorf("range must not exceed %d days", maxRangeDays)
	}
	if v := q.Get("store"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil || id < 0 {
			return f, errors.New("store: want a non-negative integer")
		}
		f.StoreID = &id
	}
	return f, nil
}

func parseGranularity(r *http.Request) (domain.Granularity, error) {
	v := r.URL.Query().Get("granularity")
	if v == "" {
		return defaultGranul, nil
	}
	g := domain.Granularity(v)
	if !g.Valid() {
		return "", errors.New("granularity: want day, week or month")
	}
	return g, nil
}

func parseLimit(r *http.Request) (int, error) {
	v := r.URL.Query().Get("limit")
	if v == "" {
		return defaultLimit, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > maxLimit {
		return 0, fmt.Errorf("limit: want 1..%d", maxLimit)
	}
	return n, nil
}
