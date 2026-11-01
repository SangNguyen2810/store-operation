// Package httpadapter is the driving adapter that exposes the date ranges over
// HTTP. It shapes the response; it holds no business rules.
package httpadapter

import (
	"context"
	"net/http"

	"github.com/SangNguyen2810/store-operation/app/ranges/internal/platform/httpx"
	"github.com/SangNguyen2810/store-operation/app/ranges/internal/ranges/domain"
)

const dateLayout = "2006-01-02"

// Ranges is what the handler needs; *app.Service satisfies it.
type Ranges interface {
	Ranges(ctx context.Context) ([]domain.Range, error)
}

type rangeDTO struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Group   string `json:"group"`
	From    string `json:"from"` // YYYY-MM-DD
	To      string `json:"to"`   // YYYY-MM-DD
	Default bool   `json:"default"`
}

// Register adds the ranges route to mux.
func Register(mux *http.ServeMux, svc Ranges) {
	mux.HandleFunc("GET /api/presets", func(w http.ResponseWriter, r *http.Request) {
		ranges, err := svc.Ranges(r.Context())
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		out := make([]rangeDTO, 0, len(ranges))
		for _, rg := range ranges {
			out = append(out, rangeDTO{
				ID: rg.ID, Label: rg.Label, Group: rg.Group,
				From: rg.From.Format(dateLayout), To: rg.To.Format(dateLayout), Default: rg.Default,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	})
}
