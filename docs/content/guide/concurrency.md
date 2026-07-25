# Immutability & concurrency

go-zod is built for multi-goroutine HTTP servers. Schemas are immutable after construction; `Parse` does not take locks. This page explains the model, what is safe to share, and where you still need care.

## The contract

1. **Schemas are immutable** — fluent methods return clones; they never mutate the receiver.  
2. **Parse is lock-free** — concurrent `Parse` / `SafeParse` / `ParseCtx` on the same schema is fine.  
3. **Payloads are pooled** — each parse acquires a short-lived `Payload`; issues are copied into `*ZodError` before release.  
4. **Outputs are yours** — maps and slices returned from parse are not deep-frozen; don’t share them across goroutines without synchronization if you mutate them.

:::tip Intended usage
Build schemas once (package `var` or `init`), share the pointer everywhere, call `Parse` from any number of handlers concurrently. That is the happy path.
:::

## Immutable fluent API

```go
var BaseName = zod.String().Min(1)

func init() {
	// These do not mutate BaseName
	_ = BaseName.Max(50)
	_ = BaseName.Email() // weird but safe — new schema value
}
```

Because clones happen at construction time, extending a shared schema in one package cannot race with parses in another:

```go
package schemas

var Password = zod.String().Min(8)

package admin

// Local stricter variant — Password unchanged
var AdminPassword = schemas.Password.Min(16)
```

## Concurrent Parse

```go
var User = zod.Object(zod.Shape{
	"email": zod.String().Email(),
	"name":  zod.String().Min(1),
})

func handler(w http.ResponseWriter, r *http.Request) {
	var input any
	_ = json.NewDecoder(r.Body).Decode(&input)

	out, err := User.Parse(input) // many goroutines may do this at once
	_ = out
	_ = err
}
```

Under the hood each call:

1. `AcquirePayload` from a `sync.Pool`  
2. Runs `Internals.Run` (no schema mutation)  
3. On failure, copies finalized issues into a new `*ZodError`  
4. `ReleasePayload` — callers must not retain the pooled payload  

Schemas themselves have no mutex. `-race` stays clean for concurrent parse of shared schemas (see repo benchmarks / `b.RunParallel`).

## What about Configure?

`zod.Configure` uses an atomic pointer swap — it is safe to call concurrently with parses. Still treat it as **process configuration**, not a per-request knob:

```go
// Once at startup
zod.Configure(zod.Config{LocaleError: zod.Locale("es")})
```

For per-request locale or messages, use `ParseCtx.Error` instead of racing global config in handlers.

## Parallel array validation

Large homogeneous slices can opt into a worker pool:

```go
ctx := context.Background()
item := zod.Object(zod.Shape{
	"id": zod.String().UUID(),
})

out, err := zod.ParseParallelSlice(ctx, item, rows, zod.ParallelOpts{
	Workers:  runtime.GOMAXPROCS(0),
	MinChunk: 64, // below this, runs sequential
})
```

Notes:

- Issue paths are prefixed with the absolute element index.  
- Issue order is deterministic (chunk order, then index).  
- Context cancellation returns `ctx.Err()`.  
- The **element schema** is still shared immutably across workers — same rules as above.  

On multi-core machines, 10k-element arrays are roughly ~2.5× faster than sequential in published benchmarks.

## Pooling

You do not manage the payload pool yourself. If you write custom schema internals or call `AcquirePayload` / `ReleasePayload` directly:

```go
p := zod.AcquirePayload(value)
defer zod.ReleasePayload(p)

schema.Internals().Run(p, nil)
// Copy anything you need from p before release
issues := append([]zod.Issue(nil), p.Issues...)
```

:::warn Do not retain pooled payloads
After `ReleasePayload`, the issue slice may be reused. Always copy issues out (as `Parse` does via `newZodError`) before release.
:::

Huge error slices are discarded rather than returned to the pool (capacity guard) so one pathological request doesn’t pin memory forever.

## Outputs and shared mutation

Parsed objects are ordinary Go values:

```go
out, err := User.Parse(input)
if err != nil {
	return err
}
m := out // map[string]any

// Fine — exclusive to this goroutine
m["extra"] = true

// Dangerous — sharing m with other goroutines while mutating
go func() { m["x"] = 1 }()
go func() { fmt.Println(m["x"]) }()
```

Schemas don’t deep-freeze results. If you need to share parsed data across goroutines:

- treat outputs as immutable by convention, or  
- deep-copy / synchronize, or  
- convert once with `ToStruct[T]` and share the struct carefully  

The same applies to `Default` values you pass in:

```go
shared := []any{"a"} // mutable backing array
schema := zod.Default(zod.Array(zod.String()), shared)

// If a parse returns this default slice and a handler mutates it,
// the next Missing-input parse may observe the mutation.
```

Prefer immutable defaults (or `DefaultFunc` that allocates fresh):

```go
schema := zod.DefaultFunc(zod.Array(zod.String()), func() any {
	return []any{}
})
```

## Building schemas at runtime

Creating schemas inside a request is allowed but usually wasteful:

```go
// Avoid in hot paths
func bad(min int) (string, error) {
	return zod.String().Min(min).Parse(input)
}
```

Prefer:

```go
var min8 = zod.String().Min(8)

func good() (string, error) {
	return min8.Parse(input)
}
```

If you must specialize per tenant, cache the derived schema (e.g. `sync.Map`) keyed by options — each entry remains immutable after creation.

## Race detector

Validate your service under the race detector when you wire custom checks that close over shared state:

```bash
go test -race ./...
```

Safe check:

```go
ch := &zod.Check{
	Fn: func(p *zod.Payload) {
		// only reads p.Value — fine
	},
}
```

Unsafe check:

```go
var counter int
ch := &zod.Check{
	Fn: func(p *zod.Payload) {
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
	schema := zod.String().Min(1).Email()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = schema.Parse("ada@example.com")
			_, _ = schema.Parse("nope")
		}()
	}
	wg.Wait()
}
```

Run with `-race` in CI. This is the concurrency model go-zod optimizes for: many readers, zero schema writers.

## Related

- [Schemas & parsing](#/guide/parsing) — fluent clones  
- [Checks & refinements](#/guide/checks) — keep `Fn` pure  
- [Parallel validation](#/guide/parallel) — deeper `ParseParallelSlice` docs  
- [Why go-zod?](#/guide/why) — architecture that enables lock-free parse  
