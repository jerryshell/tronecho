package main

import (
	"reflect"
	"testing"

	"github.com/jerryshell/tronecho/internal/store"
)

func entry(addr, label string, enabled bool, createdAt int64) addrEntry {
	return addrEntry{Address: addr, Label: label, Enabled: enabled, CreatedAt: createdAt}
}

func toMap(entries ...addrEntry) map[string]store.AddrMeta {
	m := make(map[string]store.AddrMeta, len(entries))
	for _, e := range entries {
		m[e.Address] = store.AddrMeta{Label: e.Label, Enabled: e.Enabled, CreatedAt: e.CreatedAt}
	}
	return m
}

func collect(all map[string]store.AddrMeta, limit int) []string {
	var out []string
	cursor := ""
	for {
		items, next := paginateAddrs(all, cursor, limit)
		for _, it := range items {
			out = append(out, it.Address)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	return out
}

func TestPaginateAddrs_Deterministic(t *testing.T) {
	all := toMap(
		entry("TY3N", "d", true, 4),
		entry("TA1Z", "a", true, 1),
		entry("TB2Y", "b", true, 2),
		entry("TC3X", "c", true, 3),
	)

	// With map iteration randomized, any non-deterministic pagination would
	// produce different results across repeated calls. Run enough times to be
	// confident the output is stable.
	var first string
	for i := 0; i < 100; i++ {
		items, next := paginateAddrs(all, "", 2)
		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}
		got := items[0].Address + "," + items[1].Address + ":" + next
		if i == 0 {
			first = got
		} else if got != first {
			t.Fatalf("non-deterministic pagination: first=%s, got=%s", first, got)
		}
	}
	if first != "TA1Z,TB2Y:TB2Y" {
		t.Fatalf("unexpected first page: %s", first)
	}
}

func TestPaginateAddrs_FullWalk(t *testing.T) {
	all := toMap(
		entry("TA1Z", "a", true, 1),
		entry("TB2Y", "b", true, 2),
		entry("TC3X", "c", true, 3),
	)
	want := []string{"TA1Z", "TB2Y", "TC3X"}

	got := collect(all, 2)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPaginateAddrs_LimitExceedsTotal(t *testing.T) {
	all := toMap(entry("TA1Z", "a", true, 1))
	items, next := paginateAddrs(all, "", 100)
	if len(items) != 1 || items[0].Address != "TA1Z" || next != "" {
		t.Fatalf("unexpected result: %v, %s", items, next)
	}
}

func TestPaginateAddrs_EmptyMap(t *testing.T) {
	items, next := paginateAddrs(map[string]store.AddrMeta{}, "", 10)
	if items == nil || len(items) != 0 || next != "" {
		t.Fatalf("expected empty non-nil slice, got %v, %s", items, next)
	}
}

func TestPaginateAddrs_DeletedCursor(t *testing.T) {
	all := toMap(
		entry("TA1Z", "a", true, 1),
		entry("TB2Y", "b", true, 2),
		entry("TC3X", "c", true, 3),
	)
	items, next := paginateAddrs(all, "TB2Y", 10)
	want := []string{"TC3X"}
	var got []string
	for _, it := range items {
		got = append(got, it.Address)
	}
	if !reflect.DeepEqual(got, want) || next != "" {
		t.Fatalf("got %v / %s, want %v / empty", got, next, want)
	}
}

func TestPaginateAddrs_PreservesMetadata(t *testing.T) {
	all := map[string]store.AddrMeta{
		"TA1Z": {Label: "a", Enabled: true, CreatedAt: 42},
	}
	items, _ := paginateAddrs(all, "", 10)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	it := items[0]
	if it.Label != "a" || it.Enabled != true || it.CreatedAt != 42 {
		t.Fatalf("metadata not preserved: %+v", it)
	}
}
