# Special types

Catch-alls and bottom types: `Any`, `Unknown`, `Never`, `Nil`/`Null`, and `Nan`.

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

`z.Unknown()` is identical to `Any` at runtime (`Def.Type` is `"unknown"`). Prefer it when you want the Zod “must narrow before use” intent in documentation/APIs.

```go
schema := z.Unknown()
schema.MustParse([]any{1, 2, 3})
```

:::tip Any vs Unknown
Runtime behavior matches Zod: both accept everything. Choose `Unknown` for readability when callers should treat the value as opaque.
:::

## Never

`z.Never()` rejects **every** input, including `nil`. Useful as an object [catchall](/api/object) to forbid extra keys (same effect as `.Strict()`).

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

`Internals().Values` contains `nil`, so null can participate in discriminant sets:

```go
_, ok := z.Null().Internals().Values[nil] // true
```

:::info Missing vs nil
`nil` is JSON null. Absent object keys use the `z.Missing` sentinel — see [Missing vs nil](/guide/missing-nil). `Nil()` does **not** accept `Missing`.
:::

## Nan

`z.Nan()` accepts only floating NaN. Regular `Number()` **rejects** NaN; use `Nan()` when you need the opposite.

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
| `Any()` | everything | — |
| `Unknown()` | everything | — |
| `Never()` | nothing | `"never"` |
| `Nil()` / `Null()` | `nil` only | `"null"` |
| `Nan()` | `NaN` only | `"nan"` |
