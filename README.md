# go-zod

A **native Go port of [Zod v4](https://zod.dev)** — same patterns, same issue taxonomy, same fluent API — built for high-concurrency use in [Gin](https://gin-gonic.com) and beyond.

```go
user := z.Object(z.Shape{
    "name":  z.String().Min(2).Max(100),
    "email": z.String().Email(),
    "age":   z.Optional(z.Int().Gte(0).Lt(150)),
    "tags":  z.Default(z.Array(z.String()).Max(10), []any{}),
})

data, err := user.Parse(input) // map[string]any, or *z.ZodError
```

## Why

Existing Zod-inspired Go libraries are thinly maintained. go-zod ports Zod v4's actual architecture — not just the surface API:

| Zod v4 pattern | go-zod |
|---|---|
| `ParsePayload {value, issues[]}` accumulate-don't-throw | `Payload` + pooled issue slices |
| Schema = `def` + `parse` + `run` (zero-check fast path) | identical wiring |
| Checks with `when` / `abort` / `continue` / `onattach` | `Check` |
| 11 issue codes + error-map chain | byte-compatible JSON |
| Classic fluent API | `String().Min(5).Email()` |
| Locales | `en es fr de ja pt zh` |

## Documentation

Thorough docs (Cream Soda theme — Bun-inspired cream + pink) live in [`docs/`](./docs) and are meant for **GitHub Pages**:

- Local: `cd docs && python3 -m http.server 5173` → http://127.0.0.1:5173
- After enabling Pages: `https://<user>.github.io/go-zod/`

See [docs/README.md](./docs/README.md) for publish steps.

## Install

```bash
go get github.com/iKunalChhabra/go-zod
```

Requires Go 1.22+.

Docs and examples use the import alias `z` (Zod convention):

```go
import z "github.com/iKunalChhabra/go-zod"
```

## Quick start

```go
import z "github.com/iKunalChhabra/go-zod"

schema := z.String().Min(5).Email()

s, err := schema.Parse("hi@example.com")
if err != nil {
    zerr, _ := z.AsZodError(err) // unwraps wrapped errors too
    fmt.Println(z.Prettify(zerr))
    // ✖ Too small: expected string to have >=5 characters
    //   → at
}

res := schema.SafeParse("nope")
if !res.Success {
    fmt.Println(res.Error.Issues[0].Code) // invalid_format
}
```

### Objects, unions, recursion

```go
var Category z.AnySchemaLike
Category = z.Lazy(func() z.AnySchemaLike {
    return z.Object(z.Shape{
        "name":     z.String().Min(1),
        "children": z.Default(z.Array(Category), []any{}),
    })
})

userOrGuest := z.DiscriminatedUnion("role", []z.AnySchemaLike{
    z.Object(z.Shape{"role": z.Literal("admin"), "perms": z.Array(z.String())}),
    z.Object(z.Shape{"role": z.Literal("guest"), "session": z.String().UUID()}),
})
```

### Gin

```go
import "github.com/iKunalChhabra/go-zod/zgin"

r.POST("/users", zgin.Validate(user), func(c *gin.Context) {
    body, _ := zgin.Get(c) // already parsed & validated
    c.JSON(200, body)
})

// or
r.POST("/users", func(c *gin.Context) {
    body, ok := zgin.BindJSONAny(c, user)
    if !ok { return } // 400 + Zod issues already written
    ...
})
```

Error body (default):

```json
{"success":false,"error":{"issues":[{"code":"too_small","path":["name"],"message":"Too small: expected string to have >=2 characters", ...}]}}
```

Also supports Flatten / Treeify / Prettify renderers via `zgin.Options`.

### Struct binding

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}
parsed, err := z.ToStruct[User](user).Parse(input)
```

### Concurrency

Built schemas are **immutable and safe to share** across goroutines — no locks needed around `Parse` / `SafeParse`:

```go
schema := zod.String().Email() // build once
go func() { schema.Parse(a) }()
go func() { schema.Parse(b) }()
```

Batch helpers when you want an explicit fan-out API:

```go
shared := zod.Share(schema)
outs, errs, err := shared.ParseAll(ctx, inputs, 8)
// or: zod.ConcurrentParseAny(ctx, objectSchema, inputs, 8)
```

### Parallel validation

```go
out, err := z.ParseParallelSlice(ctx, itemSchema, items, z.ParallelOpts{})
```

On 10k-element arrays this is ~2.5× faster than sequential on a 4-core machine — see [BENCHMARKS.md](./BENCHMARKS.md).

Zod classic test ports live in `parity_*_test.go` — see [PARITY.md](./PARITY.md).

## Performance (headline)

4-core Xeon, Go 1.22, median of 3 runs — full tables in [BENCHMARKS.md](./BENCHMARKS.md):

| Scenario | go-zod | go-playground/validator | Oudwins/zog |
|---|---:|---:|---:|
| FlatUser | **416 ns** | 607 ns | 1295 ns |
| Nested object | **978 ns** | 1090 ns | 2783 ns |
| Array 10k (parallel) | **2.75 ms** | 6.09 ms | 12.9 ms |
| String formats | **890 ns** | 1099 ns | 1810 ns |

Schemas are immutable and lock-free — `b.RunParallel` / `-race` clean under concurrent load.

## Design notes

- **Untyped core, typed edge.** Like Zod, the engine runs on `any`; `Schema[T]` is the generic boundary. Same-type fluent ops (`Optional`, `Refine`, …) are methods on concrete schemas; output-type-changing ops (`Transform`, `Pipe`, `ToStruct`) stay package-level because Go methods cannot introduce new type parameters.
- **JSON model first.** Objects produce `map[string]any`, arrays `[]any`, and `Int()`/`Number()` output `float64`. Use `Int64()` for a typed `int64` edge, or `ToStruct[T]` for structs.
- **`Missing` ≠ `nil`.** `Missing` is Zod's `undefined` (absent key); `nil` is JSON `null`. `Optional` accepts Missing; `Nullable` accepts nil.
- **Object field order.** `Object(Shape)` reports issues in sorted key order (Go maps are unordered). Use `ObjectOrdered([]Field{...})` for definition order.
- **Coercion.** `z.Coerce.String()/Number()/Bool()/Time()` for query/form pipelines.
- **Parse context.** `ParseCtx.Context` threads `context.Context` into `SuperRefine` via `RefinementCtx.Context()`.

## Package map

```
zod.go              package docs
schema_*.go         schemas (string, number, object, union, ...)
checks_*.go         composable checks
fluent*.go          mid-chain Optional/Default/Refine on concrete schemas
errorutils.go       Flatten / Format / Treeify / Prettify
registry.go         metadata registry
locale_*.go         i18n error maps
parallel.go         ParseParallelSlice
tostruct.go         cached reflect decode
zgin/               Gin bind + middleware (GetAs[T], ValidateToStruct[T])
bench/              comparative benchmarks
```

## Status

Ports Zod v4 (`colinhacks/zod` @ `912f0f5`). Covered: primitives, formats (incl. hostname/hash/httpURL), objects/collections, unions/xor/discriminated unions/intersection/lazy, wrappers (optional/nullable/default/catch/pipe/transform/refine/preprocess), codecs (`Decode`/`Encode`/`InvertCodec`), `ToJSONSchema`, `TemplateLiteral`, coerce (incl. BigInt), specials (`Undefined`/`Void`/`JSON`/`StringBool`), error utils, locales (7), Gin, ToStruct, parallel parse, benchmarks.

Roadmap (not in v0): `fromJSONSchema`, `z.function`/`z.promise`/`z.file`/`z.symbol`, remaining ~47 locales.

## License

MIT — see [LICENSE](./LICENSE).
