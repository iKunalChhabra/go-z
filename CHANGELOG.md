# Changelog

## Unreleased

Code-review fixes across `z` and `zgin`, with two user-visible behavior
tightenings.

### Breaking (behavior tightening)

- **`.Strict()` now reports `__proto__` as an unrecognized key.** It was
  silently dropped before. Matches Zod v4 strict-mode behavior (the JS-side
  skip only guards against prototype pollution, which does not exist in Go).
- **`ToStruct[T]`/`DecodeStruct[T]` reject lossy numeric conversions.**
  Non-integral floats into int fields (`3.9` → `3`), out-of-range values
  (`300` into `int8`), and negative values into unsigned fields now return a
  decode error instead of silently truncating or wrapping. Integer input into
  a string field now yields `"65"`, not the rune `"A"`.

### Fixed

- `ToStruct[T]`: stack overflow on recursive struct types (`Next *Node`),
  panic on fixed-size array fields (`[2]string`), undecodable nested slices,
  panic on non-empty interface fields; `json.Number` decoded exactly.
- Discriminated unions: forward-referenced/recursive `Lazy` options no longer
  panic at construction (dispatch table builds on first Parse; new `Resolve()`
  forces it at startup); self-referential `Lazy` cycles detected; a failed
  lazy build now fails consistently instead of leaving the union half-built.
- Exhaustive-key records no longer leak an internal sentinel for absent keys.
- `Map`/`Record` child schemas and `ParseParallelSlice`/`ConcurrentBatch`/
  `ConcurrentParseAny` now see the caller's `context.Context` in refinements.
- JSON Schema: `Nullable` preserved for `anyOf`/`oneOf`/`allOf`/`enum` inners;
  deterministic enum ordering (also `NativeEnum.Options()`).
- Locales: es/pt `base64url` mistranslations, empty-origin fallback in
  de/es/pt/zh `IssueTooSmall`, pt default fallback message.
- zgin: body cache re-validates per-bind options; request body restored after
  bind for downstream handlers; `AbortWithError(nil)` renders `"issues":[]`;
  new `BindJSONWithOptions`/`BindQueryWithOptions`/`BindURIWithOptions` for
  custom error status/format.
- Misc: `MultipleOf(0)` float passes like int variants; typed-nil
  `*time.Time` reports `"null"`; float→string coercion matches JS (`1e21` →
  `"1e+21"`); `__proto__` in strict mode (above); dead code removed.

## v0.2.0

Breaking changes to the numeric surface, correctness fixes across the schema
layer, request hardening in the Gin integration, and a documentation corpus that
is now compiled by the test suite.

### Breaking

- **Numeric constructors return the Go type they are named after.** `Int()` gives
  an `int`, `Int32()` an `int32`, `Uint32()` a `uint32`, `Float32()` a `float32`.
  `Number()`, `Float64()` and `Int64()` are unchanged. Inside an object,
  `out["age"].(float64)` becomes `out["age"].(int)`; `ToStruct[T]` is unaffected
  because it converts into whatever the struct field declares.
- `NumberSchema` and `Int64Schema` are now aliases for `NumericSchema[float64]`
  and `NumericSchema[int64]`. Code that names them still compiles; code that
  defines methods on them does not.
- `Number().Int()` is renamed `Number().Integer()`. The old name implied it
  changed the output type when it only added a check. `Safe()` remains an alias.
- `zgin` is a **separate module**. Install it with
  `go get github.com/iKunalChhabra/go-z/zgin`; the core module now depends on
  `golang.org/x/text` alone.
- `zgin.ContextKey` changed value from `"zod:value"` to `"go-z:value"`. Code using
  the constant is unaffected; code hardcoding the string is not.
- `zgin` JSON binders enforce a 1 MiB body limit, require a JSON `Content-Type`
  (415) and reject trailing data (400). All three are configurable through
  `BindOptions`.
- A `TemplateLiteral` part carrying a check with no pattern equivalent — `Min`,
  `Gte`, `Refine`, a transform — now fails at construction instead of being
  silently ignored.
