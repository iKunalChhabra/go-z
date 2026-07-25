# BigInt

`z.BigInt()` ports `z.bigint`. The Go output type is `*big.Int` from `math/big`.

```go
import "math/big"

schema := z.BigInt().Positive()

n, err := schema.Parse(big.NewInt(42))
// n.Cmp(big.NewInt(42)) == 0
```

## Basics

Accepted without coerce:

- `*big.Int` (non-nil)
- `int64` and `int` (converted to `*big.Int`)

```go
z.BigInt().MustParse(big.NewInt(1))
z.BigInt().MustParse(big.NewInt(0))
z.BigInt().MustParse(big.NewInt(-1))
z.BigInt().MustParse(int64(7))

res := z.BigInt().SafeParse("x")
// Message: "Invalid input: expected bigint, received string"
```

## Comparisons

Bound arguments are `*big.Int`. Issues use `Origin: "bigint"`.

| Method | Inclusive? |
|--------|------------|
| `Gt(n)` | no |
| `Gte(n)` / `Min(n)` | yes |
| `Lt(n)` | no |
| `Lte(n)` / `Max(n)` | yes |
| `Positive` | `Gt(0)` |
| `Negative` | `Lt(0)` |
| `NonPositive` | `Lte(0)` |
| `NonNegative` | `Gte(0)` |

```go
five := big.NewInt(5)

gt := z.BigInt().Gt(five)
gt.MustParse(big.NewInt(6))
_ = gt.SafeParse(big.NewInt(5)) // fail

gte := z.BigInt().Gte(five)
gte.MustParse(big.NewInt(5))
gte.MustParse(big.NewInt(6))

lt := z.BigInt().Lt(five)
lt.MustParse(big.NewInt(4))
_ = lt.SafeParse(big.NewInt(5))

pos := z.BigInt().Positive()
pos.MustParse(big.NewInt(3))
_ = pos.SafeParse(big.NewInt(0))

neg := z.BigInt().Negative()
neg.MustParse(big.NewInt(-2))

nn := z.BigInt().NonNegative()
nn.MustParse(big.NewInt(0))
nn.MustParse(big.NewInt(7))

np := z.BigInt().NonPositive()
np.MustParse(big.NewInt(0))
np.MustParse(big.NewInt(-12))
```

## MultipleOf

```go
mult := z.BigInt().MultipleOf(big.NewInt(5))
mult.MustParse(big.NewInt(15))
mult.MustParse(big.NewInt(-15))

res := mult.SafeParse(big.NewInt(13))
// Code: not_multiple_of
// Origin: "bigint"
```

:::warn Divisor zero
If the divisor is `0`, the multiple-of check is skipped (modulo by zero is undefined).
:::

## Coercion

```go
s := z.BigInt(z.Params{Coerce: true})

got := s.MustParse("5")
// got.Cmp(big.NewInt(5)) == 0

got = s.MustParse(true)  // 1
got = s.MustParse(false) // 0
got = s.MustParse("")    // 0

// Whole float64 values coerce; fractional / NaN / Inf do not
got = s.MustParse(float64(3))
_ = s.SafeParse(3.14) // fail
```

## Custom messages

```go
schema := z.BigInt().Min(big.NewInt(0), "must be non-negative")
res := schema.SafeParse(big.NewInt(-1))
// Message: "must be non-negative"
```

## API surface

```go
func BigInt(params ...any) *BigIntSchema

func (s *BigIntSchema) Gt(value *big.Int, params ...any) *BigIntSchema
func (s *BigIntSchema) Gte(value *big.Int, params ...any) *BigIntSchema // alias Min
func (s *BigIntSchema) Lt(value *big.Int, params ...any) *BigIntSchema
func (s *BigIntSchema) Lte(value *big.Int, params ...any) *BigIntSchema // alias Max
func (s *BigIntSchema) Positive(params ...any) *BigIntSchema
func (s *BigIntSchema) Negative(params ...any) *BigIntSchema
func (s *BigIntSchema) NonPositive(params ...any) *BigIntSchema
func (s *BigIntSchema) NonNegative(params ...any) *BigIntSchema
func (s *BigIntSchema) MultipleOf(value *big.Int, params ...any) *BigIntSchema
```
