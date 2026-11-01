// Package httpx holds the HTTP plumbing shared by every context: JSON and
// problem responses, request logging, and the service-wide endpoints.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Problem is an RFC 7807 error body.
type Problem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write response", "err", err)
	}
}

func WriteProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Problem{Title: http.StatusText(status), Status: status, Detail: detail})
}

// Fail logs err and answers 500 without leaking it to the client.
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "path", r.URL.Path, "err", err)
	WriteProblem(w, http.StatusInternalServerError, "internal error")
}
