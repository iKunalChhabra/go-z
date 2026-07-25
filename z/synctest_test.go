package z

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

// synctest runs these in a bubble with a fake clock, so cancellation and
// deadline behaviour is exercised deterministically instead of with sleeps that
// are either flaky or slow.

func TestParallelSliceHonoursCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		schema := Object(Shape{"name": String().Min(2)})
		data := make([]any, 4096)
		for i := range data {
			data[i] = map[string]any{"name": "ada"}
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // already cancelled before any work starts

		_, err := ParseParallelSlice(ctx, schema, data, ParallelOpts{Workers: 4, MinChunk: 64})
		if err == nil {
			t.Fatal("want a cancellation error")
		}
		// Every worker must observe the cancellation and return; if any leaked,
		// the bubble would report goroutines still running at exit.
		synctest.Wait()
	})
}

func TestParallelSliceHonoursDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		schema := Object(Shape{"name": String().Min(2)})
		data := make([]any, 2048)
		for i := range data {
			data[i] = map[string]any{"name": "ada"}
		}

		// The fake clock makes this deadline fire instantly once every
		// goroutine in the bubble is blocked, with no real waiting.
		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		defer cancel()

		out, err := ParseParallelSlice(ctx, schema, data, ParallelOpts{Workers: 4, MinChunk: 64})
		if err != nil {
			t.Fatalf("work that fits inside the deadline should succeed: %v", err)
		}
		if len(out) != len(data) {
			t.Fatalf("len = %d, want %d", len(out), len(data))
		}
		synctest.Wait()
	})
}

func TestConcurrentBatchLeavesNoGoroutines(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		schema := String().Min(2)
		inputs := make([]any, 512)
		for i := range inputs {
			inputs[i] = "value"
		}

		outs, errs, err := ConcurrentBatch(context.Background(), schema, inputs, 4)
		if err != nil {
			t.Fatal(err)
		}
		for i, e := range errs {
			if e != nil {
				t.Fatalf("input %d: %v", i, e)
			}
			if outs[i] != "value" {
				t.Fatalf("input %d = %v", i, outs[i])
			}
		}
		// Wait returns only when every goroutine in the bubble has finished or
		// blocked; the test fails if a worker outlives the call.
		synctest.Wait()
	})
}
