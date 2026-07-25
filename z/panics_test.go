package z

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// A panicking user closure inside a worker goroutine used to kill the process:
// the caller's recover cannot see a panic raised on another goroutine. It is now
// captured and re-raised on the calling goroutine, so it behaves like the
// sequential path.
func TestParallelPanicSurfacesOnCaller(t *testing.T) {
	boom := Refine(String(), func(v any) bool {
		if v == "explode" {
			panic("boom from a check")
		}
		return true
	})
	data := make([]any, 200)
	for i := range data {
		data[i] = "ok"
	}
	data[137] = "explode"

	wp := recoverWorkerPanic(t, func() {
		_, _ = ParseParallelSlice(context.Background(), boom, data, ParallelOpts{Workers: 4, MinChunk: 1})
	})
	if wp.Op != "ParseParallelSlice" {
		t.Errorf("Op = %q", wp.Op)
	}
	if wp.Index != 137 {
		t.Errorf("Index = %d, want 137", wp.Index)
	}
	if got, ok := wp.Value.(string); !ok || got != "boom from a check" {
		t.Errorf("Value = %#v", wp.Value)
	}
	if len(wp.Stack) == 0 {
		t.Error("the worker stack should be preserved")
	}
	if !strings.Contains(wp.Error(), "element 137") {
		t.Errorf("Error() = %q", wp.Error())
	}
}

func TestConcurrentBatchPanicSurfacesOnCaller(t *testing.T) {
	schema := Refine(String(), func(v any) bool {
		if v == "explode" {
			panic(errors.New("boom"))
		}
		return true
	})
	inputs := []any{"a", "b", "explode", "d"}

	wp := recoverWorkerPanic(t, func() {
		_, _, _ = ConcurrentParseAny(context.Background(), schema, inputs, 4)
	})
	if wp.Op != "ConcurrentParseAny" || wp.Index != 2 {
		t.Errorf("got Op=%q Index=%d", wp.Op, wp.Index)
	}
	// The panic value was an error, so it unwraps.
	if err := errors.Unwrap(wp); err == nil || err.Error() != "boom" {
		t.Errorf("Unwrap = %v", err)
	}
}

func TestTypedConcurrentBatchPanicSurfacesOnCaller(t *testing.T) {
	schema := String().Refine(func(s string) bool {
		if s == "explode" {
			panic("typed boom")
		}
		return true
	})
	wp := recoverWorkerPanic(t, func() {
		_, _, _ = ConcurrentBatch[string](context.Background(), schema, []any{"a", "explode"}, 2)
	})
	if wp.Op != "ConcurrentBatch" {
		t.Errorf("Op = %q", wp.Op)
	}
}

// A panicking element must not leave a payload checked out of the pool, so the
// next parse after a recovered panic still behaves.
func TestPoolSurvivesWorkerPanic(t *testing.T) {
	boom := Refine(String(), func(v any) bool {
		if v == "explode" {
			panic("boom")
		}
		return true
	})
	for range 5 {
		_ = recoverWorkerPanic(t, func() {
			_, _ = ParseParallelSlice(context.Background(), boom,
				[]any{"ok", "explode", "ok"}, ParallelOpts{Workers: 2, MinChunk: 1})
		})
	}
	out, err := ParseParallelSlice(context.Background(), boom,
		[]any{"ok", "fine"}, ParallelOpts{Workers: 2, MinChunk: 1})
	if err != nil || len(out) != 2 {
		t.Fatalf("parsing after a panic failed: %#v %v", out, err)
	}
}

func recoverWorkerPanic(t *testing.T, fn func()) (wp *WorkerPanic) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected the worker panic to be re-raised on this goroutine")
		}
		got, ok := r.(*WorkerPanic)
		if !ok {
			t.Fatalf("recovered %T, want *WorkerPanic: %v", r, r)
		}
		wp = got
	}()
	fn()
	return nil
}
