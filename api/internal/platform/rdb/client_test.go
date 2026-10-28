package rdb

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestOpenReachable(t *testing.T) {
	mr := miniredis.RunT(t)
	c, err := Open(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Set(context.Background(), "k", "v", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if got, _ := mr.Get("k"); got != "v" {
		t.Errorf("stored %q", got)
	}
}

func TestOpenRejectsBadURL(t *testing.T) {
	if _, err := Open(context.Background(), "http://not-redis"); err == nil {
		t.Error("want an error for a non-redis url")
	}
}

// A dead Redis must not stop the API from starting, and must fail fast.
func TestOpenUnreachableOnlyWarns(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()

	start := time.Now()
	c, err := Open(context.Background(), "redis://"+addr)
	if err != nil {
		t.Fatalf("unreachable redis must not be an error: %v", err)
	}
	defer c.Close()
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("took %s, timeouts should keep this short", d)
	}
	if !strings.Contains(buf.String(), "redis unreachable") {
		t.Errorf("no warning logged: %s", buf.String())
	}
}
