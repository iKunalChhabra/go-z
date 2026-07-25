# Immutability & concurrency

go-z is built for multi-goroutine HTTP servers. Schemas are immutable after construction; `Parse` does not take locks. This page explains the model, what is safe to share, and where you still need care.

## The contract

1. **Schemas are immutable** — fluent methods return clones; they never mutate the receiver.
2. **Parse is lock-free** — concurrent `Parse` / `SafeParse` / `ParseCtx` on the same schema is fine.
3. **Payloads are pooled** — each parse acquires a short-lived `Payload`; issues are copied into `*z.Error` before release.
4. **Outputs are yours** — maps and slices returned from parse are not deep-frozen; don’t share them across goroutines without synchronization if you mutate them.

:::tip Intended usage
Build schemas once (package `var` or `init`), share the pointer everywhere, call `Parse` from any number of handlers concurrently. That is the happy path.
:::

## Immutable fluent API

```go
var BaseName = z.String.Min(1)

func init {
	// These do not mutate BaseName
	_ = BaseName.Max(50)
	_ = BaseName.Email // weird but safe — new schema value
}
```

Because clones happen at construction time, extending a shared schema in one package cannot race with parses in another:

```go
package schemas

var Password = z.String.Min(8)

package admin

// Local stricter variant — Password unchanged
var AdminPassword = schemas.Password.Min(16)
```

## Concurrent Parse

```go
var User = z.Object(z.Shape{
	"email": z.String.Email,
	"name":  z.String.Min(1),
})

func handler(w http.ResponseWriter, r *http.Request) {
	var input any
	_ = json.NewDecoder(r.Body).Decode(&input)

	out, err:= User.Parse(input) // many goroutines may do this at once
	_ = out
	_ = err
}
```

Under the hood each call:

1. `AcquirePayload` from a `sync.Pool`
2. Runs `Internals.Run` (no schema mutation)
3. On failure, copies finalized issues into a new `*z.Error`
4. `ReleasePayload` — callers must not retain the pooled payload

Schemas themselves have no mutex. `-race` stays clean for concurrent parse of shared schemas (see repo benchmarks / `b.RunParallel`).

## What about Configure?

`z.Configure` uses an atomic pointer swap — it is safe to call concurrently with parses. Still treat it as **process configuration**, not a per-request knob:

```go
// Once at startup
z.Configure(z.Config{LocaleError: z.Locale("es")})
```

For per-request locale or messages, use `ParseCtx.Error` instead of racing global config in handlers.

## Parallel array validation

Large homogeneous slices can opt into a worker pool:

```go
ctx:= context.Background
item:= z.Object(z.Shape{
	"id": z.String.UUID,
})

out, err:= z.ParseParallelSlice(ctx, item, rows, z.ParallelOpts{
	Workers:  runtime.GOMAXPROCS(0),
	MinChunk: 64, // below this, runs sequential
})
```

Notes:

- Issue paths are prefixed with the absolute element index.
- Issue order is deterministic (chunk order, then index).
- Context cancellation returns `ctx.Err`.
- The **element schema** is still shared immutably across workers — same rules as above.

On multi-core machines, 10k-element arrays are roughly ~2.5× faster than sequential in published benchmarks.

## Pooling

You do not manage the payload pool yourself. If you write custom schema internals or call `AcquirePayload` / `ReleasePayload` directly:

```go
p:= z.AcquirePayload(value)
defer z.ReleasePayload(p)

schema.Internals.Run(p, nil)
// Copy anything you need from p before release
issues:= append([]z.Issue(nil), p.Issues...)
```

:::warn Do not retain pooled payloads
After `ReleasePayload`, the issue slice may be reused. Always copy issues out (as `Parse` does) before release.
:::

Huge error slices are discarded rather than returned to the pool (capacity guard) so one pathological request doesn’t pin memory forever.

## Outputs and shared mutation

Parsed objects are ordinary Go values:

```go
out, err:= User.Parse(input)
if err != nil {
	return err
}
m:= out // map[string]any

// Fine — exclusive to this goroutine
m["extra"] = true

// Dangerous — sharing m with other goroutines while mutating
go func { m["x"] = 1 }
go func { fmt.Println(m["x"]) }
```

Schemas don’t deep-freeze results. If you need to share parsed data across goroutines:

- treat outputs as immutable by convention, or
- deep-copy / synchronize, or
- convert once with `ToStruct[T]` and share the struct carefully

`Default` / `Prefault` / `Catch` clone common JSON-model defaults (`[]any`,
`map[string]any`, `[]string`, `*big.Int`) on each use so handlers cannot
corrupt later parses by mutating the returned value. For custom pointer types,
use `DefaultFunc` / `CatchFunc` and allocate fresh:

```go
schema:= z.DefaultFunc(z.Array(z.String), func any {
	return []any{}
})
```

## Building schemas at runtime

Creating schemas inside a request is allowed but usually wasteful:

```go
// Avoid in hot paths
func bad(min int) (string, error) {
	return z.String.Min(min).Parse(input)
}
```

Prefer:

```go
var min8 = z.String.Min(8)

func good (string, error) {
	return min8.Parse(input)
}
```

If you must specialize per tenant, cache the derived schema (e.g. `sync.Map`) keyed by options — each entry remains immutable after creation.

## Race detector

Validate your service under the race detector when you wire custom checks that close over shared state:

```bash
go test -race./...
```

Safe check:

```go
ch:= &z.Check{
	Fn: func(p *z.Payload) {
		// only reads p.Value — fine
	},
}
```

Unsafe check:

```go
var counter int
ch:= &z.Check{
	Fn: func(p *z.Payload) {
		counter++ // data race under parallel Parse
	},
}
```

Use atomics or don’t mutate shared state from `Check.Fn`.

## Checklist

| Do | Don’t |
|---|---|
| Share package-level schemas across goroutines | Mutate schema defs after construction (you can’t via public API — don’t bypass with unsafe) |
| Call `Parse` freely from handlers | Call `Configure` per request |
| Use `DefaultFunc` for fresh slices/maps | Share one mutable default slice across parses |
| Use `ParseParallelSlice` for huge arrays | Assume returned `map[string]any` is frozen |
| Keep `Check.Fn` free of shared mutable state | Close over request-scoped races inside checks |

## Minimal stress sketch

```go
func TestParallelParse(t *testing.T) {
	schema:= z.String.Min(1).Email
	var wg sync.WaitGroup
	for i:= 0; i < 100; i++ {
		wg.Add(1)
		go func {
			defer wg.Done
			_, _ = schema.Parse("ada@example.com")
			_, _ = schema.Parse("nope")
		}
	}
	wg.Wait
}
```

Run with `-race` in CI. This is the concurrency model go-z optimizes for: many readers, zero schema writers.

## Related

- [Schemas & parsing](#/guide/parsing) — fluent clones
- [Checks & refinements](#/guide/checks) — keep `Fn` pure
- [Parallel validation](#/guide/parallel) — deeper `ParseParallelSlice` docs
- [Why go-z?](#/guide/why) — architecture that enables lock-free parse
