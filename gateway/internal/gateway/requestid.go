package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
)

const requestIDHeader = "X-Request-Id"

// A client-supplied id is reused only when it is short and log-safe.
var safeID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// withRequestID makes sure every request carries an id: it is sent upstream
// and echoed on the response so one request can be followed across services.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !safeID.MatchString(id) {
			id = newID()
		}
		r.Header.Set(requestIDHeader, id)
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r)
	})
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // never fails on supported platforms
	return hex.EncodeToString(b)
}
