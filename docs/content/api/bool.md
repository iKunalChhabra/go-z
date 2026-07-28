# Bool

`z.Bool()` ports `z.boolean`. Output type is Go `bool`.

```go
schema := z.Bool()

schema.MustParse(true)
schema.MustParse(false)

res := schema.SafeParse("true")
// Without coerce: fail
// Message: "Invalid input: expected boolean, received string"
```

## Coercion table

Enable with `z.Params{Coerce: true}` or [`z.Coerce.Bool()`](/api/coerce).

| Input | Coerced to |
|-------|------------|
| `true` / `false` | unchanged |
| `"true"`, `"TRUE"`, `"1"` (trimmed, case-insensitive) | `true` |
| `"false"`, `"FALSE"`, `"0"` | `false` |
| `1`, `int64(1)`, `float64(1)` (and other 1-valued ints) | `true` |
| `0` / `0.0` | `false` |
| `"yes"`, `"no"`, `2`, `""`, other strings | **rejected** |

```go
s := z.Bool(z.Params{Coerce: true})

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

## StringBool

`z.StringBool()` parses **string tokens** into booleans — useful for env vars, CLI flags, and form values. Matching is trimmed and case-insensitive.

| Accepted as `true` | Accepted as `false` |
|---|---|
| `true`, `yes`, `1`, `on`, `y`, `enabled` | `false`, `no`, `0`, `off`, `n`, `disabled` |

```go
s := z.StringBool()
s.MustParse("true")    // true
s.MustParse("YES")     // true
s.MustParse(" off ")   // false

_ = s.SafeParse("maybe") // fail
_ = s.SafeParse(true)    // fail — real bools are rejected, strings only
```

Unlike `Coerce.Bool()`, `StringBool` accepts the wider `yes/no/on/off/enabled/disabled` vocabulary but rejects non-string input entirely.

## Custom messages

```go
schema := z.Bool("flag required")
res := schema.SafeParse("true")
// Message: "flag required" (still wrong type without coerce)
```

## API surface

```go
func Bool(params ...any) *BoolSchema
func StringBool(params ...any) *BoolSchema
func (s *BoolSchema) Check(checks ...*Check) *BoolSchema
```
