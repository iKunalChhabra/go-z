# go-zod

A **native Go port of [Zod v4](https://zod.dev)** — same patterns, same issue taxonomy, same fluent API — built for high-concurrency use in [Gin](https://gin-gonic.com) and beyond.

```go
user := zod.Object(zod.Shape{
    "name":  zod.String().Min(2).Max(100),
    "email": zod.String().Email(),
    "age":   zod.Optional(zod.Int().Gte(0).Lt(150)),
    "tags":  zod.Default(zod.Array(zod.String()).Max(10), []any{}),
})

data, err := user.Parse(input) // map[string]any, or *zod.ZodError
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

Thorough docs (Bun-inspired cream + pink theme) live in [`docs/`](./docs) and are meant for **GitHub Pages**:

- Local: `cd docs && python3 -m http.server 5173` → http://127.0.0.1:5173
- After enabling Pages: `https://<user>.github.io/go-zod/`

See [docs/README.md](./docs/README.md) for publish steps.

## Install

```bash
go get github.com/iKunalChhabra/go-zod
```

Requires Go 1.22+.

## Quick start

```go
import "github.com/iKunalChhabra/go-zod"

schema := zod.String().Min(5).Email()

s, err := schema.Parse("hi@example.com")
if err != nil {
    zerr := err.(*zod.ZodError)
    fmt.Println(zod.Prettify(zerr))
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
var Category zod.AnySchemaLike
Category = zod.Lazy(func() zod.AnySchemaLike {
    return zod.Object(zod.Shape{
        "name":     zod.String().Min(1),
        "children": zod.Default(zod.Array(Category), []any{}),
    })
})

userOrGuest := zod.DiscriminatedUnion("role", []zod.AnySchemaLike{
    zod.Object(zod.Shape{"role": zod.Literal("admin"), "perms": zod.Array(zod.String())}),
    zod.Object(zod.Shape{"role": zod.Literal("guest"), "session": zod.String().UUID()}),
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
parsed, err := zod.ToStruct[User](user).Parse(input)
```

### Parallel validation

```go
out, err := zod.ParseParallelSlice(ctx, itemSchema, items, zod.ParallelOpts{})
```

On 10k-element arrays this is ~2.5× faster than sequential on a 4-core machine — see [BENCHMARKS.md](./BENCHMARKS.md).

## Performance (headline)

4-core Xeon, Go 1.22, median of 3 runs — full tables in [BENCHMARKS.md](./BENCHMARKS.md):

| Scenario | go-zod | go-playground/validator | Oudwins/zog |
|---|---:|---:|---:|
| FlatUser | **739 ns** | 610 ns | 1314 ns |
| Nested object | **1281 ns** | 1090 ns | 2829 ns |
| Array 10k (parallel) | **2.94 ms** | 6.17 ms | 13.1 ms |
| String formats | ~within 6% of validator | — | ~1.6× slower |

Schemas are immutable and lock-free — `b.RunParallel` / `-race` clean under concurrent load.

## Design notes

- **Untyped core, typed edge.** Like Zod, the engine runs on `any`; `Schema[T]` is the generic boundary. Output-type-changing ops (`Transform`, `Pipe`, `ToStruct`) are package-level functions because Go methods cannot introduce new type parameters.
- **JSON model first.** Objects produce `map[string]any`, arrays `[]any`. Use `ToStruct[T]` when you want typed Go structs.
- **`Missing` ≠ `nil`.** `Missing` is Zod's `undefined` (absent key); `nil` is JSON `null`. `Optional` accepts Missing; `Nullable` accepts nil.
- **Coercion.** `zod.Coerce.String()/Number()/Bool()/Time()` for query/form pipelines.

## Package map

```
zod.go              package docs
schema_*.go         schemas (string, number, object, union, ...)
checks_*.go         composable checks
errorutils.go       Flatten / Format / Treeify / Prettify
registry.go         metadata registry
locale_*.go         i18n error maps
parallel.go         ParseParallelSlice
tostruct.go         cached reflect decode
zgin/               Gin bind + middleware
bench/              comparative benchmarks
PLAN.md             architecture + sub-agent work distribution
```

## Status

Ports Zod v4 (`colinhacks/zod` @ `912f0f5`). Covered: primitives, formats, objects/collections, unions/discriminated unions/intersection/lazy, wrappers (optional/nullable/default/catch/pipe/transform/refine), coerce, error utils, locales (7), Gin, ToStruct, parallel parse, benchmarks.

Roadmap (not in v0): codecs/`encode`/`decode`, JSON Schema conversion, `z.function`, template literals, remaining ~47 locales.

## License

MIT — see [LICENSE](./LICENSE).
