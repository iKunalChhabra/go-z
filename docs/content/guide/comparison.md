# Comparison

How go-z sits next to popular Go validators and TypeScript Zod. The goal is clarity, not a slam dunk — pick the tool that matches your validation model.

## Quick matrix

| Dimension | **go-z** | **go-playground/validator** | **ozzo-validation** | **Oudwins/zog** | **TypeScript Zod** |
| --- | --- | --- | --- | --- | --- |
| Style | Schema-as-value, fluent | Struct tags + reflect | Fluent rules on values | Zod-inspired schemas | Schema-as-value, fluent |
| Primary input | `any` / JSON maps | Structs | Values / structs | Maps / structs | `unknown` |
| Composition | First-class (Object, Union, Lazy, Pipe…) | Limited (nested structs) | Good rule composition | Good | Excellent |
| Issue model | Zod v4 codes + paths | Field errors (validator) | Custom error objects | Zod-like | Zod v4 codes + paths |
| Error maps / locales | Check → parse → custom → locale (`en es fr de ja pt zh`) | Translation hooks (diy) | Custom messages | Partial | Full Zod locales |
| Schema → type inference | **No** (Go limitation) | N/A (types first) | N/A | **No** | **Yes** (`z.infer`) |
| Concurrency story | Immutable schemas, lock-free `Parse` | Safe if structs not mutated | Safe per call | Varies | Single-threaded JS typical |
| Gin helpers | `zgin` module | Community / diy | diy | diy | N/A |
| Parallel large arrays | `ParseParallelSlice` | diy | diy | diy | N/A |
| Maturity / ecosystem | Newer native port | Ubiquitous | Stable niche | Community Zod port | Industry standard (TS) |

:::info About “Zod for Go”
Oudwins/zog and go-z both aim at Zod-like DX. go-z specifically ports **Zod v4’s architecture** (payload accumulation, check semantics, issue taxonomy, error-map chain). Use the comparison below for product fit; benchmark for your shapes when latency matters.
:::

## vs go-playground/validator

**validator** is the default choice in many Gin tutorials:

```go
type User struct {
	Name  string `json:"name" validate:"required,min=2"`
	Email string `json:"email" validate:"required,email"`
}
```

**Strengths of validator**

- Ubiquitous docs, Stack Overflow answers, and middleware
- Natural fit when you already decode into structs
- Very fast for flat structs (often slightly ahead of go-z on microbenchmarks)

**Strengths of go-z**

- Schemas compose without inventing more tags
- Unions, discriminated unions, lazy recursion, pipe/transform
- Structured issues compatible with Zod-shaped API clients
- First-class `Optional` / `Nullable` / `Missing` (not just pointers + `omitempty`)
- Locales and error maps without rolling your own i18n layer

**Choose validator** if your world is “decode JSON into struct, run `Validate.Struct`.”
**Choose go-z** if your world is “define the contract as a schema, reuse it across handlers, return Zod-like issues.”

```go
// go-z equivalent (JSON-first)
user := z.Object(z.Shape{
	"name":  z.String().Min(2),
	"email": z.String().Email(),
})
data, err := user.Parse(input) // map[string]any

// Optional typed edge
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
u, err := z.ToStruct[User](user).Parse(input)
```

## vs ozzo-validation

ozzo-validation is fluent and Go-idiomatic:

```go
err := validation.ValidateStruct(&user,
	validation.Field(&user.Name, validation.Required, validation.Length(2, 100)),
	validation.Field(&user.Email, validation.Required, is.Email),
)
```

**Strengths of ozzo**

- Clear rule functions, no tag strings
- Nice for validating structs you already have in memory

**Strengths of go-z**

- Schema values you can store, pass, and nest arbitrarily
- Object / array / union / record model closer to JSON Schema / Zod
- Issue codes and path utilities (`Flatten`, `Treeify`, `Prettify`) out of the box

If you think in “rules attached to struct fields,” ozzo feels native. If you think in “schemas that produce and consume JSON shapes,” go-z fits better.

## vs Oudwins/zog

zog targets Zod-like schemas in Go and is a reasonable alternative in the same category.

**Where go-z focuses**

- Faithful Zod v4 port goals (issue codes, check `when`/`abort`/`continue`, error-map chain)
- Immutable, concurrency-oriented core
- `zgin` integration and `ParseParallelSlice`
- Locales: `en es fr de ja pt zh`

Evaluate both against your schemas. Prefer go-z when Zod v4 parity and Gin/perf tooling matter; prefer zog when its API surface already matches your codebase.

## vs TypeScript Zod

This is the emotional home of go-z.

| Concern | TypeScript Zod | go-z |
|---|---|---|
| Fluent API | `z.string().min(5).email()` | `z.String().Min(5).Email()` |
| Objects | `z.object({ ... })` | `z.Object(z.Shape{ ... })` |
| Safe parse | `safeParse` | `SafeParse` |
| Issues | `ZodError.issues` | `Error.Issues` (via `z.AsError`) |
| Infer types | `z.infer<typeof s>` | **Not available** — declare Go types separately |
| Runtime | JS / TS | Go `any` core + `Schema[T]` edge |
| Defaults / catch | `default`, `catch` | `Default`, `Catch` |
| Coercion | `z.coerce.string()` | `z.Coerce.String()` |

### The hard honesty: no schema→type inference

In TypeScript:

```ts
const User = z.object({
  name: z.string,
  email: z.string.email,
});
type User = z.infer<typeof User>; // free
```

In Go, the type system cannot infer a struct type from a runtime `Object` value. You write both:

```go
var UserSchema = z.Object(z.Shape{
	"name":  z.String(),
	"email": z.String().Email(),
})

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

parsed, err := z.ToStruct[User](UserSchema).Parse(input)
```

:::warn This is a language limit, not a backlog item
No Go Zod port can honestly claim TypeScript-grade inference without code generation. go-z chooses runtime fidelity + optional `ToStruct` over pretending otherwise.
:::

What you *do* keep across the language boundary:

- Same issue **codes** and roughly the same **messages** (locale-dependent)
- Same mental model for optional / nullable / nullish
- Same error utilities (`flatten` / `format` / `treeify` / `prettify`)

That makes polyglot teams happier: a TS frontend and a Go API can document one error contract.

## Performance snapshot

Headline numbers (4-core, Go 1.26.5, median of nine runs — see repo `BENCHMARKS.md` for methodology and spread):

| Scenario | go-z | go-playground/validator | Oudwins/zog |
|---|---:|---:|---:|
| Flat user | ~528 ns | ~637 ns | ~1258 ns |
| Nested object | ~1184 ns | ~1112 ns | ~2646 ns |
| Array 10k (parallel) | ~2.45 ms | ~6.28 ms | ~12.5 ms |

go-z leads on flat objects, string formats and large arrays; validator is ~6% ahead on nested objects and ~3.6× ahead on the failure path, where go-z pays for building structured issues. Remember the two are validating different things: go-z parses an untyped map, validator inspects a struct it is handed.

## Decision guide

```text
Need Zod parity / JSON-first schemas / structured issues?
  └─ yes → go-z (or evaluate zog)
Need "validate this struct I already have" with tags?
  └─ yes → go-playground/validator
Need fluent rules without schemas?
  └─ yes → ozzo-validation
Writing TypeScript?
  └─ use Zod itself; share issue codes with go-z on the backend
```

## Related

- [Why go-z?](#/guide/why) — architecture motivation
- [Quickstart](#/guide/quickstart) — try the API in ten minutes
- [Benchmarks](#/guide/benchmarks) — deeper numbers when they land in the docs site
