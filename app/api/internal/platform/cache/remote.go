package cache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// maxValue matches the redis service's own limit.
	maxValue  = 1 << 20
	ttlHeader = "X-Cache-Ttl"
)

// remote talks to the redis service over HTTP: GET and PUT on /v1/cache/{key}.
type remote struct {
	base string
	hc   *http.Client
}

// errMiss means the service answered and has no such key. Every other error
// means the service itself is unavailable.
var errMiss = errors.New("cache miss")

func newRemote(rawURL string, timeout time.Duration) (*remote, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("CACHE_URL: want an http(s) URL such as http://localhost:8082, got %q", rawURL)
	}
	return &remote{base: u.Scheme + "://" + u.Host, hc: &http.Client{Timeout: timeout}}, nil
}

func (r *remote) endpoint(key string) string { return r.base + "/v1/cache/" + url.PathEscape(key) }

// get returns the value and how long it has left; ttl 0 means it never expires.
func (r *remote) get(ctx context.Context, key string) ([]byte, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint(key), nil)
	if err != nil {
		return nil, 0, err
	}
	res, err := r.hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, 0, errMiss
	default:
		return nil, 0, fmt.Errorf("redis service: status %d", res.StatusCode)
	}
	val, err := io.ReadAll(io.LimitReader(res.Body, maxValue+1))
	if err != nil {
		return nil, 0, err
	}
	if len(val) > maxValue {
		return nil, 0, errors.New("redis service: value too large")
	}
	var ttl time.Duration
	if s := res.Header.Get(ttlHeader); s != "" {
		secs, err := strconv.Atoi(s)
		if err != nil || secs < 0 {
			return nil, 0, fmt.Errorf("redis service: bad %s %q", ttlHeader, s)
		}
		ttl = time.Duration(secs) * time.Second
	}
	return val, ttl, nil
}

func (r *remote) put(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	u := r.endpoint(key) + "?ttl=" + url.QueryEscape(ttl.Round(time.Second).String())
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(val))
	if err != nil {
		return err
	}
	res, err := r.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("redis service: status %d", res.StatusCode)
	}
	return nil
}
