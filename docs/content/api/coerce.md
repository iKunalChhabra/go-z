# Coercion

`zod.Coerce` mirrors Zod’s `z.coerce` namespace. Each entry returns the corresponding primitive schema with `Def.Coerce = true`.

```go
// Equivalent pairs:
zod.Coerce.String()  // == zod.String(zod.Params{Coerce: true})
zod.Coerce.Number()  // == zod.Number(zod.Params{Coerce: true})
zod.Coerce.Bool()    // == zod.Bool(zod.Params{Coerce: true})
zod.Coerce.Time()    // == zod.Time(zod.Params{Coerce: true})
```

:::tip When to coerce
Ideal for **query strings**, **HTML forms**, and loosely typed JSON where everything arrives as a string. Prefer strict schemas for trusted JSON APIs.
:::

## String

Almost anything becomes a string via Go formatting (aligned with Zod/`String(value)` for common primitives):

```go
s := zod.Coerce.String()

s.MustParse("sup")                         // "sup"
s.MustParse(12)                            // "12"
s.MustParse(true)                          // "true"
s.MustParse(nil)                           // "null"
s.MustParse(math.NaN())                    // "NaN"
s.MustParse(math.Inf(1))                   // "Infinity"
s.MustParse(big.NewInt(15))                // "15"
s.MustParse(map[string]any{"hello": "!"}) // "map[hello:!]"
s.MustParse([]any{"a", "b"})               // "[a b]"
```

Checks still apply after coercion:

```go
id := zod.Coerce.String().Min(1)
id.MustParse(42) // "42"
```

## Number

| Input | Result |
|-------|--------|
| numeric Go types | `float64` |
| `"12.5"` | `12.5` |
| `""` | `0` |
| `true` / `false` | `1` / `0` |
| `*big.Int` | converted float |
| non-numeric string | fail |

```go
n := zod.Coerce.Number().Gte(0)

n.MustParse("3.14")
n.MustParse("")
n.MustParse(true)

res := n.SafeParse("nope")
// invalid_type expected number

// NaN/Inf still rejected after successful numeric parse
_ = n.SafeParse("NaN")
```

## Bool

See the full table on [Bool](/api/bool). Summary:

```go
b := zod.Coerce.Bool()
b.MustParse("true")
b.MustParse("0")
b.MustParse(1)
_ = b.SafeParse("yes") // fail
```

## Time

Strings: `RFC3339Nano` / `RFC3339`. Numbers: Unix **milliseconds**.

```go
t := zod.Coerce.Time()
t.MustParse("2022-11-05T00:00:00Z")
t.MustParse(float64(1667606400000))
_ = t.SafeParse("yesterday") // fail
```

## Query / form use cases

### URL query parameters

```go
// ?page=2&limit=20&active=true
querySchema := zod.Object(zod.Shape{
    "page":   zod.Coerce.Number().Int().Gte(1),
    "limit":  zod.Coerce.Number().Int().Gte(1).Lte(100),
    "active": zod.Optional(zod.Coerce.Bool()),
    "q":      zod.Optional(zod.Coerce.String().Max(200)),
})

input := map[string]any{
    "page":   "2",
    "limit":  "20",
    "active": "true",
    "q":      "go-zod",
}
out, err := querySchema.Parse(input)
// page/limit are float64; active is bool
```

:::info Defaults
Wrap coerced leaves with `zod.Default` / `zod.Prefault` when query keys may be absent — see [Default, Prefault & Catch](/api/defaults).
:::

### HTML form posts

```go
form := zod.Object(zod.Shape{
    "email":    zod.Coerce.String().Email(),
    "age":      zod.Coerce.Number().Int().Gte(0).Lt(150),
    "subscribe": zod.Coerce.Bool(), // "on"/"true"/"1" depending on your binder
    "birthday": zod.Optional(zod.Coerce.Time()),
})
```

If your form binder only emits strings, coerce is essential. If it already parses types, use strict schemas instead.

### Gin query binding sketch

```go
// With github.com/iKunalChhabra/go-zod/zgin — see Integrations → Gin
r.GET("/search", zgin.ValidateQuery(querySchema), func(c *gin.Context) {
    data := zgin.GetValidated(c) // map[string]any with coerced types
})
```

## BigInt coerce

`BigInt` supports `Params{Coerce: true}` but is **not** on the `zod.Coerce` namespace (Zod’s `z.coerce` also focuses on string/number/boolean/date):

```go
zod.BigInt(zod.Params{Coerce: true}).MustParse("99")
```

## Rules of thumb

1. Coerce **then** validate — `Coerce.String().Email()` stringifies, then checks email.
2. Failed coercion falls through to the normal type error (`invalid_type`).
3. Prefer `Coerce.*` for clarity; `Params{Coerce: true}` is equivalent on primitives.
4. Object keys themselves are never coerced — only leaf schemas you mark.
