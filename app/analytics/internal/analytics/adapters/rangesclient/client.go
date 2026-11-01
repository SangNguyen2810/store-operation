// Package rangesclient is a driven adapter that asks the ranges service which
// date ranges the dashboard offers. It keeps the answer for a short while, so a
// request rarely waits on the network. If the service cannot be reached it
// serves the last good list, and before there has been one the built-in list.
package rangesclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/SangNguyen2810/store-operation/app/analytics/internal/analytics/app"
	"github.com/SangNguyen2810/store-operation/app/analytics/internal/analytics/domain"
)

const (
	// FreshFor is how long an answer is used without asking again.
	FreshFor = 30 * time.Second
	// retryAfter is how long the service is left alone after a failure.
	retryAfter     = 5 * time.Second
	requestTimeout = 500 * time.Millisecond
	// maxBody bounds what is read; 50 ranges are a few kilobytes.
	maxBody    = 1 << 20
	dateLayout = "2006-01-02"
)

// Compile-time check that Client satisfies the port.
var _ app.RangeSource = (*Client)(nil)

// Client fetches GET /api/presets from the ranges service.
type Client struct {
	endpoint string
	hc       *http.Client
	now      func() time.Time
	fallback []domain.Range

	mu        sync.Mutex // held while fetching, so concurrent callers share one request
	list      []domain.Range
	fetchedAt time.Time
	downUntil time.Time
}

// New returns a client for the ranges service at baseURL. asOf anchors the
// built-in list that is served until the service has answered once.
func New(baseURL string, asOf time.Time) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("RANGES_URL: want an http(s) URL such as http://localhost:8083, got %q", baseURL)
	}
	fallback, err := domain.ResolveRanges(domain.DefaultRangeDefs(), asOf)
	if err != nil {
		return nil, err
	}
	return &Client{
		endpoint: u.Scheme + "://" + u.Host + "/api/presets",
		hc:       &http.Client{Timeout: requestTimeout},
		now:      time.Now,
		fallback: fallback,
	}, nil
}

// Ranges never fails: whatever the service does, there is a list to validate against.
func (c *Client) Ranges(ctx context.Context) ([]domain.Range, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.list != nil && now.Sub(c.fetchedAt) < FreshFor {
		return c.list, nil
	}
	if now.Before(c.downUntil) {
		return c.current(), nil
	}
	list, err := c.fetch(ctx)
	if err != nil {
		slog.WarnContext(ctx, "ranges: service unavailable, using the last list", "err", err, "haveList", c.list != nil)
		c.downUntil = now.Add(retryAfter)
		return c.current(), nil
	}
	c.list, c.fetchedAt, c.downUntil = list, now, time.Time{}
	return list, nil
}

// current is the last good list, or the built-in one before there was any.
func (c *Client) current() []domain.Range {
	if c.list != nil {
		return c.list
	}
	return c.fallback
}

type wireRange struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Group   string `json:"group"`
	From    string `json:"from"`
	To      string `json:"to"`
	Default bool   `json:"default"`
}

func (c *Client) fetch(ctx context.Context) ([]domain.Range, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ranges service: status %d", res.StatusCode)
	}
	var in []wireRange
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBody)).Decode(&in); err != nil {
		return nil, fmt.Errorf("ranges service: %w", err)
	}
	return decode(in)
}

// decode checks the list the way a request relies on it: bounded, with valid
// days, and exactly one default.
func decode(in []wireRange) ([]domain.Range, error) {
	if len(in) == 0 || len(in) > domain.MaxRangeDefs {
		return nil, fmt.Errorf("ranges service: %d ranges", len(in))
	}
	out := make([]domain.Range, 0, len(in))
	defaults := 0
	for _, w := range in {
		from, err := time.Parse(dateLayout, w.From)
		if err != nil {
			return nil, fmt.Errorf("ranges service: range %q: from: %w", w.ID, err)
		}
		to, err := time.Parse(dateLayout, w.To)
		if err != nil {
			return nil, fmt.Errorf("ranges service: range %q: to: %w", w.ID, err)
		}
		if from.After(to) {
			return nil, fmt.Errorf("ranges service: range %q: from is after to", w.ID)
		}
		if w.Default {
			defaults++
		}
		out = append(out, domain.Range{ID: w.ID, Label: w.Label, Group: w.Group, From: from, To: to, Default: w.Default})
	}
	if defaults != 1 {
		return nil, errors.New("ranges service: want exactly one default range")
	}
	return out, nil
}
