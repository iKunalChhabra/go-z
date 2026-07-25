# Time

`zod.Time()` ports Zod’s `z.date()`. The Go type is `time.Time` (not a calendar-date string — for those, see [`ISODate` / `ISODateTime`](/api/string-formats)).

```go
import "time"

schema := zod.Time()
now := time.Now().UTC()

got, err := schema.Parse(now)
// got.Equal(now)

got, err = schema.Parse(&now) // *time.Time also accepted
```

## Basics

Without coerce, only `time.Time` and non-nil `*time.Time` succeed:

```go
res := zod.Time().SafeParse("not-a-date")
// Message: "Invalid input: expected date, received string"
// Expected: "date"
```

## Min / Max

Inclusive bounds. Failed issues use **`Origin: "date"`** and store bounds as **Unix milliseconds** (Zod’s `Date.getTime()`).

```go
benchmark := time.Date(2022, 11, 5, 0, 0, 0, 0, time.UTC)
before := time.Date(2022, 11, 4, 0, 0, 0, 0, time.UTC)
after := time.Date(2022, 11, 6, 0, 0, 0, 0, time.UTC)

minCheck := zod.Time().Min(benchmark)
maxCheck := zod.Time().Max(benchmark)

minCheck.MustParse(benchmark)
minCheck.MustParse(after)
maxCheck.MustParse(benchmark)
maxCheck.MustParse(before)

res := minCheck.SafeParse(before)
// Code: too_small
// Origin: "date"
// Minimum: 1667606400000  (float64 ms)
// Message: "Too small: expected date to be >=1667606400000"

res = maxCheck.SafeParse(after)
// Code: too_big
// Origin: "date"
// Maximum: 1667606400000
// Message: "Too big: expected date to be <=1667606400000"
```

:::tip Origin date
Always check `Issue.Origin == "date"` when rendering date-bound errors — the numeric bound is milliseconds, not a formatted timestamp.
:::

## Coercion (RFC3339)

With coerce, strings are parsed as `RFC3339Nano` then `RFC3339`. Numeric inputs are treated as **Unix milliseconds**.

```go
s := zod.Time(zod.Params{Coerce: true})
// or: zod.Coerce.Time()

got := s.MustParse("2022-11-05T00:00:00Z")
// time.Date(2022, 11, 5, 0, 0, 0, 0, time.UTC)

_ = s.SafeParse("not-rfc3339") // fail → invalid_type date

// Unix ms
got = s.MustParse(float64(1667606400000))
got = s.MustParse(int64(1667606400000))
```

## Custom messages

```go
schema := zod.Time().Min(benchmark, "too early")
res := schema.SafeParse(before)
// Message: "too early"
```

## API surface

```go
func Time(params ...any) *TimeSchema
func (s *TimeSchema) Min(min time.Time, params ...any) *TimeSchema
func (s *TimeSchema) Max(max time.Time, params ...any) *TimeSchema
```
