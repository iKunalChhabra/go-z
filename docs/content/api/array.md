# Array

`zod.Array(element)` ports `z.array(...)`. Output is `[]any`. Typed Go slices (e.g. `[]string`) are accepted and converted element-wise.

```go
tags := zod.Array(zod.String()).Min(1).Max(10)

out, err := tags.Parse([]any{"go", "zod"})
// out == []any{"go", "zod"}

out, err = tags.Parse([]string{"a", "b"}) // reflect-coerced
```

## Basics

```go
schema := zod.Array(zod.Number())
schema.MustParse([]any{1, 2, 3})

res := schema.SafeParse("nope")
// Expected: "array"
```

## Length checks

Issues use **`Origin: "array"`** (not `"string"`).

| Method | Rule |
|--------|------|
| `Min(n)` | len ≥ n |
| `Max(n)` | len ≤ n |
| `Length(n)` | len == n (`Exact: true` on failure) |
| `NonEmpty()` | `Min(1)` |

```go
minTwo := zod.Array(zod.String()).Min(2, "need two")
maxTwo := zod.Array(zod.String()).Max(2)
justTwo := zod.Array(zod.String()).Length(2)
nonEmpty := zod.Array(zod.String()).NonEmpty()

minTwo.MustParse([]any{"a", "b"})
res := minTwo.SafeParse([]any{"a"})
// Message: "need two"
// Origin: "array", Code: too_small

_ = maxTwo.SafeParse([]any{"a", "b", "c"}) // too_big
justTwo.MustParse([]any{"a", "b"})
_ = nonEmpty.SafeParse([]any{})
```

## Element paths

Element failures are path-prefixed with the **index**:

```go
schema := zod.Array(zod.String().Email())
res := schema.SafeParse([]any{"ok@x.co", "nope"})
// Issues[0].Path == []any{1}
// Format == "email"

nested := zod.Array(zod.Object(zod.Shape{
    "id": zod.String(),
}))
res = nested.SafeParse([]any{
    map[string]any{"id": "a"},
    map[string]any{},
})
// Path == []any{1, "id"}
```

## Element accessor

```go
schema := zod.Array(zod.Int())
_ = schema.Element() // the Int schema
```

## Custom messages

```go
schema := zod.Array(zod.String(), "expected a list")
res := schema.SafeParse(map[string]any{})
// Message: "expected a list"
```

## API surface

```go
func Array(elem AnySchemaLike, params ...any) *ArraySchema
func (s *ArraySchema) Element() AnySchemaLike
func (s *ArraySchema) Min / Max / Length / NonEmpty(...)
```
