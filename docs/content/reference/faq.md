# FAQ

## 1. Does `.Optional` keep my type?

Yes. Wrappers are generic, so the fluent methods preserve the inner type:

```go
name, err:= z.String.Optional.Parse(v)   // (*string, error) — nil when absent
age, err:= z.Int64.Default(18).Parse(v)    // (int64, error)
```

`Optional` and `Nullable` return `*T` because their output domain is “a T or nothing”; every other wrapper (`Default`, `Prefault`, `Catch`, `NonOptional`, `Readonly`) always produces a value, so it returns `T`. The package-level `z.Optional(schema)` still takes an `AnySchemaLike` for heterogeneous containers and yields the erased `*OptionalSchema[any]`; use `z.OptionalOf(schema)` (or the fluent method) when the inner type is known. See [Optional & friends](#/api/optional).

## 2. What’s the difference between `Missing` and `nil`?

`Missing` is `undefined` (absent key). `nil` is JSON `null`. `Optional` accepts Missing; `Nullable` accepts nil; `Nullish` accepts both. See [Missing vs nil](#/guide/missing-nil).

## 3. Why do objects return `map[string]any`?

go-z follows untyped JSON core. Use `z.ToStruct[T](objectSchema)` when you want a Go struct. There is no compile-time `z.infer`.

## 4. Are schemas safe for concurrent use?

Yes. Fluent methods clone; finished schemas are immutable and lock-free. Share them across goroutines / Gin handlers freely.

## 5. Should I rebuild schemas per request?

No. Build once at package init (or startup) and reuse. Object plans and `ToStruct` plans are compiled/cached at construction.

## 6. When should I use `DiscriminatedUnion` vs `Union`?

Use `DiscriminatedUnion` for object variants that share a literal tag — O(1) dispatch and clearer errors. Use `Union` for primitives or shapes without a discriminator.

## 7. Why did my `Default` accept an invalid fallback?

`Default` does **not** re-parse the fallback (by design). Use `Prefault` if the default must pass the inner schema.

## 8. How do I validate Gin query / path params that are strings?

Use `z.Coerce.Number` / `Bool` / `Time` with `zgin.BindQuery` / `BindURI`. Query values are unwrapped by `CoerceQueryValues`.

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

## 11. Where is `.encode` / `.decode` / codecs?

`z.Codec(in, out, z.CodecTx{Decode: …, Encode: …})` with `z.Decode` / `z.Encode` / `z.SafeDecode` / `z.SafeEncode` and `z.InvertCodec`. Pipes reverse on encode, and `Default` / `Prefault` / `Catch` are skipped in the encode direction.

## 11b. Why do bad params panic instead of returning an error?

Params are only read while a schema is being **defined** — normally at package init or startup — so a typo like `z.String.Min(5, 42)` fails immediately and loudly rather than silently dropping your custom message. Schema definition is effectively compile time for an application: if the process starts, every schema in it was built with valid params. Nothing in the request path can panic from this.

## 12. How do I do recursive types?

```go
var Node z.AnySchemaLike
Node = z.Lazy(func z.AnySchemaLike {
    return z.Object(z.Shape{
        "children": z.Array(Node),
    })
})
```

Declare the `var` first, then assign `Lazy` that closes over it.

## 13. Why is failure-path slower than go-playground/validator?

go-z finalizes schema-shaped issues (error-map chain, locales, rich JSON). Tag validators are thinner on the error path by design. Happy-path object parse is within ~20% of validator on FlatUser/Nested.

## 14. How do I set a locale?

```go
cfg:= z.GetConfig
cfg.LocaleError = z.EsLocale // FrLocale, DeLocale, JaLocale, PtLocale, ZhLocale
```

Or pass a per-parse `ParseCtx{Error: myMap}`.

## 15. Can `ToStruct` target a pointer type?

No — `T` must be a non-pointer struct. Pointer **fields** inside the struct are fine (`*string`, `*time.Time`).

## 16. How do I abort refinements early?

```go
z.Refine(schema, pred, z.RefineOpts{Abort: true})
// or
ctx.AddIssue(iss.WithAbort)
```

`Custom` defaults to abort; `Refine` defaults to continue.

## 17. What’s `Readonly` for?

API parity with the TypeScript original. It’s a documented no-op in Go (no `Object.freeze` for maps).

## 18. Module / import paths?

```go
import "github.com/iKunalChhabra/go-z/z"         // package is named z
import "github.com/iKunalChhabra/go-z/zgin"      // Gin helpers
```

Requires Go 1.22+.
