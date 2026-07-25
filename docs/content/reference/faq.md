# FAQ

## 1. Why isn’t there `.Optional()` on schemas?

Go methods cannot introduce new type parameters. Wrappers that change optionality or output type are package functions: `z.Optional(z.String())`, not `z.String().Optional()`.

## 2. What’s the difference between `Missing` and `nil`?

`Missing` is Zod’s `undefined` (absent key). `nil` is JSON `null`. `Optional` accepts Missing; `Nullable` accepts nil; `Nullish` accepts both. See [Missing vs nil](#/guide/missing-nil).

## 3. Why do objects return `map[string]any`?

go-zod follows Zod’s untyped JSON core. Use `z.ToStruct[T](objectSchema)` when you want a Go struct. There is no compile-time `z.infer`.

## 4. Are schemas safe for concurrent use?

Yes. Fluent methods clone; finished schemas are immutable and lock-free. Share them across goroutines / Gin handlers freely.

## 5. Should I rebuild schemas per request?

No. Build once at package init (or startup) and reuse. Object plans and `ToStruct` plans are compiled/cached at construction.

## 6. When should I use `DiscriminatedUnion` vs `Union`?

Use `DiscriminatedUnion` for object variants that share a literal tag — O(1) dispatch and clearer errors. Use `Union` for primitives or shapes without a discriminator.

## 7. Why did my `Default` accept an invalid fallback?

`Default` does **not** re-parse the fallback (Zod behavior). Use `Prefault` if the default must pass the inner schema.

## 8. How do I validate Gin query / path params that are strings?

Use `z.Coerce.Number()` / `Bool()` / `Time()` with `zgin.BindQuery` / `BindURI`. Query values are unwrapped by `CoerceQueryValues`.

## 9. How do I customize HTTP error JSON?

```go
zgin.AbortWithError(c, zerr, zgin.Options{
    Format: zgin.FormatFlatten, // or FormatTree, FormatPretty
    Status: http.StatusUnprocessableEntity,
})
```

Default is `FormatIssues`. Details: [HTTP error shapes](#/integrations/http-errors).

## 10. Can I parse large arrays faster?

Yes — `z.ParseParallelSlice(ctx, elemSchema, data, z.ParallelOpts{})`. Defaults: `Workers=GOMAXPROCS`, `MinChunk=64`. Measured ~**2.5×** at 10k elements on 4 cores. See [Parallel](#/guide/parallel) and [Benchmarks](#/guide/benchmarks).

## 11. Where is `.encode()` / `.decode()` / codecs?

Not in v0. Use one-way `Pipe`, `Transform`, `Preprocess`, or `ToStruct`. Codecs are on the roadmap.

## 12. How do I do recursive types?

```go
var Node z.AnySchemaLike
Node = z.Lazy(func() z.AnySchemaLike {
    return z.Object(z.Shape{
        "children": z.Array(Node),
    })
})
```

Declare the `var` first, then assign `Lazy` that closes over it.

## 13. Why is failure-path slower than go-playground/validator?

go-zod finalizes Zod-shaped issues (error-map chain, locales, rich JSON). Tag validators are thinner on the error path by design. Happy-path object parse is within ~20% of validator on FlatUser/Nested.

## 14. How do I set a locale?

```go
cfg := z.GetConfig()
cfg.LocaleError = z.EsLocale // FrLocale, DeLocale, JaLocale, PtLocale, ZhLocale
```

Or pass a per-parse `ParseCtx{Error: myMap}`.

## 15. Can `ToStruct` target a pointer type?

No — `T` must be a non-pointer struct. Pointer **fields** inside the struct are fine (`*string`, `*time.Time`).

## 16. How do I abort refinements early?

```go
z.Refine(schema, pred, z.RefineOpts{Abort: true})
// or
ctx.AddIssue(iss.WithAbort())
```

`Custom` defaults to abort; `Refine` defaults to continue.

## 17. What’s `Readonly` for?

API parity with Zod. It’s a documented no-op in Go (no `Object.freeze` for maps).

## 18. Module / import paths?

```go
import z "github.com/iKunalChhabra/go-zod"         // alias z (Zod convention)
import "github.com/iKunalChhabra/go-zod/zgin"      // Gin helpers
```

Requires Go 1.22+.
