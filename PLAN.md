# go-zod — Native Go port of Zod v4

A faithful, high-performance Go port of [Zod v4](https://github.com/colinhacks/zod)
(`colinhacks/zod` @ `912f0f5`, `packages/zod/src/v4`), designed for first-class use in
[Gin](https://github.com/gin-gonic/gin). Module: `github.com/iKunalChhabra/go-zod`,
package `zod`.

This document is the master plan. It defines the architecture, the exact Zod
patterns being ported, the file-level ownership map for parallel sub-agents, and
the acceptance criteria for each work package.

---

## 1. What we are porting (Zod v4 architecture recap)

Zod v4 (`packages/zod/src/v4/core`) is built from these load-bearing patterns.
The Go port preserves every one of them:

| Zod v4 pattern | Source | Go port |
|---|---|---|
| `ParsePayload {value, issues[]}` threaded through parsing; issues accumulate, never throw mid-parse | `core/schemas.ts` | `ParsePayload` struct, pooled via `sync.Pool` |
| Schema = `def` (serializable definition) + `parse` (bare type check) + `run` (parse + checks); zero-check schemas fast-path `run = parse` | `core/core.ts`, `core/schemas.ts` | `internals` struct holding `Def`, `parseFn`, `runFn` function pointers; `runFn == parseFn` when no checks |
| Checks are composable objects with `check(payload)`, `onattach` hooks, optional `when` predicates, continue-after-failure + abort semantics | `core/checks.ts` | `ZodCheck` struct with `CheckFn`, `OnAttach`, `When`, `Abort` |
| 11 issue codes: `invalid_type`, `too_big`, `too_small`, `invalid_format`, `not_multiple_of`, `unrecognized_keys`, `invalid_union`, `invalid_key`, `invalid_element`, `invalid_value`, `custom` | `core/errors.ts` | `ZodRawIssue` / `ZodIssue` structs with identical `code` strings and fields; identical JSON serialization |
| Error-map precedence: per-check/schema error → per-parse ctx error → global `customError` → locale error map | `core/util.ts finalizeIssue` | identical chain in `finalizeIssue` |
| `ZodError` with `issues[]` + utilities `flatten`, `format`, `treeify`, `prettify`, `toDotPath` | `core/errors.ts` | same functions, same output shapes |
| Top-level `parse` / `safeParse` (+ async variants) | `core/parse.ts` | `Parse` (returns `(T, error)`), `MustParse` (panics), `SafeParse` (result struct). Go is synchronous; async variants are intentionally dropped |
| Wrappers: `optional`, `nullable`, `default`, `prefault`, `catch`, `nonoptional`, `readonly`, `pipe`, `transform`, `preprocess` | `core/schemas.ts` | same wrappers (`readonly` is a documented no-op) |
| Composites: `object` (strict/loose/catchall), `array`, `tuple` (+rest), `record`, `map`, `set` | `core/schemas.ts` | same, on `map[string]any` / `[]any` inputs (JSON model) + typed struct binding |
| `union`, `discriminatedUnion` (propValues fast path), `intersection`, `lazy` (recursion) | `core/schemas.ts` | same, incl. discriminator map fast path |
| ~54 string formats/regexes (email, uuid, cuid2, ulid, base64, jwt, ipv4/6, cidr, e164, ISO datetime…) | `core/regexes.ts`, `core/checks.ts` | same regexes (RE2-compatible translations where needed), same format names in issues |
| Refinements: `refine`, `superRefine`, `check`, `overwrite` (in-place transforms that keep inference) | `core/checks.ts`, `classic/schemas.ts` | same |
| Coercion: `z.coerce.string()/number()/boolean()/date()` | `core/api.ts` | `zod.Coerce.String()` etc., plus Gin query/form coercion |
| Registries + metadata: `.describe()`, `.meta()`, global registry | `core/registries.ts` | same |
| Locales (53 languages) | `locales/` | `en` complete + top locales; same message templates |
| JIT compilation of object parsers (`Doc` class, eval-based) | `core/doc.ts` | Go analog: **precompiled object plans** — field lists, check slices, and type-switch parsers resolved at schema build time, zero reflection in hot path |
| Method-chaining "classic" API (`z.string().min(5).email()`) | `classic/schemas.ts` | same chaining; each concrete schema's fluent methods return the same concrete type |

Out of scope for v0 (documented in README roadmap): codecs/`encode`/`decode`
(bidirectional pipes), JSON Schema conversion, `z.function()`, `z.promise()`,
template literals, the "mini" functional API.

## 2. Go API design (the critical decisions)

### 2.1 Typed surface over an untyped core — exactly like Zod

Zod's runtime is untyped (`ParsePayload.value: unknown`); static types are a
compile-time layer. The Go port does the same: the engine operates on `any`,
and generics provide the typed boundary:

```go
// The generic boundary. All schemas implement Schema[T].
type Schema[T any] interface {
    Parse(data any) (T, error)        // like z.parse
    MustParse(data any) T             // panics with *ZodError
    SafeParse(data any) SafeParseResult[T]
    internals() *Internals            // untyped core
}

// z.String() *StringSchema — fluent methods return *StringSchema
user := zod.Object(zod.Shape{
    "name":  zod.String().Min(2).Max(100),
    "email": zod.String().Email(),
    "age":   zod.Int().Gte(0).Lt(150).Optional(),
    "tags":  zod.Array(zod.String()).Max(10).Default([]any{}),
})
data, err := user.Parse(input)           // map[string]any
```

Because Go methods cannot introduce new type parameters, output-type-changing
operations are package-level functions (this is the only idiomatic deviation):

```go
zod.Transform(schema, func(s string, ctx *zod.RefinementCtx) int { ... }) // Schema[int]
zod.Pipe(a, b)                       // Schema[B]
zod.ToStruct[User](objectSchema)     // Schema[User], cached reflection plan
```

### 2.2 Concurrency & performance model

- **Immutable schemas**: all fluent methods clone (Zod's immutable API); a built
  schema is read-only ⇒ lock-free concurrent `Parse` from any number of goroutines.
- **Pooling**: `ParsePayload` and issue slices from `sync.Pool`; zero allocations
  on the happy path for primitives.
- **Precompiled plans**: object schemas resolve their field list, per-field
  parsers and check chains once at build time (Go analog of Zod's JIT `Doc`).
- **Fast paths**: no-check schemas skip the check loop entirely (`run == parse`);
  discriminated unions dispatch through a discriminator map.
- **Opt-in parallelism**: `ParseParallel(ctx, data, opts)` for large arrays /
  objects — chunked worker pool, deterministic issue ordering after join.
  Parallelism is opt-in with honest benchmarks: goroutine overhead beats
  validation cost only above a measured threshold (~1k elements).

### 2.3 Gin integration (`zgin` subpackage) — "same validation in Gin"

```go
type CreateUserReq = map[string]any
var createUser = zod.Object(zod.Shape{...})

r.POST("/users", func(c *gin.Context) {
    body, ok := zgin.BindJSON(c, createUser)   // 400 + Zod-shaped error JSON on failure
    if !ok { return }
    ...
})
// or middleware:
r.POST("/users", zgin.Validate(createUser), handler) // parsed value in c.Get(zgin.Key)
```

Error responses use Zod's exact serialization: `{"issues":[{"code":"too_small",
"minimum":2,"origin":"string","path":["name"],"message":"Too small: ..."}]}` with
`Flatten`/`Treeify`/`Prettify` renderers selectable. Query/form/URI binding uses
the coercion pipeline (`zod.Coerce`) like Zod's `z.coerce`.

## 3. Repository layout & file ownership

```
go.mod                      module github.com/iKunalChhabra/go-zod   (wave 0)
PLAN.md                     this file
zod/…                       (root package `zod`, one file per concern, mirrors core/*)

WAVE 0 — parent agent (foundation; defines every contract)
  payload.go     ParsePayload, pooling, ParseCtx (ReportInput, ErrorMap, Locale)
  issue.go       ZodRawIssue/ZodIssue, 11 code constants, finalizeIssue, JSON marshal
  errors.go      ZodError, SafeParseResult[T]
  errmap.go      ErrorMap type, precedence chain, Config (global custom/locale maps)
  check.go       ZodCheck, check runner (when/abort/continue semantics)
  schema.go      Internals, Def, runFn/parseFn wiring, clone; Schema[T] interface
  parse.go       Parse/MustParse/SafeParse generic entry points
  util.go        path handling, primitives, typeOf naming (Zod "expected" strings)
  locale_en.go   English locale (port of locales/en.ts message templates)
  zod.go         package doc + facade

WAVE 1 — five parallel sub-agents, disjoint files
  A strings:     schema_string.go, checks_string.go, regexes.go, formats.go(+tests)
  B numbers:     schema_number.go, checks_number.go, schema_bool.go, schema_time.go,
                 schema_literal.go, schema_enum.go, schema_special.go (+tests)
  C collections: schema_object.go, schema_array.go, schema_record.go, schema_map.go,
                 schema_set.go, schema_tuple.go (+tests)
  D unions:      schema_union.go, schema_discunion.go, schema_intersection.go,
                 schema_lazy.go (+tests)
  E wrappers:    schema_optional.go, schema_default.go, schema_catch.go,
                 schema_pipe.go, schema_transform.go, refine.go, coerce.go (+tests)

WAVE 2 — three parallel sub-agents, disjoint dirs/files
  F errors/meta: errorutils.go (Flatten/Format/Treeify/Prettify/ToDotPath),
                 registry.go, locales/ (es, fr, de, ja, pt, zh) (+tests)
  G gin:         zgin/ (bind.go, middleware.go, respond.go, coerce_query.go),
                 tostruct.go (struct binding w/ cached reflect plans) (+tests)
  H perf:        parallel.go (chunked workers), pool tuning, fastpath audit,
                 bench/ scaffolding (separate module) (+tests)

WAVE 3 — parent agent
  integration fixes, benchmark runs vs go-playground/validator + zog,
  BENCHMARKS.md with results, README.md, PR
```

**Rules for sub-agents** (avoid integration conflicts):
1. Only create/modify the files listed for your work package. Never touch
   another package's files, `go.mod`, or wave-0 files — if a wave-0 contract is
   insufficient, implement what you can and report the gap in your final message.
2. All schemas follow the wave-0 pattern: a `Def` struct, `newInternals` +
   `parseFn` closure, fluent methods that clone via `addCheck`/`withDef`.
3. Issue production must match Zod byte-for-byte: same `code`, same auxiliary
   fields (`expected`, `origin`, `minimum`, `maximum`, `inclusive`, `format`,
   `pattern`, `divisor`, `keys`, `values`, `discriminator`…), same default
   English messages (port `locales/en.ts` templates exactly).
4. Port the corresponding Zod v4 test files (`v4/classic/tests/*.test.ts`) into
   Go table tests; cite which cases you ported in comments.
5. `go build ./... && go vet ./... && go test ./...` must pass on your files.

## 4. Work-package specs (hand these to sub-agents)

### WP-A — Strings & formats
Port `$ZodString` + all string checks/formats from `core/schemas.ts`,
`core/checks.ts`, `core/regexes.ts`, and the fluent surface of `ZodString` in
`classic/schemas.ts`: `Min/Max/Length/Regex/Includes/StartsWith/EndsWith/
Uppercase/Lowercase/Trim/ToLowerCase/ToUpperCase/NormalizeNFC/NonEmpty` plus
formats `Email/URL/UUID/UUIDv4/6/7/GUID/NanoID/CUID/CUID2/ULID/KSUID/XID/Base64/
Base64URL/Hex/JWT/E164/Emoji/IPv4/IPv6/CIDRv4/CIDRv6/MAC/ISODate/ISOTime/
ISODateTime/ISODuration`. Overwrite checks (`Trim` etc.) mutate `payload.value`
(Zod's `$ZodCheckOverwrite`). Translate regexes to RE2 (Go `regexp`); where Zod
uses lookahead (e.g. duration), hand-code the predicate and keep the issue's
`pattern` field equal to Zod's regex string literal. Tests: port
`string.test.ts`, `string-formats.test.ts`.

### WP-B — Numbers, bools, time, literals, enums, specials
Port `$ZodNumber/$ZodNumberFormat` (float64 + Int/Int32/Uint32/Float32/Float64,
`safe` integer semantics), `Gt/Gte/Lt/Lte/Min/Max/Positive/Negative/
NonPositive/NonNegative/MultipleOf/Step/Finite/Int`; `$ZodBoolean`; `$ZodDate` →
`zod.Time()` (accepts `time.Time`, optional RFC3339 coercion) with `Min/Max`;
`$ZodLiteral` (multi-value), `$ZodEnum` (string enums + `NativeEnum` via map);
`Any/Unknown/Never/Nil/NaN`. Big-int: `zod.BigInt()` over `*big.Int`. Tests:
port `number.test.ts`, `bigint.test.ts`, `enum.test.ts`, `literal.test.ts`,
`date.test.ts`.

### WP-C — Object & collections
Port `$ZodObject` with the three modes (default strip? — **No**: Zod v4 objects
are non-strict by default and *keep* unknown keys out of validation but do not
error; `Strict()` errors with `unrecognized_keys`; `Loose()` passes them
through; `Catchall(schema)` validates them) — replicate v4 semantics exactly,
including key ordering of issues and `optin`/`optout` optionality handling
(`.Optional()` fields may be absent). Runtime ops: `Pick/Omit/Partial/
Required/Extend/Merge/Keyof`. Precompiled field plans at build time. Port
`$ZodArray` (`Min/Max/Length/NonEmpty`), `$ZodRecord` (key schema + value
schema, `invalid_key` issues), `$ZodMap` (Go `map[any]any` + `invalid_key`/
`invalid_element`), `$ZodSet` (Go has none: accept `[]any` with uniqueness or
`map[T]struct{}`; document), `$ZodTuple` (+ rest). Tests: port `object.test.ts`,
`array.test.ts`, `record.test.ts`, `map.test.ts`, `set.test.ts`, `tuple.test.ts`.

### WP-D — Unions, intersection, lazy
Port `$ZodUnion` (try all, collect per-option errors into one `invalid_union`
issue with `errors: [][]issue`), `$ZodDiscriminatedUnion` (build discriminator
value → option map from `propValues` at construction; single-pass dispatch;
fall back to `invalid_union` with `discriminator`), `$ZodIntersection` (parse
both, deep-merge results, `unmergeable` panic parity), `$ZodLazy` (`func()
Schema` for recursive types, memoized). Tests: port `union.test.ts`,
`discriminated-unions.test.ts`, `intersection.test.ts`, `recursive-types.test.ts`.

### WP-E — Wrappers, transforms, refinements, coercion
Port `$ZodOptional/$ZodNullable` (Go: `nil` handling + absent-key optionality
flags), `$ZodDefault` (short-circuits on missing/nil, supports func defaults),
`$ZodPrefault` (default *then* parse), `$ZodCatch` (fallback on any failure,
`CatchCtx` with issues), `$ZodNonOptional` (`nonoptional` issue), `Readonly`
(no-op, kept for API parity), `$ZodPipe` (A→B), `$ZodTransform` +
`Preprocess`, `Refine/SuperRefine/Check` with `RefinementCtx.AddIssue`
(custom issues, `abort`, `path`, `continue` semantics — port
`$ZodCheckCustom`), `Overwrite`, and the `Coerce` namespace
(string/number/bool/time following Zod's coercion tables). Tests: port
`optional.test.ts`, `nullable.test.ts`, `default.test.ts`, `catch.test.ts`,
`pipe.test.ts`, `transform.test.ts`, `refine.test.ts`, `coerce.test.ts`.

### WP-F — Error utilities, registry, locales
Port `flattenError/formatError/treeifyError/prettifyError/toDotPath` with
identical output shapes (incl. nested traversal of `invalid_union.errors`,
`invalid_key/invalid_element.issues`); `$ZodRegistry` (schema→meta map,
id→schema index) + `.Describe()/.Meta()`; locales `es fr de ja pt zh` ported
from `v4/locales/*.ts` (message parity with Zod's templates incl. unit words
and "expected/received" phrasing). Tests: port `error.test.ts` (format/flatten
sections), `registries.test.ts`, spot-check locales.

### WP-G — Gin integration & struct binding
`zgin` package: `BindJSON[T]`, `BindQuery[T]` (coercion pipeline), `BindURI[T]`,
`Validate(schema)` middleware storing the parsed value in the context; error
responder producing Zod-serialized issues with pluggable renderer
(`Issues|Flatten|Tree|Pretty`), `http.StatusBadRequest` default, content-type
negotiation. `tostruct.go`: `zod.ToStruct[T](objSchema)` — validated
`map[string]any` decoded into `T` through a cached reflection plan
(build-once per (schema,T) pair, no per-parse reflection walks; `json` tag
aware). End-to-end tests with `httptest` + Gin router, including error-shape
golden tests against real Zod v4 output.

### WP-H — Performance engine & benchmark scaffolding
`parallel.go`: `ParseParallelSlice(ctx, arraySchema, data, ParallelOpts{Workers,
MinChunk})` — chunked fan-out over worker goroutines, per-chunk payloads,
deterministic merge (issues re-indexed by chunk offset), `context.Context`
cancellation. Audit wave-0/1 hot paths: pooled payloads, preallocated issue
slices, no `fmt` in hot path, benchmark-guided inlining. `bench/` separate Go
module comparing: go-zod vs `go-playground/validator` vs `Oudwins/zog`
(+ raw hand-written baseline) on: (1) simple flat struct, (2) nested object
w/ 3 levels, (3) `[]item` × {100, 1k, 10k} incl. parallel mode, (4) string-
format-heavy schema, (5) failure paths w/ error rendering, (6)
`b.RunParallel` concurrent-load variants of 1–3. Output: `BENCHMARKS.md`.

## 5. Integration order & gates

1. Wave 0 lands first (parent). Gate: `go build`, unit tests on core.
2. Wave 1 agents run **in parallel** on disjoint files; parent integrates,
   fixes cross-package compile issues, runs full test suite.
3. Wave 2 agents run in parallel (F/G/H depend only on wave-0/1 contracts).
4. Wave 3 parent: benchmark runs on the CI machine, results committed, README.

Every wave ends with: `go build ./... && go vet ./... && go test ./...` green,
commit + push to `cursor/go-zod-port-8af4`, PR update.

## 6. Success criteria

- API parity: the Zod v4 README examples translate 1:1 to go-zod.
- Error parity: golden tests assert byte-identical issue JSON vs Zod v4 for a
  corpus of failing inputs (30+ cases across all issue codes).
- Test coverage: ported test cases from Zod's `classic/tests` for every WP.
- Performance: ≥ `Oudwins/zog` throughput on all benchmarks; within ~2× of
  `go-playground/validator` on struct-tag scenarios (it skips parse/convert);
  documented parallel speedup on 10k-element arrays.
- Thread safety: `-race` clean under `b.RunParallel` and concurrent tests.
