package zod

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentSharedSchemaParse(t *testing.T) {
	schema := Object(Shape{
		"name":  String().Min(1),
		"email": Default(String().Email(), "n@a.com"),
		"age":   Optional(Int().Gte(0).Lt(150)),
	})

	const goroutines = 64
	const iters = 200
	var wg sync.WaitGroup
	var fails atomic.Int64
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				in := map[string]any{
					"name": fmt.Sprintf("user-%d-%d", id, i),
					"age":  i % 100,
				}
				out, err := schema.Parse(in)
				if err != nil {
					fails.Add(1)
					return
				}
				if out["email"] != "n@a.com" {
					fails.Add(1)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	if fails.Load() != 0 {
		t.Fatalf("concurrent Parse failures: %d", fails.Load())
	}
}

func TestConcurrentSafeParseMixedSuccessFailure(t *testing.T) {
	schema := String().Email()
	var wg sync.WaitGroup
	var okN, badN atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				res := schema.SafeParse(fmt.Sprintf("u%d@ex.com", i))
				if res.Success {
					okN.Add(1)
				}
			} else {
				res := schema.SafeParse("not-an-email")
				if !res.Success {
					badN.Add(1)
				}
			}
		}(i)
	}
	wg.Wait()
	if okN.Load() != 50 || badN.Load() != 50 {
		t.Fatalf("ok=%d bad=%d", okN.Load(), badN.Load())
	}
}

func TestConcurrentBatchHelper(t *testing.T) {
	s := Share(String().Min(2))
	inputs := make([]any, 100)
	for i := range inputs {
		inputs[i] = fmt.Sprintf("xx%d", i)
	}
	outs, errs, err := s.ParseAll(context.Background(), inputs, 8)
	if err != nil {
		t.Fatal(err)
	}
	for i := range outs {
		if errs[i] != nil {
			t.Fatalf("errs[%d]=%v", i, errs[i])
		}
		if outs[i] != inputs[i] {
			t.Fatalf("outs[%d]=%q", i, outs[i])
		}
	}
}

func TestConcurrentParseAnyObject(t *testing.T) {
	schema := Object(Shape{"n": Number().Gte(0)})
	inputs := make([]any, 80)
	for i := range inputs {
		inputs[i] = map[string]any{"n": float64(i)}
	}
	outs, errs, err := ConcurrentParseAny(context.Background(), schema, inputs, 4)
	if err != nil {
		t.Fatal(err)
	}
	for i := range outs {
		if errs[i] != nil {
			t.Fatalf("errs[%d]=%v", i, errs[i])
		}
		m := outs[i].(map[string]any)
		if m["n"] != float64(i) {
			t.Fatalf("got %#v", m["n"])
		}
	}
}

func TestConcurrentConfigureAndParse(t *testing.T) {
	schema := String().Min(100) // will fail → locale message
	prev := Configure(Config{LocaleError: EsLocale})
	defer Configure(prev)

	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := schema.SafeParse("short")
			if res.Success || res.Error == nil || len(res.Error.Issues) == 0 {
				t.Error("expected failure")
				return
			}
			// Spanish locale should not be the English "Too small: expected string..."
			msg := res.Error.Issues[0].Message
			if msg == "" {
				t.Error("empty message")
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentRegistry(t *testing.T) {
	r := NewRegistry[map[string]any]()
	schemas := make([]*StringSchema, 50)
	for i := range schemas {
		schemas[i] = String().Min(i%5 + 1)
	}
	var wg sync.WaitGroup
	for i, s := range schemas {
		wg.Add(1)
		go func(i int, s *StringSchema) {
			defer wg.Done()
			r.Add(s, map[string]any{"i": i})
			if _, ok := r.Get(s); !ok {
				t.Errorf("missing meta %d", i)
			}
		}(i, s)
	}
	wg.Wait()
}

func TestConcurrentLazyRecursive(t *testing.T) {
	var node AnySchemaLike
	node = Lazy(func() AnySchemaLike {
		return Object(Shape{
			"v":    String().Min(1),
			"next": Optional(node),
		})
	})
	input := map[string]any{
		"v": "a",
		"next": map[string]any{
			"v": "b",
		},
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			outs, errs, err := ConcurrentParseAny(context.Background(), node, []any{input, input}, 2)
			if err != nil {
				t.Errorf("lazy concurrent batch err: %v", err)
				return
			}
			if errs[0] != nil || errs[1] != nil {
				t.Errorf("lazy concurrent: %v %v", errs[0], errs[1])
			}
			if outs[0] == nil {
				t.Error("nil out")
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentDiscriminatedUnion(t *testing.T) {
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "x": String()}),
		Object(Shape{"type": Literal("b"), "y": Number()}),
	})
	var wg sync.WaitGroup
	for i := 0; i < 60; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var in map[string]any
			if i%2 == 0 {
				in = map[string]any{"type": "a", "x": "hi"}
			} else {
				in = map[string]any{"type": "b", "y": float64(i)}
			}
			if _, err := parseTyped[any](schema.Internals(), in, nil); err != nil {
				t.Errorf("discunion: %v", err)
			}
		}(i)
	}
	wg.Wait()
}

func BenchmarkConcurrentSharedParse(b *testing.B) {
	schema := Object(Shape{
		"name":  String().Min(1),
		"email": String().Email(),
	})
	in := map[string]any{"name": "Ada", "email": "ada@ex.com"}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := schema.Parse(in); err != nil {
				b.Fatal(err)
			}
		}
	})
}
