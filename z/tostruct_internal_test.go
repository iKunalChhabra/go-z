package z

import (
	"encoding/json"
	"math"
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

func TestScalarToInt64Boundaries(t *testing.T) {
	ok := []struct {
		in   any
		want int64
	}{
		{int8(-8), -8},
		{int16(300), 300},
		{int32(-70000), -70000},
		{int64(math.MaxInt64), math.MaxInt64},
		{int64(math.MinInt64), math.MinInt64},
		{uint8(200), 200},
		{uint16(60000), 60000},
		{uint32(4000000000), 4000000000},
		{uint64(math.MaxInt64), math.MaxInt64},
		{float64(0), 0},
		{float64(math.MinInt64), math.MinInt64}, // -2^63 is exact in float64
		{float64(-1), -1},
		{json.Number("9007199254740993"), 9007199254740993},
		{json.Number("-9007199254740993"), -9007199254740993},
	}
	for _, tc := range ok {
		got, err := scalarToInt64(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("scalarToInt64(%v) = %v, %v; want %v", tc.in, got, err, tc.want)
		}
	}

	bad := []any{
		float64(3.9), float64(-0.5),
		math.NaN(), math.Inf(1), math.Inf(-1),
		float64(math.MaxInt64), // rounds to 2^63, above int64
		float64(math.MaxUint64),
		uint64(math.MaxUint64), // above int64
		json.Number("1.5"), json.Number("1e300"), json.Number("abc"),
		"65", nil, true,
	}
	for _, in := range bad {
		if got, err := scalarToInt64(in); err == nil {
			t.Errorf("scalarToInt64(%v) = %v, want error", in, got)
		}
	}
}

func TestScalarToUint64Boundaries(t *testing.T) {
	ok := []struct {
		in   any
		want uint64
	}{
		{int(0), 0},
		{int8(8), 8},
		{int64(math.MaxInt64), math.MaxInt64},
		{uint64(math.MaxUint64), math.MaxUint64},
		{uint32(4000000000), 4000000000},
		{float64(0), 0},
		{float64(1 << 53), 1 << 53},
		{json.Number("18446744073709551615"), math.MaxUint64},
	}
	for _, tc := range ok {
		got, err := scalarToUint64(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("scalarToUint64(%v) = %v, %v; want %v", tc.in, got, err, tc.want)
		}
	}

	bad := []any{
		int(-1), int8(-1), int64(math.MinInt64),
		float64(-1), float64(3.9),
		math.NaN(), math.Inf(1), math.Inf(-1),
		float64(18446744073709551616.0), // 2^64
		json.Number("-1"), json.Number("1.5"), json.Number("18446744073709551616"),
		"1", nil,
	}
	for _, in := range bad {
		if got, err := scalarToUint64(in); err == nil {
			t.Errorf("scalarToUint64(%v) = %v, want error", in, got)
		}
	}
}
