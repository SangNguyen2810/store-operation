package domain

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestPreviousWindow(t *testing.T) {
	cases := []struct {
		name             string
		from, to         string
		wantFrom, wantTo string
	}{
		{"30 days", "2022-06-28", "2022-07-27", "2022-05-29", "2022-06-27"},
		{"single day", "2022-07-27", "2022-07-27", "2022-07-26", "2022-07-26"},
		{"across year", "2022-01-05", "2022-01-11", "2021-12-29", "2022-01-04"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Filter{From: day(c.from), To: day(c.to)}.PreviousWindow()
			if !got.From.Equal(day(c.wantFrom)) || !got.To.Equal(day(c.wantTo)) {
				t.Fatalf("got %s..%s, want %s..%s", got.From.Format("2006-01-02"), got.To.Format("2006-01-02"), c.wantFrom, c.wantTo)
			}
		})
	}
}
