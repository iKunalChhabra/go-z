# Special types

Catch-alls and bottom types: `Any`, `Unknown`, `Never`, `Nil`/`Null`, `Nan`, plus the absence types `Undefined`/`Void` and the recursive `JSON`.

## Any

`z.Any()` accepts every input and returns it unchanged (`schemaBase[any]`).

```go
schema := z.Any()
schema.MustParse("x")
schema.MustParse(42)
schema.MustParse(nil)
schema.MustParse(map[string]any{"a": 1})
```

Use when a field is intentionally untyped, or as a temporary placeholder.

## Unknown

`z.Unknown()` is identical to `Any` at runtime (`Def.Type` is `"unknown"`). Prefer it when you want to signal “must narrow before use” in documentation/APIs.

```go
schema := z.Unknown()
schema.MustParse([]any{1, 2, 3})
```

:::tip Any vs Unknown
Runtime behavior matches the original: both accept everything. Choose `Unknown` for readability when callers should treat the value as opaque.
:::

## Never

`z.Never()` rejects **every** input, including `nil`. Useful as an object [catchall](/api/object) to forbid extra keys (same effect as `.Strict`).

```go
res := z.Never().SafeParse("x")
// Expected: "never"
// Message: "Invalid input: expected never, received string"

_ = z.Never().SafeParse(nil) // also fails

// Forbid unknown keys:
schema := z.Object(z.Shape{"id": z.String()}).Catchall(z.Never())
```

## Nil / Null

`z.Nil()` and `z.Null()` are aliases — both accept only Go `nil` (JSON `null`). Expected type in issues is `"null"`.

```go
for _, s := range []*z.NilSchema{z.Nil(), z.Null()} {
    got, err := s.Parse(nil)
    // got == nil, err == nil

    res := s.SafeParse("x")
    // Expected: "null"
    // Message: "Invalid input: expected null, received string"
}
```

`Internals.Values` contains `nil`, so null can participate in discriminant sets:

```go
_, ok := z.Null().Internals().Values[nil] // true
```

:::info Missing vs nil
`nil` is JSON null. Absent object keys use the `z.Missing` sentinel — see [Missing vs nil](/guide/missing-nil). `Nil` does **not** accept `Missing`.
:::

## Undefined / Void

`z.Undefined()` and `z.Void()` accept **only** the `z.Missing` sentinel — the value an absent object key parses as (see [Missing vs nil](#/guide/missing-nil)). They never accept `nil`. `Void` is the looser alias for “no meaningful return value”; `Undefined` mirrors the JS name. Expected types in issues are `"undefined"` and `"void"` respectively.

```go
s := z.Undefined()
got, err := s.Parse(z.Missing) // got == nil, err == nil

_ = s.SafeParse(nil) // fail — Missing is not nil
_ = s.SafeParse("")  // fail

_ = z.Void().MustParse(z.Missing) // same contract, "void" in messages
```

These compose naturally with [Optional](#/api/optional): `Optional(T)` is effectively `Union(T, Undefined)`.

## JSON

`z.JSON()` accepts any valid JSON-shaped value: strings, numbers, booleans, `nil`, arrays, and string-keyed maps — recursively. It is implemented as a lazy union, so nested structures are validated, not just the top level.

```go
s := z.JSON()
s.MustParse(map[string]any{
    "name":  "ada",
    "tags":  []any{"admin", "owner"},
    "meta":  map[string]any{"age": 36.0, "active": true},
    "spouse": nil,
})

_ = s.SafeParse(func() {}) // fail — not JSON-shaped
```

Use it for free-form `metadata` / `payload` fields where the shape is unknowable but must stay JSON-serializable. Anything `encoding/json` cannot represent (functions, channels, maps with non-string keys) fails validation.

## Nan

`z.Nan` accepts only floating NaN. Regular `Number` **rejects** NaN; use `Nan` when you need the opposite.

```go
import "math"

s := z.Nan()
got := s.MustParse(math.NaN())
// math.IsNaN(got) == true

_ = s.SafeParse(5)       // fail
_ = s.SafeParse("John")  // fail
_ = s.SafeParse(true)    // fail
_ = s.SafeParse(nil)     // fail

res := s.SafeParse(1.0)
// Message: "Invalid input: expected NaN, received number"
```

## Quick reference

| Constructor | Accepts | `Expected` on failure |
|-------------|---------|------------------------|
| `Any` | everything | — |
| `Unknown` | everything | — |
| `Never` | nothing | `"never"` |
| `Nil` / `Null` | `nil` only | `"null"` |
| `Undefined` | `z.Missing` only | `"undefined"` |
| `Void` | `z.Missing` only | `"void"` |
| `JSON` | JSON-shaped values, recursively | per failing leaf |
| `Nan` | `NaN` only | `"nan"` |
