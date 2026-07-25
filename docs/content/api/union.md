# Union

Try options in order; the first successful parse wins.

## Union

```go
schema := zod.Union([]zod.AnySchemaLike{
    zod.String(),
    zod.Number(),
})

schema.Parse("hi") // "hi"
schema.Parse(3.14) // 3.14
schema.SafeParse(true) // fails: invalid_union
```

Optional params: string message, `ErrorMap`, or `Params`.

```go
zod.Union(options, "must be string or number")
zod.Union(options, zod.Params{Error: myMap})
```

### UnionOf

Variadic convenience:

```go
schema := zod.UnionOf(zod.String(), zod.Number(), zod.Bool())
```

## Parse algorithm

1. **Empty options** → immediate `invalid_union` with empty `Errors`.
2. **Single option** → fast path: delegate directly to that schema (Zod’s `first` shortcut).
3. Otherwise try each option on a **pooled payload copy**:
   - First success → return its output.
   - All fail → continue.

### Surfacing a single continuable failure

If every option failed but **exactly one** result is non-aborted (e.g. refine/`continue: true` issues), those issues are surfaced **directly** on the parent — not wrapped in `invalid_union`. This matches Zod.

### invalid_union Errors

When multiple options abort (or zero non-aborted remain), go-zod emits:

```go
Issue{
    Code:   zod.IssueInvalidUnion,
    Errors: [][]Issue{ /* one slice per option, finalized */ },
    Input:  input,
}
```

`Errors[i]` is the finalized issue list from option `i`. Locale message for a plain union (no `Values`) is `"Invalid input"`.

```go
res := schema.SafeParse(true)
if !res.Success {
    iss := res.Error.Issues[0]
    // iss.Code == "invalid_union"
    // iss.Errors[0] — string option failures
    // iss.Errors[1] — number option failures
}
```

:::info Optionality of unions
If **any** option has OptIn / OptOut, the union inherits those flags. If every option exposes a `Values` set, the union merges them for discriminated-union / literal dispatch.
:::

## Traits & immutability

```go
u := zod.Union([]zod.AnySchemaLike{zod.String(), zod.Number()})
u2 := u.Check(myCheck) // immutable clone; Options are copied at construction
```

Callers cannot mutate the schema’s option list after construction — `Union` copies the slice.

## When to prefer DiscriminatedUnion

| Use `Union` when… | Use `DiscriminatedUnion` when… |
|---|---|
| Options are primitives / heterogeneous shapes without a shared key | Object variants share a literal discriminator (`type`, `role`, …) |
| Order-of-try is fine | You want O(1) dispatch + clearer discriminator errors |

See [Discriminated union](#/api/discriminated-union).

## Signatures

```go
func Union(options []AnySchemaLike, params ...any) *UnionSchema
func UnionOf(options ...AnySchemaLike) *UnionSchema

type UnionSchema struct {
    Options []AnySchemaLike
    // ...
}
func (s *UnionSchema) Check(checks ...*Check) *UnionSchema
```
