# Numbers

Every numeric constructor produces the Go type its name promises: `z.Int()` gives you an `int`, `z.Uint32()` a `uint32`, `z.Number()` a `float64`. They are all instantiations of one generic schema, `NumericSchema[T]`, so the whole fluent surface behaves the same regardless of which one you pick.

```go
schema := z.Int().Gte(0).Lte(100)

n, err := schema.Parse(42)
// n is int(42) — not float64
```

## Input vs output

Input is accepted in whatever numeric form it arrives. JSON decodes numbers to `float64`, so that is the common case, and a `float64` that holds a whole number is a perfectly good `int`:

```go
z.Int().MustParse(42.0)      // 42   (from JSON)
z.Int().MustParse(int64(42)) // 42
z.Number().MustParse(42)     // 42.0 (int widened)

res := z.Int().SafeParse("12")
// invalid_type, Expected: "number"
```

The conversion has to be exact. A value the output type cannot hold is reported as an issue — it is never silently truncated or wrapped:

```go
res := z.Int().SafeParse(3.7)
// invalid_type, Expected: "int"
// Message: "Invalid input: expected int, received number"

// Each constructor has its own output type, so each result has its own type too.
tooBig := z.Int32().SafeParse(2147483648) // too_big
negative := z.Uint32().SafeParse(-1)      // too_small
```

### NaN and Infinity are rejected

`NaN` and `±Inf` are not numbers as far as validation is concerned:

```go
import "math"

schema := z.Number()

res := schema.SafeParse(math.NaN())
// Message: "Invalid input: expected number, received NaN"

res = schema.SafeParse(math.Inf(1))  // fail — Expected: "number"
res = schema.SafeParse(math.Inf(-1)) // fail
```

`-0` is normalized to `+0`.

:::info Finite is a no-op
`Number().Finite()` returns the same schema. Inf is already rejected at the type gate.
:::

## Constructors

| Constructor | Output type | Range / rule |
|-------------|-------------|--------------|
| `Number` | `float64` | any finite float64 |
| `Float64` | `float64` | full float64 range |
| `Float32` | `float32` | within float32 range, else `too_big` / `too_small` |
| `Int` | `int` | whole number within `±(2^53−1)`, the JSON safe-integer range |
| `Int32` | `int32` | whole number in `[−2^31, 2^31−1]` |
| `Uint32` | `uint32` | whole number in `[0, 2^32−1]` |
| `Int64` | `int64` | the full Go `int64` range |

```go
n, _ := z.Int().Parse(10)   // int(10)
i, _ := z.Int64().Parse(10) // int64(10)
p, _ := z.Uint32().Parse(8080)
f, _ := z.Float32().Parse(1.5)
```

### Custom numeric types with `NumericOf`

`z.NumericOf[T]()` builds a schema over any type satisfying the `Numeric` constraint — including named types you define:

```go
type UserID uint64

id := z.NumericOf[UserID]().Gte(UserID(1))
got, err := id.Parse(UserID(42)) // got is UserID, no assertion
```

All fluent checks (`Gt`/`Gte`/`Min`/`Lt`/`Lte`/`Max`/`MultipleOf`/`Integer`/…) take and return `T`.

:::info Int vs Int64
`Int` models a JSON number, so it stops at the safe-integer range (`±2^53−1`) where `float64` can no longer represent every integer exactly. `Int64` covers the full 64-bit range and never round-trips through `float64`, which makes it the right choice for database identifiers and counters:

```go
z.Int().SafeParse(int64(9007199254740993))   // fails — outside the safe range
z.Int64().MustParse(int64(9007199254740993)) // 9007199254740993
```

Because `Int64` is not a JSON number, a non-integer input is an `invalid_type` failure rather than a format failure.
:::

### Whole numbers with a float64 output

When you want to *keep* `float64` as the output type but require a whole number, use the check instead of the constructor:

```go
z.Number().Integer() // float64 output, must be a whole number
z.Number().Safe()    // alias of Integer
```

## Comparisons

