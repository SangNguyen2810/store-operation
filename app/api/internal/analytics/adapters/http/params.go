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
	defaultLimit  = 10
	maxLimit      = 50
	defaultGranul = domain.Day
)

// parseFilter reads from, to and store. The window must be one of the offered
// ranges (GET /api/presets); with neither date it is the default range.
func parseFilter(r *http.Request, ranges []domain.Range) (domain.Filter, error) {
	q := r.URL.Query()
	var f domain.Filter

	fromS, toS := q.Get("from"), q.Get("to")
	switch {
	case fromS == "" && toS == "":
		def, ok := domain.DefaultRange(ranges)
		if !ok {
			return f, errors.New("no default range is configured")
		}
		f.From, f.To = def.From, def.To
	case fromS == "" || toS == "":
		return f, errors.New("from and to must be sent together")
	default:
		var err error
		if f.From, err = time.Parse(dateLayout, fromS); err != nil {
			return f, errors.New("from: want YYYY-MM-DD")
		}
		if f.To, err = time.Parse(dateLayout, toS); err != nil {
			return f, errors.New("to: want YYYY-MM-DD")
		}
		if _, ok := domain.FindRange(ranges, f.From, f.To); !ok {
			return f, errors.New("from and to: not one of the offered ranges (see GET /api/presets)")
		}
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
