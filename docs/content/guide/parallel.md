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
ctx, cancel := context.WithTimeout(context.Background, 2*time.Second)
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