Bounds take the schema's own type, so there is no casting at the call site: `z.Int().Gte(1)` takes an `int`, `z.Int64().Gte(1)` an `int64`.

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
schema := z.Number().Gt(5)
schema.MustParse(6)
_ = schema.SafeParse(5) // fail — exclusive

schema = z.Number().Gte(5) // same as Min(5)
schema.MustParse(5)

pos := z.Int().Positive()
pos.MustParse(1)
_ = pos.SafeParse(0)
_ = pos.SafeParse(-1)

nn := z.Int().NonNegative()
nn.MustParse(0)
```

Issues use `Origin: "number"` (or `"int"` for integer type failures).

## MultipleOf / Step

```go
schema := z.Number().MultipleOf(5)
schema.MustParse(15)
schema.MustParse(-15)
res := schema.SafeParse(7.5)
// Code: not_multiple_of
// Origin: "number"
// Divisor: 5

// Step is an alias of MultipleOf
_ = z.Number().Step(0.1)
```

Uses a float-safe remainder so values like `0.1` steps work more reliably than naive `%`.

## Coercion

```go
s := z.Number(z.Params{Coerce: true})
// or: z.Coerce.Number()

s.MustParse("12.5")  // 12.5
s.MustParse("")      // 0  (JS Number("") === 0)
s.MustParse(true)    // 1
s.MustParse(false)   // 0

res := s.SafeParse("nope")
// coerce fails → still invalid_type number
```

Integer schemas parse the string as an integer first, so long numeric strings keep every digit:

```go
z.Int64(z.Params{Coerce: true}).MustParse("9007199254740993") // exact
z.Int(z.Params{Coerce: true}).SafeParse("33.7")               // fails, not truncated
```

Also accepts `*big.Int` when coerce is on. See [Coercion](/api/coerce).

## Wrappers and refinements

Wrappers keep the output type, and `Refine` receives it directly:

```go
count, _ := z.Int().Default(1).Parse(z.Missing) // int(1)
maybe, _ := z.Int().Optional().Parse(z.Missing) // (*int)(nil)

even := z.Int().Refine(func(n int) bool { return n%2 == 0 }, "must be even")
```

## Custom messages

```go
schema := z.Int().Min(0, "must be ≥ 0").Max(100, "must be ≤ 100")
res := schema.SafeParse(-1)
// Message: "must be ≥ 0"
```

## API surface

```go
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

type NumericSchema[T Numeric] struct{ /* unexported */ }

// NumberSchema and Int64Schema are aliases for the float64 and int64 forms.
type NumberSchema = NumericSchema[float64]
type Int64Schema = NumericSchema[int64]

func Number(params ...any) *NumberSchema
func Float64(params ...any) *NumericSchema[float64]
func Float32(params ...any) *NumericSchema[float32]
func Int(params ...any) *NumericSchema[int]
func Int32(params ...any) *NumericSchema[int32]
func Uint32(params ...any) *NumericSchema[uint32]
func Int64(params ...any) *Int64Schema

func (s *NumericSchema[T]) Gt(value T, params ...any) *NumericSchema[T]
func (s *NumericSchema[T]) Gte(value T, params ...any) *NumericSchema[T] // alias Min
func (s *NumericSchema[T]) Lt(value T, params ...any) *NumericSchema[T]
func (s *NumericSchema[T]) Lte(value T, params ...any) *NumericSchema[T] // alias Max
func (s *NumericSchema[T]) Positive(params ...any) *NumericSchema[T]
func (s *NumericSchema[T]) Negative(params ...any) *NumericSchema[T]
func (s *NumericSchema[T]) NonPositive(params ...any) *NumericSchema[T]
func (s *NumericSchema[T]) NonNegative(params ...any) *NumericSchema[T]
func (s *NumericSchema[T]) MultipleOf(value T, params ...any) *NumericSchema[T] // alias Step
func (s *NumericSchema[T]) Integer(params ...any) *NumericSchema[T] // alias Safe
func (s *NumericSchema[T]) Finite(params ...any) *NumericSchema[T]
func (s *NumericSchema[T]) Refine(pred func(T) bool, params ...any) *NumericSchema[T]
```

Adding another width — `Uint64`, `Int16` — is a one-line constructor on the same generic type.
