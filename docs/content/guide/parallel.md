# Parallel validation

Validate large `[]any` slices with a worker pool — `z.ParseParallelSlice`.

## ParseParallelSlice

```go
out, err := z.ParseParallelSlice(ctx, itemSchema, items, z.ParallelOpts{})
```

- Validates each element with `elemSchema`.
- Returns validated values (same length as input) and a combined `*z.Error` when any element fails.
- Issue paths are prefixed with the **absolute element index**.

```go
items := []any{
    map[string]any{"name": "a", "email": "a@x.co", "age": 20.0},
    map[string]any{"name": "bb", "email": "bad", "age": -1.0},
}
out, err := z.ParseParallelSlice(ctx, flatUser, items, z.ParallelOpts{
    Workers:  4,
    MinChunk: 64,
})
if err != nil {
    zerr := err.(*z.Error)
    // paths like [1, "email"], [1, "age"]
    _ = out // still populated for successful / attempted slots
}
```

## ParallelOpts

```go
type ParallelOpts struct {
    Workers  int // goroutine pool size; default GOMAXPROCS
    MinChunk int // minimum len(data) to enable parallelism; default 64
}
```

| Field | Zero value becomes | Notes |
|---|---|---|
| `Workers` | `runtime.GOMAXPROCS(0)` | `Workers <= 1` forces sequential |
| `MinChunk` | `64` | Below threshold → sequential |

## When it runs sequential

Parallelism is skipped when:

- `len(data) < MinChunk`
- `Workers <= 1`
- `len(data) == 0` (returns empty slice)

:::info Overhead threshold
Goroutine scheduling costs more than validation for small slices. Defaults target batches of dozens+ elements. Benchmarks show clear wins around hundreds–thousands of items — see [Benchmarks](#/guide/benchmarks).
:::

## Context cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

out, err := z.ParseParallelSlice(ctx, schema, items, z.ParallelOpts{})
if errors.Is(err, context.DeadlineExceeded) {
    // workers observe ctx.Err between elements
}
```

On cancel, returns `(nil, ctx.Err)` — not a `*z.Error`.

## Determinism

Issue order is **deterministic**:

1. Chunks in ascending start index.
2. Within a chunk, element index order.
3. Within an element, schema issue order (object fields are pre-sorted).

Workers write into a results array by job id; the join phase concatenates in order. Safe for snapshot tests and stable API error ordering.

## Nil schema / nil data

| Input | Behavior |
|---|---|
| `data == nil` | Treated as empty → `[]any{}` after normalize; or copy-through if schema nil |
| `elemSchema == nil` | Copies `data` to output, no validation |

## Example: batch import

```go
func importUsers(ctx context.Context, rows []any) ([]any, error) {
    return z.ParseParallelSlice(ctx, userSchema, rows, z.ParallelOpts{
        Workers:  runtime.GOMAXPROCS(0),
        MinChunk: 128,
    })
}
```

## Signature

```go
func ParseParallelSlice(
    ctx context.Context,
    elemSchema AnySchemaLike,
    data []any,
    opts ParallelOpts,
) ([]any, error)
```

## ConcurrentBatch / ConcurrentParseAny

Lower-level worker-pool helpers when you want **per-element errors** instead of one combined `*z.Error`. Both always run the pool (no `MinChunk` threshold) and preserve input order in the results.

```go
func ConcurrentBatch[T any](ctx context.Context, schema Schema[T], inputs []any, workers int) ([]T, []error, error)
func ConcurrentParseAny(ctx context.Context, schema AnySchemaLike, inputs []any, workers int) ([]any, []error, error)
```

- `workers <= 0` → `GOMAXPROCS`.
- Returns `outs[i]` / `errs[i]` aligned with `inputs[i]`; elements that failed leave their zero value in `outs`.
- The third return value is the first non-nil element error (or `ctx.Err()` on cancellation) — a cheap “did anything fail” signal without scanning `errs`.

```go
outs, errs, err := z.ConcurrentBatch(context.Background(), userSchema, rows, 4)
if err != nil {
    for i, e := range errs {
        if e != nil {
            log.Printf("row %d: %v", i, e)
        }
    }
}
_ = outs
```

Use `ConcurrentParseAny` for untyped schemas; `ConcurrentBatch[T]` keeps typed `Schema[T]` outputs typed.

## Shared.ParseAll

[`Share`](#/guide/concurrency) wraps a schema with a tiny bit of reusable state; `Shared[T].ParseAll` is the same worker-pool batch as `ConcurrentBatch`, hung off the shared schema:

```go
func (s Shared[T]) ParseAll(ctx context.Context, inputs []any, workers int) (outs []T, errs []error, err error)
```

```go
shared := z.Share(userSchema)
outs, errs, err := shared.ParseAll(ctx, rows, 0) // workers = GOMAXPROCS
```

## Worker panics

A panic inside a worker goroutine (from a `Refine`, `Transform`, or `Check` closure) is captured and **re-raised on the calling goroutine**, so your normal `recover()` keeps working. The recovered value is a `*z.WorkerPanic`:

```go
type WorkerPanic struct {
    Op    string // call the panic happened under, e.g. "ParseParallelSlice"
    Index int    // input element being validated, or -1 if unknown
    Value any    // whatever was passed to panic
    Stack []byte // worker goroutine stack
}
```

```go
defer func() {
    if r := recover(); r != nil {
        if wp, ok := r.(*z.WorkerPanic); ok {
            log.Printf("panic in %s at index %d: %v", wp.Op, wp.Index, wp.Value)
        }
    }
}()
out, err := z.ParseParallelSlice(ctx, schema, items, z.ParallelOpts{Workers: 4})
```

`WorkerPanic.Unwrap()` exposes `Value` when the panicked value was an `error`, so `errors.Is/As` keep working through it.
