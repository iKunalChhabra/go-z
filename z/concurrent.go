package z

import (
	"context"
	"runtime"
	"sync"
)

func defaultWorkers() int {
	n := runtime.GOMAXPROCS(0)
	if n < 1 {
		return 1
	}
	return n
}

// Concurrent safety model
//
// Built schemas are immutable: every fluent method (Min, Email, …) clones the
// Def/checks and returns a new schema value. After construction, a schema may
// be shared across any number of goroutines and Parse/SafeParse/MustParse may
// be called concurrently without locks.
//
// What IS safe without extra wrapping:
//   - Sharing one schema across goroutines for Parse / SafeParse / MustParse
//   - Configure / GetConfig (atomic pointer)
//   - Registry methods (internal mutex)
//   - Lazy memoization (sync.Once)
//   - ParseParallelSlice (own worker pool)
//   - Payload pooling (sync.Pool; each Parse acquires/releases its own payload)
//
// What is NOT schema-shared state (caller's responsibility):
//   - Mutating the map/slice returned by Parse while another goroutine reads it
//   - Mutating a Shape map after passing it to Object (Object clones the map,
//     but nested schema pointers are shared by design — treat schemas as
//     immutable after build)
//   - Check closures that close over mutable variables without synchronization
//
// Shared and ConcurrentBatch below are convenience helpers for fan-out patterns;
// they are not required for ordinary concurrent Parse of a shared schema.

// Shared wraps any schema as an explicitly concurrency-safe handle. It does
// not add locks around Parse — schemas are already immutable — but documents
// intent and provides batch helpers.
type Shared[T any] struct {
	schema Schema[T]
}

// Share returns a Shared handle over schema.
func Share[T any](schema Schema[T]) Shared[T] {
	return Shared[T]{schema: schema}
}

// Schema returns the underlying schema.
func (s Shared[T]) Schema() Schema[T] { return s.schema }

// Parse is a thin alias of schema.Parse (safe for concurrent use).
func (s Shared[T]) Parse(data any) (T, error) { return s.schema.Parse(data) }

// SafeParse is a thin alias of schema.SafeParse (safe for concurrent use).
func (s Shared[T]) SafeParse(data any) SafeParseResult[T] { return s.schema.SafeParse(data) }

// MustParse is a thin alias of schema.MustParse.
func (s Shared[T]) MustParse(data any) T { return s.schema.MustParse(data) }

// ParseAll validates each input with the shared schema from multiple
// goroutines (one per element, bounded by a worker pool of size workers).
// workers <= 0 means GOMAXPROCS. Results and errors are returned in input
// order; err is the first non-nil error (others are still collected in errs).
func (s Shared[T]) ParseAll(ctx context.Context, inputs []any, workers int) (outs []T, errs []error, err error) {
	return ConcurrentBatch(ctx, s.schema, inputs, workers)
}

// ConcurrentBatch validates inputs with schema using a worker pool. Order of
// outs/errs matches inputs. Safe with a single shared schema — no per-call
// cloning is required.
func ConcurrentBatch[T any](ctx context.Context, schema Schema[T], inputs []any, workers int) ([]T, []error, error) {
	const op = "ConcurrentBatch"
	n := len(inputs)
	outs := make([]T, n)
	errs := make([]error, n)
	if n == 0 {
		return outs, errs, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if workers <= 0 {
		workers = defaultWorkers()
	}
	if workers > n {
		workers = n
	}

	pctx := &ParseCtx{Context: ctx} // never mutated during parse; share it
	jobs := make(chan int, n)
	var wg sync.WaitGroup
	var panics panicRecord
	for range workers {
		wg.Go(func() {
			for i := range jobs {
				if panics.stop() {
					return
				}
				if ctx.Err() != nil {
					errs[i] = ctx.Err()
					continue
				}
				func() {
					defer panics.capture(i)
					outs[i], errs[i] = schema.ParseCtx(inputs[i], pctx)
				}()
			}
		})
	}
	for i := 0; i < n; i++ {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			panics.rethrow(op)
			return outs, errs, ctx.Err()
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	panics.rethrow(op)
	if err := ctx.Err(); err != nil {
		return outs, errs, err
	}
	var first error
	for _, e := range errs {
		if e != nil {
			first = e
			break
		}
	}
	return outs, errs, first
}

// ConcurrentParseAny is like ConcurrentBatch for type-erased AnySchemaLike
// schemas (Object, Union, …). Outputs are []any.
func ConcurrentParseAny(ctx context.Context, schema AnySchemaLike, inputs []any, workers int) ([]any, []error, error) {
	const op = "ConcurrentParseAny"
	n := len(inputs)
	outs := make([]any, n)
	errs := make([]error, n)
	if n == 0 {
		return outs, errs, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if schema == nil {
		copy(outs, inputs)
		return outs, errs, nil
	}
	if workers <= 0 {
		workers = defaultWorkers()
	}
	if workers > n {
		workers = n
	}
	in := schema.Internals()
	pctx := &ParseCtx{Context: ctx} // never mutated during parse; share it
	jobs := make(chan int, n)
	var wg sync.WaitGroup
	var panics panicRecord
	for range workers {
		wg.Go(func() {
			for i := range jobs {
				if panics.stop() {
					return
				}
				if ctx.Err() != nil {
					errs[i] = ctx.Err()
					continue
				}
				func() {
					defer panics.capture(i)
					outs[i], errs[i] = parseTyped[any](in, inputs[i], pctx)
				}()
			}
		})
	}
	for i := 0; i < n; i++ {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			panics.rethrow(op)
			return outs, errs, ctx.Err()
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	panics.rethrow(op)
	if err := ctx.Err(); err != nil {
		return outs, errs, err
	}
	var first error
	for _, e := range errs {
		if e != nil {
			first = e
			break
		}
	}
	return outs, errs, first
}
