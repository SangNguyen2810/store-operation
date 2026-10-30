package server

import (
	"encoding/json"
	"net/http"
)

// writeProblem answers with an RFC 7807 body, the same shape the other services use.
func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail,omitempty"`
	}{http.StatusText(status), status, detail})
}
