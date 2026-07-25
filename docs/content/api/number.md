# Number & Int

`z.Number` ports `z.number`. Parsed values are always normalized to Go `float64`. Integer helpers (`Int`, `Int32`, …) are number schemas with an attached number-format check.

```go
schema:= z.Number.Gte(0).Lte(100)

n, err:= schema.Parse(42)
// n == 42.0 (float64)
```

## Basics

Accepted inputs (without coerce): `float64`, `float32`, and all Go integer types. Output is always `float64`.

```go
schema:= z.Number
schema.MustParse(1234)
schema.MustParse(42) // int → 42.0

res:= schema.SafeParse("12")
// invalid_type, Expected: "number"
```

### NaN and Infinity are rejected

the upstream library treats `NaN` and `±Inf` as non-numbers:

```go
import "math"

schema:= z.Number

res:= schema.SafeParse(math.NaN)
// Message: "Invalid input: expected number, received NaN"

res = schema.SafeParse(math.Inf(1))  // fail — Expected: "number"
res = schema.SafeParse(math.Inf(-1)) // fail
```

`-0` is normalized to `+0`.

:::info Finite is a no-op
`Number.Finite` returns the same schema. Inf is already rejected at the type gate.
:::

## Constructors

| Constructor | Check | Range / rule | Output type |
|-------------|-------|----------------|-------------|
| `Number` | — | finite float64 | `float64` |
| `Int` | `safeint` | integral + within `±(2^53−1)` | `float64` |
| `Int64` | type gate | Go `int64` range | `int64` |
| `Int32` | `int32` | integral, `[−2^31, 2^31−1]` | `float64` |
| `Uint32` | `uint32` | integral, `[0, 2^32−1]` | `float64` |
| `Float32` | `float32` | within float32 exact range | `float64` |
| `Float64` | `float64` | full float64 range | `float64` |

```go
n, _:= z.Int.Parse(10)       // n is float64(10)
i, _:= z.Int64.Parse(10)     // i is int64(10)

res:= z.Int.SafeParse(1.5)
// Expected: "int"
// Message: "Invalid input: expected int, received number"

z.Int32.MustParse(2147483647)
_ = z.Uint32.SafeParse(-1) // too_small
```

:::info Int vs Int64
`Int` matches JSON-number model (`float64` + safeint). Prefer `Int64` when you want a typed Go integer without `ToStruct`.
:::

Fluent equivalents on an existing number schema:

```go
z.Number.Int  // same as attaching safeint
z.Number.Safe // alias of Int in the upstream library
```

## Comparisons

| Method | Alias | Inclusive? | Issue |
|--------|-------|------------|-------|
| `Gt(n)` | — | no | `too_small` |
| `Gte(n)` | `Min(n)` | yes | `too_small` |
| `Lt(n)` | — | no | `too_big` |
| `Lte(n)` | `Max(n)` | yes | `too_big` |
| `Positive` | `Gt(0)` | — | `too_small` |
| `Negative` | `Lt(0)` | — | `too_big` |
| `NonPositive` | `Lte(0)` | — | — |
| `NonNegative` | `Gte(0)` | — | — |

```go
schema:= z.Number.Gt(5)
schema.MustParse(6)
_ = schema.SafeParse(5) // fail — exclusive

schema = z.Number.Gte(5) // same as Min(5)
schema.MustParse(5)

schema = z.Number.Lt(5)
schema.MustParse(4)
_ = schema.SafeParse(5)

pos:= z.Number.Positive
pos.MustParse(1)
_ = pos.SafeParse(0)
_ = pos.SafeParse(-1)

neg:= z.Number.Negative
neg.MustParse(-1)

nn:= z.Number.NonNegative
nn.MustParse(0)
nn.MustParse(1)
```

Issues use `Origin: "number"` (or `"int"` for integer-format type failures).

## MultipleOf / Step

```go
schema:= z.Number.MultipleOf(5)
schema.MustParse(15)
schema.MustParse(-15)
res:= schema.SafeParse(7.5)
// Code: not_multiple_of
// Origin: "number"
// Divisor: 5

// Step is a deprecated alias of MultipleOf
_ = z.Number.Step(0.1)
```

Uses a float-safe remainder (`floatSafeRemainder`) so values like `0.1` steps work more reliably than naive `%`.

## Coercion

```go
s:= z.Number(z.Params{Coerce: true})
// or: z.Coerce.Number

s.MustParse("12.5")  // 12.5
s.MustParse("")      // 0  (JS Number("") === 0)
s.MustParse(true)    // 1
s.MustParse(false)   // 0

res:= s.SafeParse("nope")
// coerce fails → still invalid_type number
```

Also accepts `*big.Int` when coerce is on. See [Coercion](/api/coerce).

## Custom messages

```go
schema:= z.Number.Min(0, "must be ≥ 0").Max(100, "must be ≤ 100")
res:= schema.SafeParse(-1)
// Message: "must be ≥ 0"
```

## API surface

```go
func Number(params...any) *NumberSchema
func Int(params...any) *NumberSchema
func Int32(params...any) *NumberSchema
func Uint32(params...any) *NumberSchema
func Float32(params...any) *NumberSchema
func Float64(params...any) *NumberSchema

func (s *NumberSchema) Gt / Gte / Lt / Lte / Min / Max(...)
func (s *NumberSchema) Positive / Negative / NonPositive / NonNegative(...)
func (s *NumberSchema) MultipleOf / Step(...)
func (s *NumberSchema) Int / Safe / Finite(...)
```
