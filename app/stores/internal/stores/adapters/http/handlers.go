// Package httpadapter is the driving adapter that exposes the stores over HTTP.
// It shapes the response; it holds no SQL and no business rules.
package httpadapter

import (
	"context"
	"net/http"

	"github.com/SangNguyen2810/store-operation/app/stores/internal/platform/httpx"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/domain"
)

// Stores is what the handler needs; *app.Service satisfies it.
type Stores interface {
	Stores(ctx context.Context) ([]domain.Store, error)
}

// Response types are the API contract. They are separate from the domain so
// the domain can change without breaking clients.
type storeDTO struct {
	ID    int    `json:"id"`
	Label string `json:"label"`

	Manager    string `json:"manager"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	District   string `json:"district"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
	Country    string `json:"country"`

	InventoryItems  int `json:"inventoryItems"`
	ActiveCustomers int `json:"activeCustomers"`
}

// Register adds the stores route to mux.
func Register(mux *http.ServeMux, svc Stores) {
	mux.HandleFunc("GET /api/stores", func(w http.ResponseWriter, r *http.Request) {
		stores, err := svc.Stores(r.Context())
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		out := make([]storeDTO, 0, len(stores))
		for _, s := range stores {
			out = append(out, storeDTO{
				ID: s.ID, Label: s.Label,
				Manager: s.Manager, Phone: s.Phone, Address: s.Address, District: s.District,
				PostalCode: s.PostalCode, City: s.City, Country: s.Country,
				InventoryItems: s.InventoryItems, ActiveCustomers: s.ActiveCustomers,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	})
}
