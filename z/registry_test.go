package z

import (
	"fmt"
	"sync"
	"testing"
)

// Ported from classic/tests/registries.test.ts (add/get/has/remove/clear/id/describe/meta).

func TestRegistryAddGetHasRemove(t *testing.T) {
	reg := NewRegistry[map[string]any]()
	a := String()
	reg.Add(a, map[string]any{"field": "sup"})
	if !reg.Has(a) {
		t.Fatal("expected has")
	}
	got, ok := reg.Get(a)
	if !ok || got["field"] != "sup" {
		t.Fatalf("get: %#v ok=%v", got, ok)
	}
	reg.Remove(a)
	if reg.Has(a) {
		t.Fatal("expected removed")
	}
	if _, ok := reg.Get(a); ok {
		t.Fatal("get after remove")
	}
}

func TestRegistryTypedMeta(t *testing.T) {
	type meta struct {
		Name        string
		Description string
	}
	reg := NewRegistry[meta]()
	a := String()
	reg.Add(a, meta{Name: "hello", Description: "world"})
	got, ok := reg.Get(a)
	if !ok || got.Name != "hello" || got.Description != "world" {
		t.Fatalf("got %#v", got)
	}
}

func TestRegistryClear(t *testing.T) {
	reg := NewRegistry[struct{}]()
	a := String()
	reg.Add(a, struct{}{})
	reg.Clear()
	if reg.Has(a) {
		t.Fatal("clear should remove")
	}
}

func TestRegistryIDIndex(t *testing.T) {
	reg := NewRegistry[map[string]any]()
	a := String()
	b := Number()
	reg.Add(a, map[string]any{"id": "shared-id"})
	if got, ok := reg.GetByID("shared-id"); !ok || got.Internals() != a.Internals() {
		t.Fatalf("id map a: %#v ok=%v", got, ok)
	}
	// Silent overwrite — same as.
	reg.Add(b, map[string]any{"id": "shared-id"})
	if got, ok := reg.GetByID("shared-id"); !ok || got.Internals() != b.Internals() {
		t.Fatalf("id map b: %#v ok=%v", got, ok)
	}
	reg.Remove(b)
	if _, ok := reg.GetByID("shared-id"); ok {
		t.Fatal("id should be cleared on remove")
	}
}

func TestGlobalDescribeMeta(t *testing.T) {
	GlobalRegistry.Clear()
	t.Cleanup(func() { GlobalRegistry.Clear() })

	a := String()
	Describe(a, "Hello")
	if GetDescription(a) != "Hello" {
		t.Fatalf("description: %q", GetDescription(a))
	}
	Meta(a, map[string]any{"name": "hello"})
	got, ok := GlobalRegistry.Get(a)
	if !ok {
		t.Fatal("missing meta")
	}
	if got["description"] != "Hello" || got["name"] != "hello" {
		t.Fatalf("merged meta: %#v", got)
	}
	Meta(a, map[string]any{"id": "str-1", "b": true})
	if got, ok := GlobalRegistry.GetByID("str-1"); !ok || got.Internals() != a.Internals() {
		t.Fatalf("global id: %#v ok=%v", got, ok)
	}
}

func TestGlobalRegistryDistinctSchemas(t *testing.T) {
	GlobalRegistry.Clear()
	t.Cleanup(func() { GlobalRegistry.Clear() })

	a1 := String()
	a2 := String()
	Describe(a2, "only a2")
	if GetDescription(a1) != "" {
		t.Fatal("a1 should not inherit a2 description")
	}
	if GetDescription(a2) != "only a2" {
		t.Fatal("a2 description missing")
	}
}

// Describe and Meta used to read with Get and write with Add, so two goroutines
// could read the same metadata and each discard the other's field.
func TestConcurrentMetadataMergesDoNotLoseFields(t *testing.T) {
	schema := String()
	const n = 50
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			Meta(schema, map[string]any{fmt.Sprintf("key%d", i): i})
		})
	}
	wg.Go(func() { Describe(schema, "described") })
	wg.Wait()

	meta, ok := GlobalRegistry.Get(schema)
	if !ok {
		t.Fatal("no metadata registered")
	}
	missing := 0
	for i := range n {
		if _, ok := meta[fmt.Sprintf("key%d", i)]; !ok {
			missing++
		}
	}
	if missing > 0 {
		t.Errorf("%d of %d merged fields were lost", missing, n)
	}
	if meta["description"] != "described" {
		t.Errorf("description = %#v", meta["description"])
	}
}
