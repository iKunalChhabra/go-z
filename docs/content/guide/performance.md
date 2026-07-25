# Performance model

go-zod is built for high-concurrency HTTP workloads: immutable schemas, pooled payloads, precompiled object plans, and a zero-check fast path.

## Payload pools

Every parse acquires a `Payload` from `sync.Pool`:

```go
p := AcquirePayload(data)
in.Run(p, ctx)
// issues copied out on failure, then:
ReleasePayload(p)
```

- Issue slices are reused (capacity reset; huge slices >64 are discarded so the pool doesn’t pin memory).
- Unions, intersections, containers, and parallel workers all use the same pool.
- Happy-path primitive parses allocate very little beyond the output value.

:::info Never retain pooled payloads
After `ReleasePayload`, do not keep references to the payload or its issue slice. Failed parses copy finalized issues into `*ZodError` first.
:::

## Precompiled object plans

At `Object(...)` construction, go-zod compiles a stable field plan:

- Keys sorted once → **deterministic issue order**
- Per-field `*Internals` resolved once
- Key set for unknown-key modes (`Strict` / `Loose` / `Catchall`)

Hot-path parse walks `[]objectField` — **no reflection**, no map iteration over the shape. This is the Go analog of Zod v4’s JIT `Doc` compilation.

```go
// Built once at init — plan is fixed for the life of the schema
var user = z.Object(z.Shape{
    "name":  z.String().Min(2),
    "email": z.String().Email(),
})
```

`ToStruct` adds a second cached plan (reflection decode) keyed by schema internals + target type — see [Struct binding](#/integrations/tostruct).

## Zero-check fast path

Schemas are `Def` + `Parse` + `Run`:

| Checks attached? | `Run` |
|---|---|
| No | `Run == Parse` (direct function pointer) |
| Yes | `Run` = parse, then `runChecks` |

Fluent methods that only add checks (`Min`, `Email`, …) clone the def; schemas with **no** checks skip the check loop entirely — same deferred fast path as Zod.

Discriminated unions add another fast path: O(1) map dispatch instead of trying every option.

## Immutability & concurrency

- Every fluent method returns a **new** schema (cloned def / plan).
- Built schemas are read-only → **lock-free** concurrent `Parse` from any number of goroutines.
- `-race` clean under `b.RunParallel` (see [Benchmarks](#/guide/benchmarks)).

:::tip Share schemas, not builders mid-flight
Assign finished schemas to package-level vars. Don’t mutate shared maps used as `Shape` after `Object` returns — `Object` clones the shape, but holding live schema pointers you later wrap is fine because wraps clone.
:::

## Practical guidance

| Do | Avoid |
|---|---|
| Build schemas once at startup | Rebuild `Object` / `ToStruct` per request |
| Prefer `DiscriminatedUnion` over large `Union` | Linear unions of many object variants |
| Use `ParseParallelSlice` for large arrays | Parallelism for tiny slices (below `MinChunk`) |
| Keep check chains short on hot paths | Heavy `SuperRefine` in tight loops when a format check would do |

## Related

- [Parallel validation](#/guide/parallel)
- [Benchmarks](#/guide/benchmarks)
- [Immutability & concurrency](#/guide/concurrency)
