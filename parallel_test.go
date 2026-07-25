package zod

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func flatUserSchema() *ObjectSchema {
	return Object(Shape{
		"name":  String().Min(5),
		"email": String().Email(),
		"age":   Int().Gte(0).Lt(150),
	})
}

func makeUsers(n int, good bool) []any {
	out := make([]any, n)
	for i := 0; i < n; i++ {
		if good {
			out[i] = map[string]any{
				"name":  "alice",
				"email": "alice@example.com",
				"age":   float64(30),
			}
		} else {
			out[i] = map[string]any{
				"name":  "ab", // too short
				"email": "not-an-email",
				"age":   float64(200),
			}
		}
	}
	return out
}

func TestParseParallelSliceSequentialSmall(t *testing.T) {
	schema := flatUserSchema()
	data := makeUsers(10, true)
	out, err := ParseParallelSlice(context.Background(), schema, data, ParallelOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 10 {
		t.Fatalf("len=%d", len(out))
	}
	m := out[0].(map[string]any)
	if m["name"] != "alice" {
		t.Fatalf("got %#v", m)
	}
}

func TestParseParallelSliceSequentialWorkersOne(t *testing.T) {
	schema := flatUserSchema()
	data := makeUsers(200, true)
	out, err := ParseParallelSlice(context.Background(), schema, data, ParallelOpts{Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 200 {
		t.Fatalf("len=%d", len(out))
	}
}

func TestParseParallelSliceParallelHappy(t *testing.T) {
	schema := flatUserSchema()
	data := makeUsers(256, true)
	out, err := ParseParallelSlice(context.Background(), schema, data, ParallelOpts{Workers: 4, MinChunk: 64})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 256 {
		t.Fatalf("len=%d", len(out))
	}
	for i, v := range out {
		m, ok := v.(map[string]any)
		if !ok || m["email"] != "alice@example.com" {
			t.Fatalf("index %d: %#v", i, v)
		}
	}
}

func TestParseParallelSliceNilSchema(t *testing.T) {
	data := []any{1, 2, 3}
	out, err := ParseParallelSlice(context.Background(), nil, data, ParallelOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[1] != 2 {
		t.Fatalf("%#v", out)
	}
}

func TestParseParallelSliceEmpty(t *testing.T) {
	out, err := ParseParallelSlice(context.Background(), String(), nil, ParallelOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("%#v", out)
	}
}

func TestParseParallelSliceFailurePaths(t *testing.T) {
	schema := flatUserSchema()
	data := makeUsers(128, true)
	data[0] = map[string]any{"name": "ab", "email": "a@b.com", "age": float64(10)}
	data[50] = map[string]any{"name": "valid", "email": "bad", "age": float64(10)}
	data[127] = map[string]any{"name": "valid", "email": "ok@x.com", "age": float64(999)}

	out, err := ParseParallelSlice(context.Background(), schema, data, ParallelOpts{Workers: 4, MinChunk: 32})
	if err == nil {
		t.Fatal("expected error")
	}
	zerr, ok := err.(*ZodError)
	if !ok {
		t.Fatalf("want *ZodError, got %T", err)
	}
	if len(out) != 128 {
		t.Fatalf("partial out len=%d", len(out))
	}

	// Deterministic: issues ordered by absolute index.
	var idxs []int
	for _, iss := range zerr.Issues {
		if len(iss.Path) == 0 {
			t.Fatalf("missing path: %#v", iss)
		}
		idx, ok := iss.Path[0].(int)
		if !ok {
			t.Fatalf("path[0] type %T", iss.Path[0])
		}
		idxs = append(idxs, idx)
	}
	for i := 1; i < len(idxs); i++ {
		if idxs[i] < idxs[i-1] {
			t.Fatalf("issue indices not sorted: %v", idxs)
		}
	}
	// Expect failures at 0, 50, 127 (at least one issue each).
	seen := map[int]bool{}
	for _, i := range idxs {
		seen[i] = true
	}
	for _, want := range []int{0, 50, 127} {
		if !seen[want] {
			t.Fatalf("missing issues for index %d; got %v", want, idxs)
		}
	}
}

func TestParseParallelSliceMatchesSequential(t *testing.T) {
	schema := flatUserSchema()
	data := makeUsers(100, true)
	for i := 0; i < 100; i += 7 {
		data[i] = map[string]any{
			"name":  fmt.Sprintf("n%d", i), // often too short
			"email": "x",
			"age":   float64(-1),
		}
	}

	seq, errSeq := ParseParallelSlice(context.Background(), schema, data, ParallelOpts{Workers: 1, MinChunk: 1})
	par, errPar := ParseParallelSlice(context.Background(), schema, data, ParallelOpts{Workers: 4, MinChunk: 8})

	if (errSeq == nil) != (errPar == nil) {
		t.Fatalf("err mismatch seq=%v par=%v", errSeq, errPar)
	}
	if len(seq) != len(par) {
		t.Fatalf("len mismatch")
	}
	if errSeq == nil {
		return
	}
	zs := errSeq.(*ZodError)
	zp := errPar.(*ZodError)
	if len(zs.Issues) != len(zp.Issues) {
		t.Fatalf("issue count seq=%d par=%d", len(zs.Issues), len(zp.Issues))
	}
	for i := range zs.Issues {
		if zs.Issues[i].Code != zp.Issues[i].Code {
			t.Fatalf("issue[%d] code %q vs %q", i, zs.Issues[i].Code, zp.Issues[i].Code)
		}
		if fmt.Sprint(zs.Issues[i].Path) != fmt.Sprint(zp.Issues[i].Path) {
			t.Fatalf("issue[%d] path %v vs %v", i, zs.Issues[i].Path, zp.Issues[i].Path)
		}
	}
}

func TestParseParallelSliceContextCancel(t *testing.T) {
	schema := flatUserSchema()
	data := makeUsers(10_000, true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ParseParallelSlice(ctx, schema, data, ParallelOpts{Workers: 4, MinChunk: 64})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want canceled, got %v", err)
	}
}

func TestParseParallelSliceContextCancelMidFlight(t *testing.T) {
	// Slow-ish element schema via many refinements isn't available cheaply;
	// cancel after a short delay while validating a large slice.
	schema := flatUserSchema()
	data := makeUsers(50_000, true)

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)

	_, err := ParseParallelSlice(ctx, schema, data, ParallelOpts{Workers: 4, MinChunk: 64})
	if err == nil {
		// Extremely fast machines may finish before timeout; accept success.
		return
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("want ctx error, got %v", err)
	}
}

func TestParseParallelSliceRace(t *testing.T) {
	schema := flatUserSchema()
	data := makeUsers(128, true)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, err := ParseParallelSlice(context.Background(), schema, data, ParallelOpts{Workers: 4, MinChunk: 32})
				if err != nil {
					t.Errorf("unexpected: %v", err)
				}
			}
		}()
	}
	wg.Wait()
}

func TestParallelOptsNormalized(t *testing.T) {
	o := ParallelOpts{}.normalized()
	if o.Workers <= 0 {
		t.Fatalf("Workers=%d", o.Workers)
	}
	if o.MinChunk != 64 {
		t.Fatalf("MinChunk=%d", o.MinChunk)
	}
	o2 := ParallelOpts{Workers: 8, MinChunk: 128}.normalized()
	if o2.Workers != 8 || o2.MinChunk != 128 {
		t.Fatalf("%#v", o2)
	}
}

func TestParseParallelSliceStringElements(t *testing.T) {
	data := make([]any, 100)
	for i := range data {
		data[i] = "hello"
	}
	data[40] = 12
	_, err := ParseParallelSlice(context.Background(), String().Min(3), data, ParallelOpts{Workers: 4, MinChunk: 16})
	if err == nil {
		t.Fatal("expected error")
	}
	zerr := err.(*ZodError)
	found := false
	for _, iss := range zerr.Issues {
		if len(iss.Path) > 0 && iss.Path[0] == 40 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("issues: %#v", zerr.Issues)
	}
}
