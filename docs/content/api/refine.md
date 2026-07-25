# Refine, SuperRefine & Custom

Attach custom validation after (or instead of) structural parsing.

## Refine

```go
schema:= z.Refine(z.String, func(v any) bool {
    s, _:= v.(string)
    return strings.Contains(s, "@")
}, "must contain @")

schema.SafeParse("hello") // custom issue
```

Failed predicates produce a `custom` issue. Defaults to **continue** (non-abort) like `.refine`, so subsequent checks still run unless you pass `Abort: true`.

### Params

```go
z.Refine(inner, pred, "message")
z.Refine(inner, pred, z.Params{Error: map, Abort: true})
z.Refine(inner, pred, z.RefineOpts{
    Error:  z.MessageFromString("bad"),
    Abort:  true,
    Path:   []any{"field"},
    Params: map[string]any{"rule": "contains-at"},
})
```

`RefineOpts`:

| Field | Meaning |
|---|---|
| `Error` | Error map / message for the issue |
| `Abort` | Abort subsequent checks on failure |
| `Path` | Issue path segments |
| `Params` | Free-form metadata on the issue (`Params` JSON field) |

## SuperRefine

```go
schema:= z.SuperRefine(z.Object(z.Shape{
    "password": z.String,
    "confirm":  z.String,
}), func(v any, ctx *z.RefinementCtx) {
    m:= v.(map[string]any)
    if m["password"] != m["confirm"] {
        ctx.AddIssue(z.Issue{
            Code:    z.IssueCustom,
            Message: "passwords must match",
            Path:    []any{"confirm"},
        }.WithContinue)
    }
})
```

Use when you need multiple issues, custom paths, or access to `RefinementCtx`.

## RefinementCtx

```go
type RefinementCtx struct { /*... */ }

func (ctx *RefinementCtx) Value any
func (ctx *RefinementCtx) AddIssue(iss Issue)
func (ctx *RefinementCtx) AddMessage(msg string) // shorthand custom issue
```

| Method | Notes |
|---|---|
| `Value` | Current payload value |
| `AddIssue` | Defaults `Code` to `custom`; defaults `continue: true` |
| `AddMessage` | `AddIssue` with a string message |

Control abort behavior with `Issue.WithAbort` / `Issue.WithContinue`.

## CheckSchema

Composition primitive for attaching raw `*Check` values to any `AnySchemaLike`:

```go
schema:= z.CheckSchema(z.String, z.MinLength(5), myCheck)
```

Fluent schemas (`String.Check(...)`) already expose this; `CheckSchema` is for wrappers / unions that only satisfy `AnySchemaLike`.

## Custom

```go
schema:= z.Custom(func(v any) bool {
    _, ok:= v.(MyType)
    return ok
}, "expected MyType")
```

Accepts **any** input type, then runs the predicate. Defaults to **`abort: true`** (like `z.custom`), unlike `Refine`.

```go
// Override abort default
z.Custom(pred, z.RefineOpts{Abort: false})
```

## OverwriteSchema

See [Pipe & Transform](#/api/pipe-transform) — in-place rewrite check after inner parse.

## Pattern summary

| API | Input type check | Failure | Default abort |
|---|---|---|---|
| `Refine` | Uses inner schema | `custom` | continue |
| `SuperRefine` | Uses inner schema | via ctx | continue |
| `CheckSchema` | Uses inner + checks | check-defined | per check |
| `Custom` | none (any) | `custom` | **abort** |

## Typed refinements for any schema

`RefineOf` / `SuperRefineOf` / `OverwriteOf` take a typed predicate and work with **every** schema type, including ones with no fluent `Refine` method of their own:

```go
nonEmpty:= z.RefineOf(z.Set(z.String), func(v []any) bool {
    return len(v) > 0
}, "set must not be empty")

upper:= z.OverwriteOf(z.String, strings.ToUpper)
```

The fluent `Refine` / `SuperRefine` methods on `String`, `Number`, `Int64`, `Bool`, `Time`, `Enum`, `BigInt`, `Object`, `Array`, `Tuple`, `Record`, `Map`, and `Set` are sugar that returns the **same** concrete schema type, so type-specific chaining continues (`z.Set(...).Refine(...).Min(1)`). The generic helpers return a `*CheckedSchema[T]`, which ends the type-specific chain but keeps the typed edge.

## Signatures

```go
// Type-erased
func Refine(inner AnySchemaLike, pred func(any) bool, params...any) *CheckedSchema[any]
func SuperRefine(inner AnySchemaLike, fn func(any, *RefinementCtx), params...any) *CheckedSchema[any]
func CheckSchema(inner AnySchemaLike, checks...*Check) *CheckedSchema[any]
func Custom(pred func(any) bool, params...any) *CustomSchema
func OverwriteSchema(inner AnySchemaLike, fn func(any) any) *CheckedSchema[any]

// Typed — works with every schema type
func RefineOf[T any](inner Schema[T], pred func(T) bool, params...any) *CheckedSchema[T]
func SuperRefineOf[T any](inner Schema[T], fn func(T, *RefinementCtx), params...any) *CheckedSchema[T]
func OverwriteOf[T any](inner Schema[T], fn func(T) T) *CheckedSchema[T]
func CheckOf[T any](inner Schema[T], checks...*Check) *CheckedSchema[T]
```
