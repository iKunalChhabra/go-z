# Default, Prefault & Catch

Wrappers that supply a fallback when input is absent or when parsing fails.

## Default

```go
schema := z.Default(z.String(), "anonymous")

schema.Parse(z.Missing) // "anonymous" — default applied, inner NOT re-parsed
schema.Parse("hi")        // "hi"
```

**Behavior**

1. If input is `Missing`, set the default and return (skip inner).
2. Otherwise run the inner schema.
3. If the inner output is still `Missing`, apply the default again.

Sets **OptIn** (objects may omit the key) but **not OptOut** — output is always present.

:::warn Defaults are not validated
A default that would fail the inner schema still succeeds when input is `Missing`. Use [Prefault](#prefault) if the fallback must pass validation.
:::

### DefaultFunc

```go
n := 0
schema := z.DefaultFunc(z.String(), func() any {
    n++
    return fmt.Sprintf("user-%d", n)
})
schema.MustParse(z.Missing) // "user-1"
schema.MustParse(z.Missing) // "user-2"
```

The function is called **each time** the default is needed. Prefer this for mutable / unique defaults (UUIDs, timestamps).

## Prefault

```go
inner := z.Refine(z.String(), func(v any) bool {
    s, _ := v.(string)
    return len(s) > 0 && s[0] == 'c'
}, "must start with c")

ok := z.Prefault(inner, "c")
ok.Parse(z.Missing) // "c" — prefault runs through inner + refine

bad := z.Prefault(inner, "z")
bad.SafeParse(z.Missing) // fails — "z" fails refine
```

**Behavior:** if input is `Missing`, substitute the prefault value, then **always** parse through the inner schema.

### PrefaultFunc

```go
schema := z.PrefaultFunc(z.String().Min(1), func() any {
    return "fallback"
})
```

Same as `Prefault`, but the fallback is produced by a function each time.

## Catch

```go
schema := z.Catch(z.String().Email(), "nobody@example.com")

schema.Parse("not-an-email") // "nobody@example.com" — always succeeds
schema.Parse("a@b.co")       // "a@b.co"
```

On any parse failure, replaces the value with the fallback and **clears all issues**. Catch always succeeds.

Sets OptIn so Missing input can be caught (inner may fail on Missing → catch fires).

### CatchFunc

```go
schema := z.CatchFunc(z.String().Email(), func(ctx z.CatchCtx) any {
    // ctx.Issues — finalized issues that caused the failure
    // ctx.Input  — value under parse when catch fired
    return "invalid@" + fmt.Sprint(len(ctx.Issues)) + ".example"
})
```

`CatchCtx` fields:

| Field | Type | Meaning |
|---|---|---|
| `Issues` | `[]Issue` | Finalized issues from the failed inner parse |
| `Input` | `any` | Payload value when catch ran |

## Typed variants (`*Of`)

The plain constructors operate on `any`. When your inner schema is a typed `Schema[T]`, use the `*Of` variants to keep the wrapper typed end-to-end — the fallback must be a `T`, and `Parse` returns `T`:

```go
schema := z.DefaultOf[int](z.Int(), 8080)
port, err := schema.Parse(z.Missing) // port is int — no type assertion

required := z.DefaultOf[string](z.String().Optional().NonOptional(), "fallback")
name, err := required.Parse(z.Missing) // name is string
```

| Typed | Untyped | Fallback shape |
|---|---|---|
| `DefaultOf[T](inner, defVal T)` | `Default` | value |
| `DefaultFuncOf[T](inner, fn func() T)` | `DefaultFunc` | func |
| `PrefaultOf[T](inner, defVal T)` | `Prefault` | value, validated |
| `PrefaultFuncOf[T](inner, fn func() T)` | `PrefaultFunc` | func, validated |
| `CatchOf[T](inner, fallback T)` | `Catch` | value |
| `CatchFuncOf[T](inner, fn func(CatchCtx) T)` | `CatchFunc` | func |

Semantics (Missing vs failure, validation of the fallback) are identical to the untyped versions above.

## Default vs Prefault vs Catch

| | Trigger | Validates fallback? | Can still fail? |
|---|---|---|---|
| `Default` | Missing (in or out) | No | Yes (present bad input) |
| `Prefault` | Missing input | Yes (through inner) | Yes |
| `Catch` | Any issues | No (replacement) | No — always succeeds |

```go
// Object field patterns
z.Object(z.Shape{
    "tags":    z.Default(z.Array(z.String()), []any{}),
    "role":    z.Prefault(z.Enum("user", "admin"), "user"),
    "avatar":  z.Catch(z.String().URL(), ""),
})
```

## Signatures

```go
func Default(inner AnySchemaLike, defVal any) *DefaultSchema[any]
func DefaultFunc(inner AnySchemaLike, fn func() any) *DefaultSchema[any]
func Prefault(inner AnySchemaLike, defVal any) *PrefaultSchema[any]
func PrefaultFunc(inner AnySchemaLike, fn func() any) *PrefaultSchema[any]
func Catch(inner AnySchemaLike, fallback any) *CatchSchema[any]
func CatchFunc(inner AnySchemaLike, fn func(CatchCtx) any) *CatchSchema[any]
```

All expose `.Unwrap()` to recover the inner schema.

:::tip Chaining
Outer defaults win for Missing input:

```go
z.Default(z.Default(z.String(), "inner"), "outer")
// Missing → "outer"
```
:::
