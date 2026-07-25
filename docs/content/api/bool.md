# Bool

`zod.Bool()` ports `z.boolean()`. Output type is Go `bool`.

```go
schema := zod.Bool()

schema.MustParse(true)
schema.MustParse(false)

res := schema.SafeParse("true")
// Without coerce: fail
// Message: "Invalid input: expected boolean, received string"
```

## Coercion table

Enable with `zod.Params{Coerce: true}` or [`zod.Coerce.Bool()`](/api/coerce).

| Input | Coerced to |
|-------|------------|
| `true` / `false` | unchanged |
| `"true"`, `"TRUE"`, `"1"` (trimmed, case-insensitive) | `true` |
| `"false"`, `"FALSE"`, `"0"` | `false` |
| `1`, `int64(1)`, `float64(1)` (and other 1-valued ints) | `true` |
| `0` / `0.0` | `false` |
| `"yes"`, `"no"`, `2`, `""`, other strings | **rejected** |

```go
s := zod.Bool(zod.Params{Coerce: true})

s.MustParse(true)
s.MustParse("true")
s.MustParse("FALSE")
s.MustParse("1")
s.MustParse("0")
s.MustParse(1)
s.MustParse(0)
s.MustParse(float64(1))

_ = s.SafeParse("yes") // fail
_ = s.SafeParse(2)     // fail
```

## Custom messages

```go
schema := zod.Bool("flag required")
res := schema.SafeParse("true")
// Message: "flag required" (still wrong type without coerce)
```

## API surface

```go
func Bool(params ...any) *BoolSchema
func (s *BoolSchema) Check(checks ...*Check) *BoolSchema
```
