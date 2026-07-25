# Array

`z.Array(element)` ports `z.array(...)`. Output is `[]any`. Typed Go slices (e.g. `[]string`) are accepted and converted element-wise.

```go
tags := z.Array(z.String()).Min(1).Max(10)

out, err := tags.Parse([]any{"go", "zod"})
// out == []any{"go", "zod"}

out, err = tags.Parse([]string{"a", "b"}) // reflect-coerced
```

## Basics

```go
schema := z.Array(z.Number())
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
| `NonEmpty` | `Min(1)` |

```go
minTwo := z.Array(z.String()).Min(2, "need two")
maxTwo := z.Array(z.String()).Max(2)
justTwo := z.Array(z.String()).Length(2)
nonEmpty := z.Array(z.String()).NonEmpty()

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
schema := z.Array(z.String().Email())
res := schema.SafeParse([]any{"ok@x.co", "nope"})
// Issues[0].Path == []any{1}
// Format == "email"

nested := z.Array(z.Object(z.Shape{
    "id": z.String(),
}))
res = nested.SafeParse([]any{
    map[string]any{"id": "a"},
    map[string]any{},
})
// Path == []any{1, "id"}
```

## Element accessor

```go
schema := z.Array(z.Int())
_ = schema.Element() // the Int schema
```

## Custom messages

```go
schema := z.Array(z.String(), "expected a list")
res := schema.SafeParse(map[string]any{})
// Message: "expected a list"
```

## API surface

```go
func Array(elem AnySchemaLike, params ...any) *ArraySchema
func (s *ArraySchema) Element() AnySchemaLike
func (s *ArraySchema) Min(n int, params ...any) *ArraySchema
func (s *ArraySchema) Max(n int, params ...any) *ArraySchema
func (s *ArraySchema) Length(n int, params ...any) *ArraySchema
func (s *ArraySchema) NonEmpty(params ...any) *ArraySchema
```