- `ToJSONSchema` returns an error for an unrecognized `Target` instead of quietly
  emitting draft 2020-12.

### Added

- `NumericOf[T]` for widths without a named constructor (`uint16`, `uint64`) and
  for named types such as `type Port uint16`.
- `ParseAny` / `ParseAnyCtx`: validate through an `AnySchemaLike` without a typed
  wrapper.
- `ObjectShapeOf`: find an object's shape behind wrappers, for tooling that has to
  know the fields before parsing.
- `AllIssueCodes`, the enumerable issue taxonomy.
- `WorkerPanic`: a panic in a user closure inside a parallel worker is captured and
  re-raised on the calling goroutine, carrying the element index and worker stack.
- `zgin.BindOptions` (`MaxBodyBytes`, `AllowAnyContentType`, `AllowTrailingData`,
  `ContextKey`), `zgin.GetFrom` / `GetAsFrom`, `zgin.CoerceQueryValuesFor`.
- Fuzz targets for the hand-written matchers, JSON parsing and the numeric edges.

### Fixed

- Numeric bounds compare exactly instead of through `float64`, so
  `Int64().Lt(math.MaxInt64)` behaves at the boundary and `MultipleOf` uses integer
  remainder. Unsigned bounds travel as `uint64` rather than wrapping through
  `int64`. `float64(math.MinInt64)` parses.
- `TemplateLiteral` honours a part's `Regex` and format checks, which previously
  compiled to `[\s\S]*`. Fragments are grouped so alternation keeps its
  precedence, an escaped `$` survives the anchor trim, and raw `*regexp.Regexp`
  parts take the same path as schema parts.
- A panicking `Refine` or `Transform` inside `ParseParallelSlice`,
  `ConcurrentBatch` or `ConcurrentParseAny` no longer kills the process.
- `Intersection` unions both object shapes for unrecognized-key reporting, so a
  strict side keeps its strictness beside a loose one.
- `ToStruct` promotes embedded struct fields like `encoding/json`, allocating nil
  embedded pointers as it descends; decodes typed maps; refuses to `fmt.Sprint` an
  object into a string field; and caches decode plans by type instead of leaking
  one entry per schema instance.
- `ToJSONSchema` rewrites the document per target (draft-07 tuple form, OpenAPI 3.0
  `nullable`, single-value enums, boolean exclusive bounds), keeps the tighter of
  two bounds, and emits integer bounds exactly.
- `Treeify` bounds allocation for sparse path indices; `Treeify` and `TreeifyMap`
  share one implementation.
- `Registry.Describe` / `Meta` merge under the write lock, so concurrent calls no
  longer lose each other's fields.
- `parsePtr` reports an internal type mismatch instead of returning a nil that
  reads as "absent".
- `BigInt` returns a fresh `*big.Int`, so mutating the input cannot change a
  validated value; `asBigInt` accepts every Go integer width.
- All seven locales render the discriminated-union detail and know every format
  name and size unit.
- `zgin`: chained `Validate` middlewares work (the decoded body is cached),
  single-occurrence query parameters satisfy array schemas, and integers above
  2^53 survive decoding.

### Documentation

- Every Go block in the README and the docs site is parsed **and compiled** against
  the real modules by `go test ./z/`, and shell blocks are checked too. This
  replaced a corpus in which most samples could not compile.
- Benchmarks re-measured on Go 1.26.5 with the sampling protocol stated. Two
  headline numbers moved against go-z and are published that way: ~1.2× faster
  than `go-playground/validator` on flat objects (not ~1.5×) and ~6% slower on
  nested objects. A `ToStruct` row makes the struct-binding comparison fair.

## v0.1.0

First release: schema-first validation ported from Zod v4 — primitives and string
formats, objects and collections, unions, intersections, lazy recursion, wrappers,
codecs, `ToJSONSchema`, template literals, coercion, error utilities, seven
locales, Gin integration, struct binding and parallel parsing.
