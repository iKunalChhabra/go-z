# BigInt

`zod.BigInt()` ports `z.bigint()`. The Go output type is `*big.Int` from `math/big`.

```go
import "math/big"

schema := zod.BigInt().Positive()

n, err := schema.Parse(big.NewInt(42))
// n.Cmp(big.NewInt(42)) == 0
```

## Basics

Accepted without coerce:

- `*big.Int` (non-nil)
- `int64` and `int` (converted to `*big.Int`)

```go
zod.BigInt().MustParse(big.NewInt(1))
zod.BigInt().MustParse(big.NewInt(0))
zod.BigInt().MustParse(big.NewInt(-1))
zod.BigInt().MustParse(int64(7))

res := zod.BigInt().SafeParse("x")
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
| `Positive()` | `Gt(0)` |
| `Negative()` | `Lt(0)` |
| `NonPositive()` | `Lte(0)` |
| `NonNegative()` | `Gte(0)` |

```go
five := big.NewInt(5)

gt := zod.BigInt().Gt(five)
gt.MustParse(big.NewInt(6))
_ = gt.SafeParse(big.NewInt(5)) // fail

gte := zod.BigInt().Gte(five)
gte.MustParse(big.NewInt(5))
gte.MustParse(big.NewInt(6))

lt := zod.BigInt().Lt(five)
lt.MustParse(big.NewInt(4))
_ = lt.SafeParse(big.NewInt(5))

pos := zod.BigInt().Positive()
pos.MustParse(big.NewInt(3))
_ = pos.SafeParse(big.NewInt(0))

neg := zod.BigInt().Negative()
neg.MustParse(big.NewInt(-2))

nn := zod.BigInt().NonNegative()
nn.MustParse(big.NewInt(0))
nn.MustParse(big.NewInt(7))

np := zod.BigInt().NonPositive()
np.MustParse(big.NewInt(0))
np.MustParse(big.NewInt(-12))
```

## MultipleOf

```go
mult := zod.BigInt().MultipleOf(big.NewInt(5))
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
s := zod.BigInt(zod.Params{Coerce: true})

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
schema := zod.BigInt().Min(big.NewInt(0), "must be non-negative")
res := schema.SafeParse(big.NewInt(-1))
// Message: "must be non-negative"
```

## API surface

```go
func BigInt(params ...any) *BigIntSchema

func (s *BigIntSchema) Gt / Gte / Lt / Lte / Min / Max(value *big.Int, params ...any) *BigIntSchema
func (s *BigIntSchema) Positive / Negative / NonPositive / NonNegative(params ...any) *BigIntSchema
func (s *BigIntSchema) MultipleOf(value *big.Int, params ...any) *BigIntSchema
```
