package cache

import (
	"strconv"
	"testing"
	"time"
)

func TestMemoryHoldsAtMostMaxEntries(t *testing.T) {
	clk := &clock{t: time.Now()}
	m := newMemory(clk.now)
	for i := range maxEntries + 100 {
		m.set("k"+strconv.Itoa(i), []byte("v"), time.Hour+time.Duration(i)*time.Second)
	}
	if n := m.len(); n != maxEntries {
		t.Fatalf("holds %d entries, want %d", n, maxEntries)
	}
	// The ones closest to expiry went first.
	if _, ok := m.get("k0"); ok {
		t.Error("the entry closest to expiry should have been dropped")
	}
	if _, ok := m.get("k" + strconv.Itoa(maxEntries+99)); !ok {
		t.Error("the newest entry should be kept")
	}
}

func TestMemoryPrefersDroppingExpiredEntries(t *testing.T) {
	clk := &clock{t: time.Now()}
	m := newMemory(clk.now)
	m.set("short", []byte("v"), time.Second)
	for i := range maxEntries - 1 {
		m.set("k"+strconv.Itoa(i), []byte("v"), time.Hour)
	}
	clk.advance(2 * time.Second)
	m.set("new", []byte("v"), time.Hour)
	if _, ok := m.get("k0"); !ok {
		t.Error("a live entry was dropped while an expired one was available")
	}
	if n := m.len(); n != maxEntries-1+1 {
		t.Errorf("holds %d entries, want %d", n, maxEntries)
	}
}

func TestMemoryOverwriteDoesNotEvict(t *testing.T) {
	clk := &clock{t: time.Now()}
	m := newMemory(clk.now)
	for i := range maxEntries {
		m.set("k"+strconv.Itoa(i), []byte("v"), time.Hour)
	}
	m.set("k3", []byte("w"), time.Hour)
	if n := m.len(); n != maxEntries {
		t.Fatalf("holds %d entries after an overwrite, want %d", n, maxEntries)
	}
	if v, _ := m.get("k3"); string(v) != "w" {
		t.Errorf("k3 = %q", v)
	}
}

func TestMemoryExpiry(t *testing.T) {
	clk := &clock{t: time.Now()}
	m := newMemory(clk.now)
	m.set("k", []byte("v"), time.Minute)
	clk.advance(time.Minute - time.Nanosecond)
	if _, ok := m.get("k"); !ok {
		t.Fatal("expired too early")
	}
	clk.advance(time.Nanosecond)
	if _, ok := m.get("k"); ok {
		t.Fatal("served at its expiry instant")
	}
	if m.len() != 0 {
		t.Fatal("an expired read should remove the entry")
	}
}
