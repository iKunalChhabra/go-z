# Why go-zod?

Zod changed how TypeScript apps validate data: schema as value, fluent composition, structured issues, and first-class error maps. go-zod brings those **patterns and that architecture** to Go — not a half-hearted “Zod-like” surface over struct tags.

## The problem with “just use tags”

Struct-tag validators (`validate:"required,email,min=2"`) are convenient for simple CRUD, but they push you into a different model:

| Tag validators | Schema-as-value (Zod / go-zod) |
|---|---|
| Rules live in string tags | Rules are typed Go builders |
| One struct ≈ one shape | Compose, wrap, reuse schemas |
| Errors are often flat strings | Issues have codes, paths, origins |
| Hard to share across JSON / query / form | Same schema, different entry points |
| Optionality = zero values + pointers | Explicit `Optional` / `Nullable` / `Missing` |

Tags couple validation to a Go type. Zod couples validation to a **schema value**. go-zod keeps that second model, because APIs are JSON-shaped long before they become structs.

```go
// Reusable schema value — not a tag string
email := z.String().Email()

user := z.Object(z.Shape{
	"email": email,
	"backup": z.Optional(email),
})
```

## Why port Zod’s architecture (not only the names)

Many Zod-inspired Go libraries copy method names and stop there. go-zod ports the **runtime design** that makes Zod predictable under composition:

### 1. Payload: accumulate, don’t throw mid-parse

Parsing threads a `Payload{Value, Issues}` through the pipeline. Containers prefix child issue paths. Failures don’t unwind the stack with the first problem — you get the whole picture.

```go
// Conceptual flow (simplified)
p := z.AcquirePayload(input)
schema.Internals().Run(p, ctx)
if len(p.Issues) > 0 {
	return newZodError(p.Issues, ctx)
}
return p.Value
```

That matches Zod’s `ParsePayload` mental model and unlocks good UX: multiple field errors in one response.

### 2. Schema = Def + parse + run

Every schema has:

- a **Def** — kind (`"string"`, `"object"`, …), attached checks, optional schema-level error map
- a **parse** function — bare type check / normalize (no checks)
- a **run** function — parse, then checks (or `run == parse` when there are zero checks — fast path)

Fluent methods clone the def and append checks. Zero-check schemas stay cheap.

### 3. Checks: when / abort / continue / onattach

Constraints like `Min` and `Email` are **check objects**, not hard-coded branches inside each schema type. They support:

- **When** — gate whether the check runs
- **Abort** — stop later checks on failure
- **continue** semantics on issues — non-aborting checks let the chain proceed
- **OnAttach** — stash metadata on the schema bag (e.g. minimum length hints)

See [Checks & refinements](#/guide/checks).

### 4. Issue taxonomy that matches Zod JSON

Eleven codes, same strings as Zod v4:

`invalid_type` · `too_big` · `too_small` · `invalid_format` · `not_multiple_of` · `unrecognized_keys` · `invalid_union` · `invalid_key` · `invalid_element` · `invalid_value` · `custom`

Clients and shared docs can speak one language across TypeScript and Go services.

### 5. Error-map resolution chain

Message resolution order mirrors Zod:

1. Explicit message on the issue  
2. Producing schema/check error map  
3. Per-parse `ParseCtx.Error`  
4. Global `Config.CustomError`  
5. Locale map (`EnLocale` by default)  
6. Fallback `"Invalid input"`

Locales ship for `en es fr de ja pt zh`. See [Error maps & locales](#/guide/error-maps).

## Architecture at a glance

```text
                    ┌─────────────────────────┐
   input (any) ───► │ Payload{value, issues}  │
                    └───────────┬─────────────┘
                                │
                    ┌───────────▼─────────────┐
                    │ Internals.Run           │
                    │  1. type parse          │
                    │  2. runChecks (checks)  │
                    └───────────┬─────────────┘
                                │
              issues? ──yes──► FinalizeIssue → *ZodError
                     │
                     no
                     ▼
                 typed T / map[string]any
```

Schemas are **immutable after construction**. Sharing one `var User = z.Object(...)` across thousands of goroutines is the intended usage pattern — see [Concurrency](#/guide/concurrency).

## Untyped core, typed edge

Like Zod’s TypeScript runtime (which is dynamically typed under `z.infer`), go-zod’s engine runs on `any`. Generics appear at the boundary:

```go
// Schema[T] — typed Parse surface
var s z.Schema[string] = z.String().Min(1)

out, err := s.Parse("hello") // out is string

// Containers hold heterogeneous children via AnySchemaLike
z.Object(z.Shape{
	"a": z.String(),
	"b": z.Int(),
})
```

Output-type-changing operations (`Transform`, `Pipe`, `ToStruct`) are package-level functions because Go methods cannot introduce new type parameters. That’s an honest language boundary — not a missing feature.

## When go-zod is the right tool

- You want **Zod muscle memory** in a Go backend  
- You validate **JSON / maps / dynamic shapes** before or instead of structs  
- You need **structured, code-stable issues** for API clients  
- You care about **locales**, refinements, unions, discriminated unions, lazy recursion  
- You run **high-concurrency** HTTP handlers and want lock-free schema reuse  

## When tags may still win

- Extremely simple structs with a handful of `required` / `email` rules  
- You already standardize on `go-playground/validator` across a large monorepo and don’t need composition  
- You need compile-time proof that every field of a struct is covered (neither tags nor go-zod give you Zod’s `z.infer` — see [Comparison](#/guide/comparison))

:::info Honest about types
TypeScript Zod infers static types from schemas (`z.infer<typeof schema>`). Go cannot do that. go-zod gives you `Schema[T]` and `ToStruct[T]`, but you still write (or generate) the Go types yourself. The win is runtime composition and issue quality — not deleting type declarations.
:::

## Design principles (short)

1. **Port the model** — payload, checks, issue codes, error maps  
2. **JSON first** — `map[string]any` / `[]any`, structs optional  
3. **Immutable schemas** — clone on fluent call, share freely  
4. **Parity where it matters** — messages and codes match Zod; Go idioms where Go differs (`error`, generics limits)  
5. **Optional integrations** — Gin lives in `zgin`, not the core import graph  

Next: [Comparison](#/guide/comparison) for a side-by-side with other Go validators and TypeScript Zod.
