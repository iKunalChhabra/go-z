# Optional, Nullable & friends

Wrappers that change whether a value may be **absent** (`zod.Missing`, Zod’s `undefined`) or **null** (`nil`, JSON `null`).

:::info Missing vs nil
`Missing` means the key was omitted. `nil` means the key was present with JSON `null`. They are not interchangeable — see [Missing vs nil](#/guide/missing-nil).
:::

## Optional

```go
schema := zod.Optional(zod.String())

schema.Parse("hello")       // "hello", nil
schema.Parse(zod.Missing)   // Missing, nil  — accepted
schema.SafeParse(nil)       // fails: null is not optional
```

`Optional` sets **OptIn** and **OptOut** so object fields may omit the key. On success with an absent key, the value stays `Missing` and object parsers omit it from the output map.

```go
user := zod.Object(zod.Shape{
    "nickname": zod.Optional(zod.String()),
})

user.Parse(map[string]any{}) // map[string]any{}, nil — key omitted
```

### Nested optional

Nested `Optional` wrappers still run the inner optional’s logic, then restore `Missing` when the original input was absent (Zod’s `handleOptionalResult`).

### Unwrap

```go
inner := zod.String()
opt := zod.Optional(inner)
opt.Unwrap() // == inner
```

## Nullable

```go
schema := zod.Nullable(zod.String())

schema.Parse("hi") // "hi"
schema.Parse(nil)  // nil — accepted
schema.SafeParse(zod.Missing) // fails unless inner is also optional
```

`Nullable` accepts JSON `null` (`nil`) in addition to the inner type. OptIn/OptOut are **inherited** from the inner schema (unlike `Optional`, which always sets them).

## Nullish

```go
schema := zod.Nullish(zod.String())
// equivalent to:
// zod.Optional(zod.Nullable(zod.String()))
```

Accepts `Missing`, `nil`, or the inner type. Returns `*OptionalSchema`.

## NonOptional

```go
schema := zod.NonOptional(zod.Optional(zod.String()))

schema.SafeParse(zod.Missing) // fails: invalid_type, expected "nonoptional"
schema.Parse("ok")            // "ok"
```

Runs the inner schema, then rejects if the result is still `Missing`. Emits `invalid_type` with `expected: "nonoptional"`.

:::tip When to use NonOptional
Use after wrappers or object `.Partial()` when a field must be present in the output even though an intermediate schema was optional.
:::

## Readonly

```go
schema := zod.Readonly(zod.String())
schema.Parse("x") // same as String() — no behavior change
```

Documented **no-op** for API parity with Zod’s `$ZodReadonly`. Go has no useful equivalent of `Object.freeze` for `map[string]any`. Parse behavior and OptIn/OptOut match the inner schema.

## Comparison

| Wrapper | Accepts Missing | Accepts nil | OptIn | OptOut |
|---|---|---|---|---|
| `Optional` | yes | no | true | true |
| `Nullable` | (from inner) | yes | from inner | from inner |
| `Nullish` | yes | yes | true | true |
| `NonOptional` | no (after parse) | (from inner) | cleared for Missing | (propagated meta) |
| `Readonly` | (from inner) | (from inner) | from inner | from inner |

`NonOptional` clears `Missing` from the values set and rejects Missing after the inner run.

## Signatures

```go
func Optional(inner AnySchemaLike, params ...any) *OptionalSchema
func Nullable(inner AnySchemaLike, params ...any) *NullableSchema
func Nullish(inner AnySchemaLike, params ...any) *OptionalSchema
func NonOptional(inner AnySchemaLike, params ...any) *NonOptionalSchema
func Readonly(inner AnySchemaLike, params ...any) *ReadonlySchema
```

Optional params follow the usual pattern: string message, `ErrorMap`, or `Params`.

:::warn Package functions, not methods
In TypeScript Zod you write `z.string().optional()`. In go-zod use **package-level** wrappers: `zod.Optional(zod.String())`. Go methods cannot return a new type parameter for wrappers that change the output shape.
:::
