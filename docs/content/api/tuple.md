# Tuple

`zod.Tuple(items)` ports `z.tuple([...])`. Fixed positions plus optional trailing `Rest`. Output is `[]any`.

```go
pair := zod.Tuple([]zod.AnySchemaLike{
    zod.String(),
    zod.Number(),
})

out := pair.MustParse([]any{"age", 42})
// out[0] == "age", out[1] == 42.0
```

## Fixed length

Without `Rest`, length must match the item list (accounting for optional trailing elements):

```go
schema := zod.Tuple([]zod.AnySchemaLike{zod.String(), zod.String()})

schema.MustParse([]any{"asdf", "1234"})

res := schema.SafeParse([]any{"asdf"})
// too_small, Origin: "array", Minimum: 2

res = schema.SafeParse([]any{"asdf", "1234", true})
// too_big, Origin: "array", Maximum: 2

res = schema.SafeParse(map[string]any{})
// Expected: "tuple"
```

## Element type errors & paths

```go
schema := zod.Tuple([]zod.AnySchemaLike{zod.String(), zod.String()})
res := schema.SafeParse([]any{"asdf", 1234})
// Path: [1], Code: invalid_type, Expected: "string"
```

## Rest

`.Rest(schema)` accepts zero or more additional elements of that type:

```go
schema := zod.Tuple([]zod.AnySchemaLike{zod.String()}).Rest(zod.String())

schema.MustParse([]any{"a"})
schema.MustParse([]any{"a", "b", "c"})

res := schema.SafeParse([]any{"a", 1})
// Path: [1] — rest element type error
```

## Optional elements

Trailing `Optional(...)` items may be omitted. Optional-out failures for absent slots are swallowed (Zod tuple semantics). Combine with `Rest`:

```go
schema := zod.Tuple([]zod.AnySchemaLike{
    zod.String(),
    zod.Optional(zod.String()),
    zod.Optional(zod.String()),
}).Rest(zod.String())

schema.MustParse([]any{"asdf"})
schema.MustParse([]any{"asdf", "1234"})
schema.MustParse([]any{"asdf", "1234", "asdf"})
schema.MustParse([]any{"asdf", "1234", "asdf", "true", "false"})
```

:::warn Optionals must be trailing
Like Zod, optional tuple slots should be at the end of the fixed prefix. Leading optionals break the “first required index” calculation.
:::

## Inspect items

```go
t := zod.Tuple([]zod.AnySchemaLike{zod.String(), zod.Bool()})
items := t.Items() // copy of the fixed schemas
```

## API surface

```go
func Tuple(items []AnySchemaLike, params ...any) *TupleSchema
func (t *TupleSchema) Rest(schema AnySchemaLike) *TupleSchema
func (t *TupleSchema) Items() []AnySchemaLike
```
