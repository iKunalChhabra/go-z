# Literal & Enum

Literals and enums constrain values to a closed set. Both populate `Internals().Values`, which [discriminated unions](/api/discriminated-union) and [enum-keyed records](/api/record) use for exhaustive matching.

## Literal

`zod.Literal(values...)` ports `z.literal(...)`. Multi-value literals are first-class.

```go
zod.Literal("tuna").MustParse("tuna")
zod.Literal(42).MustParse(42)
zod.Literal(42).MustParse(float64(42)) // JSON numbers arrive as float64
zod.Literal(true).MustParse(true)
```

### Failure

```go
res := zod.Literal("tuna").SafeParse("shark")
// Code: invalid_value
// Values: ["tuna"]
// Message: `Invalid input: expected "tuna"`
```

### Multi-value

```go
s := zod.Literal("a", "b", 1)
s.MustParse("a")
s.MustParse("b")
s.MustParse(1)

res := s.SafeParse("c")
// Message: `Invalid option: expected one of "a"|"b"|1`
```

You can also pass a single `[]any` or `[]string` as the value list:

```go
zod.Literal([]string{"red", "green", "blue"}).MustParse("green")
```

### Params vs values

A trailing `string` / `ErrorMap` / `Params` is treated as schema params **when** there is at least one preceding value (a sole string is a literal value):

```go
s := zod.Literal("tuna", "That's not a tuna")
res := s.SafeParse("shark")
// Message: "That's not a tuna"

zod.Literal("hello") // the value is "hello", not a message
```

### BigInt literals

```go
import "math/big"

s := zod.Literal(big.NewInt(12))
s.MustParse(big.NewInt(12))
res := s.SafeParse(big.NewInt(13))
// Message: "Invalid input: expected 12n"
```

### Values / Value helpers

```go
lit := zod.Literal("a", "b")
lit.Values() // []any{"a", "b"}

zod.Literal("only").Value() // "only"
// lit.Value() panics when multiple values exist
```

### Internals.Values for discriminators

```go
role := zod.Literal("admin", "guest")
vals := role.Internals().Values
_, ok := vals["admin"] // true

// Integral number literals are also indexed as float64 for JSON discriminants
n := zod.Literal(1)
_, ok = n.Internals().Values[float64(1)] // true
```

```go
user := zod.DiscriminatedUnion("role", []zod.AnySchemaLike{
    zod.Object(zod.Shape{
        "role":  zod.Literal("admin"),
        "perms": zod.Array(zod.String()),
    }),
    zod.Object(zod.Shape{
        "role":    zod.Literal("guest"),
        "session": zod.String().UUID(),
    }),
})
```

## Enum

`zod.Enum("a", "b", ...)` ports `z.enum([...])`. Members are strings; output type is `string`.

```go
color := zod.Enum("red", "green", "blue")
color.MustParse("red")

res := color.SafeParse("yellow")
// Code: invalid_value
// Values includes "red", "green", "blue"

color.Options() // []string{"red", "green", "blue"}
```

`Internals().Values` contains each option for discriminant / record exhaustiveness:

```go
_, ok := color.Internals().Values["green"] // true
```

### NativeEnum

`zod.NativeEnum(map[string]string)` ports `z.nativeEnum` / `z.enum(object)`. **Accepted values are the map values**, not the keys.

```go
Status := map[string]string{
    "Pending": "pending",
    "Active":  "active",
    "Done":    "done",
}

schema := zod.NativeEnum(Status)
schema.MustParse("pending")
_ = schema.SafeParse("Pending") // key is not accepted — value is

schema.EnumMap() // copy of the key→value map
schema.Options() // the values (order not guaranteed from map iteration)
```

Custom message via params:

```go
schema := zod.NativeEnum(Status, "invalid status")
res := schema.SafeParse("nope")
// Message: "invalid status"
```

:::warn Enum args are all values
`Enum("active", "inactive", "must be a status")` treats the third string as a **member**, not a message. Use `EnumWith` for params:

```go
schema := zod.EnumWith([]string{"active", "inactive"}, "must be a status")
```
:::

## Literal vs Enum

| | Literal | Enum |
|--|---------|------|
| Value types | any comparable-ish (`string`, numbers, bool, `*big.Int`, …) | strings only |
| Multi-value | yes | yes (variadic strings) |
| `Internals.Values` | yes | yes |
| Typical use | discriminators, constants | closed string sets |

## API surface

```go
func Literal(values ...any) *LiteralSchema
func (s *LiteralSchema) Values() []any
func (s *LiteralSchema) Value() any

func Enum(values ...string) *EnumSchema
func NativeEnum(m map[string]string, params ...any) *EnumSchema
func (s *EnumSchema) Options() []string
func (s *EnumSchema) EnumMap() map[string]string
```
