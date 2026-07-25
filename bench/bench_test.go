package bench

import (
	"context"
	"fmt"
	"testing"

	z "github.com/Oudwins/zog"
	"github.com/iKunalChhabra/go-zod"
)

// ---------------------------------------------------------------------------
// FlatUser
// ---------------------------------------------------------------------------

func BenchmarkFlatUser_GoZod(b *testing.B) {
	schema := gozodFlatUser()
	data := validFlatUserMap()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFlatUser_GoZod_Parallel(b *testing.B) {
	schema := gozodFlatUser()
	data := validFlatUserMap()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := schema.Parse(data); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkFlatUser_Validator(b *testing.B) {
	v := playgroundValidator()
	u := validFlatUser()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := v.Struct(&u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFlatUser_Validator_Parallel(b *testing.B) {
	v := playgroundValidator()
	u := validFlatUser()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := v.Struct(&u); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkFlatUser_Zog(b *testing.B) {
	schema := zogFlatUser()
	data := map[string]any{
		"name":  "alice",
		"email": "alice@example.com",
		"age":   30,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var dest FlatUser
		if errs := schema.Parse(data, &dest); errs != nil {
			b.Fatal(errs)
		}
	}
}

func BenchmarkFlatUser_Zog_Parallel(b *testing.B) {
	schema := zogFlatUser()
	data := map[string]any{
		"name":  "alice",
		"email": "alice@example.com",
		"age":   30,
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var dest FlatUser
			if errs := schema.Parse(data, &dest); errs != nil {
				b.Fatal(errs)
			}
		}
	})
}

func BenchmarkFlatUser_Handwritten(b *testing.B) {
	u := validFlatUser()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := handwrittenFlatUser(&u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFlatUser_Handwritten_Parallel(b *testing.B) {
	u := validFlatUser()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := handwrittenFlatUser(&u); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Nested
// ---------------------------------------------------------------------------

func BenchmarkNested_GoZod(b *testing.B) {
	schema := gozodNestedUser()
	data := validNestedUserMap()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNested_GoZod_Parallel(b *testing.B) {
	schema := gozodNestedUser()
	data := validNestedUserMap()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := schema.Parse(data); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkNested_Validator(b *testing.B) {
	v := playgroundValidator()
	u := validNestedUser()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := v.Struct(&u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNested_Validator_Parallel(b *testing.B) {
	v := playgroundValidator()
	u := validNestedUser()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := v.Struct(&u); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkNested_Zog(b *testing.B) {
	schema := zogNestedUser()
	data := map[string]any{
		"name":  "alice",
		"email": "alice@example.com",
		"age":   30,
		"address": map[string]any{
			"city": "Seattle",
			"zip":  "98101",
		},
		"tags": []any{"go", "zod", "perf"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var dest NestedUser
		if errs := schema.Parse(data, &dest); errs != nil {
			b.Fatal(errs)
		}
	}
}

func BenchmarkNested_Zog_Parallel(b *testing.B) {
	schema := zogNestedUser()
	data := map[string]any{
		"name":  "alice",
		"email": "alice@example.com",
		"age":   30,
		"address": map[string]any{
			"city": "Seattle",
			"zip":  "98101",
		},
		"tags": []any{"go", "zod", "perf"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var dest NestedUser
			if errs := schema.Parse(data, &dest); errs != nil {
				b.Fatal(errs)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// ArrayN — sequential + ParseParallelSlice
// ---------------------------------------------------------------------------

func BenchmarkArrayN_GoZod_Sequential(b *testing.B) {
	schema := gozodFlatUser()
	for _, n := range []int{100, 1000, 10000} {
		b.Run(itoa(n), func(b *testing.B) {
			data := makeFlatUserMapSlice(n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, item := range data {
					if _, err := schema.Parse(item); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkArrayN_GoZod_ParseParallelSlice(b *testing.B) {
	schema := gozodFlatUser()
	for _, n := range []int{100, 1000, 10000} {
		b.Run(itoa(n), func(b *testing.B) {
			data := makeFlatUserMapSlice(n)
			opts := zod.ParallelOpts{}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := zod.ParseParallelSlice(context.Background(), schema, data, opts); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkArrayN_Validator(b *testing.B) {
	v := playgroundValidator()
	for _, n := range []int{100, 1000, 10000} {
		b.Run(itoa(n), func(b *testing.B) {
			data := makeFlatUserSlice(n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j := range data {
					if err := v.Struct(&data[j]); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkArrayN_Zog(b *testing.B) {
	schema := zogFlatUser()
	for _, n := range []int{100, 1000, 10000} {
		b.Run(itoa(n), func(b *testing.B) {
			items := make([]map[string]any, n)
			for i := range items {
				items[i] = map[string]any{
					"name":  "alice",
					"email": "alice@example.com",
					"age":   30,
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j := range items {
					var dest FlatUser
					if errs := schema.Parse(items[j], &dest); errs != nil {
						b.Fatal(errs)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// StringFormats
// ---------------------------------------------------------------------------

func BenchmarkStringFormats_GoZod(b *testing.B) {
	schema := gozodFormats()
	data := validFormatMap()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringFormats_Validator(b *testing.B) {
	v := playgroundValidator()
	u := validFormatPayload()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := v.Struct(&u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringFormats_Zog(b *testing.B) {
	schema := zogFormats()
	data := map[string]any{
		"email": "alice@example.com",
		"uuid":  "550e8400-e29b-41d4-a716-446655440000",
		"url":   "https://example.com/path",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var dest FormatPayload
		if errs := schema.Parse(data, &dest); errs != nil {
			b.Fatal(errs)
		}
	}
}

// ---------------------------------------------------------------------------
// FailurePath — invalid data + error construction
// ---------------------------------------------------------------------------

func BenchmarkFailurePath_GoZod(b *testing.B) {
	schema := gozodFlatUser()
	data := invalidFlatUserMap()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := schema.Parse(data)
		if err == nil {
			b.Fatal("expected error")
		}
		_ = err.Error()
	}
}

func BenchmarkFailurePath_Validator(b *testing.B) {
	v := playgroundValidator()
	u := invalidFlatUser()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := v.Struct(&u)
		if err == nil {
			b.Fatal("expected error")
		}
		_ = err.Error()
	}
}

func BenchmarkFailurePath_Zog(b *testing.B) {
	schema := zogFlatUser()
	data := map[string]any{
		"name":  "ab",
		"email": "not-email",
		"age":   200,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var dest FlatUser
		errs := schema.Parse(data, &dest)
		if errs == nil {
			b.Fatal("expected error")
		}
		_ = z.Issues.Prettify(errs)
		_ = fmt.Sprintf("%d", len(errs))
	}
}

func itoa(n int) string {
	switch n {
	case 100:
		return "100"
	case 1000:
		return "1000"
	case 10000:
		return "10000"
	default:
		return "n"
	}
}
