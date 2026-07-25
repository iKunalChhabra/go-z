package z

import (
	"reflect"
	"testing"
)

// The decode plan depends only on the target type, so building many schemas for
// the same struct must not grow the cache.
func TestToStructPlanCacheIsKeyedByType(t *testing.T) {
	type cacheProbe struct {
		A string `json:"a"`
	}
	first := planFor(reflect.TypeFor[cacheProbe]())
	for range 100 {
		_ = ToStruct[cacheProbe](Object(Shape{"a": String()}))
	}
	entries := 0
	decodePlans.Range(func(k, _ any) bool {
		if k == reflect.TypeFor[cacheProbe]() {
			entries++
		}
		return true
	})
	if entries != 1 {
		t.Fatalf("cache holds %d entries for one type", entries)
	}
	if planFor(reflect.TypeFor[cacheProbe]()) != first {
		t.Error("the plan should be reused, not rebuilt")
	}
}
