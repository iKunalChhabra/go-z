# Optional, Nullable & friends

Wrappers that change whether a value may be **absent** (`z.Missing`, `undefined`) or **null** (`nil`, JSON `null`).

:::info Missing vs nil
`Missing` means the key was omitted. `nil` means the key was present with JSON `null`. They are not interchangeable — see [Missing vs nil](#/guide/missing-nil).
:::

## Typed edges

Wrappers are generic over the inner schema's output type, so the type survives the wrapper:

```go
name, err:= z.String.Optional.Parse(v) // (*string, error)
age, err:= z.Int64.Default(18).Parse(v)  // (int64, error)
```

Two shapes, by output domain:

| Wrapper | Parse returns | Why |
|---|---|---|
| `Optional`, `Nullable` | `*T` | output is “a T or nothing”; nil means absent / null |
| `Default`, `Prefault`, `Catch`, `NonOptional`, `Readonly` | `T` | always produces a value |

Every wrapper has two constructors:

- **Erased** — `z.Optional(anySchemaLike)` returns `*OptionalSchema[any]`. Use inside `z.Shape{}` and other heterogeneous containers.
- **Typed** — `z.OptionalOf(z.String)` returns `*OptionalSchema[string]`. The fluent methods call these for you.

Optional and Nullable also expose `ParseAny` / `SafeParseAny`, which return the raw JSON model so `Missing` stays distinguishable from `nil`:

```go
raw, _:= z.String.Optional.ParseAny(z.Missing) // z.Missing, not nil
```

## Optional

```go
schema:= z.OptionalOf(z.String)

schema.Parse("hello")     // (*string)("hello"), nil
schema.Parse(z.Missing)   // nil, nil  — accepted
schema.SafeParse(nil)     // fails: null is not optional
```

`Optional` sets **OptIn** and **OptOut** so object fields may omit the key. On success with an absent key, the value stays `Missing` and object parsers omit it from the output map.

```go
user:= z.Object(z.Shape{
    "nickname": z.Optional(z.String),
})

user.Parse(map[string]any{}) // map[string]any{}, nil — key omitted
```

### Nested optional

Nested `Optional` wrappers still run the inner optional’s logic, then restore `Missing` when the original input was absent (`handleOptionalResult`).

### Unwrap

```go
inner:= z.String
opt:= z.Optional(inner)
opt.Unwrap // == inner
```

## Nullable

```go
schema:= z.NullableOf(z.String)

schema.Parse("hi") // (*string)("hi")
schema.Parse(nil)  // nil — accepted
schema.SafeParse(z.Missing) // fails unless inner is also optional
```

`Nullable` accepts JSON `null` (`nil`) in addition to the inner type. OptIn/OptOut are **inherited** from the inner schema (unlike `Optional`, which always sets them).

## Nullish

```go
schema:= z.String.Nullish // or z.NullishOf(z.String)
// equivalent to:
// z.Optional(z.Nullable(z.String))
```

Accepts `Missing`, `nil`, or the inner type, and returns `*OptionalSchema[T]` — both absent and null decode to a nil `*T`, so the type does not collapse into `**T`.

## NonOptional

```go
schema:= z.NonOptional(z.Optional(z.String))

schema.SafeParse(z.Missing) // fails: invalid_type, expected "nonoptional"
schema.Parse("ok")            // "ok"
```

Runs the inner schema, then rejects if the result is still `Missing`. Emits `invalid_type` with `expected: "nonoptional"`.

:::tip When to use NonOptional
Use after wrappers or object `.Partial` when a field must be present in the output even though an intermediate schema was optional.
:::

## Readonly

```go
schema:= z.Readonly(z.String)
schema.Parse("x") // same as String — no behavior change
```

Documented **no-op**, kept for API parity. Go has no useful equivalent of `Object.freeze` for `map[string]any`. Parse behavior and OptIn/OptOut match the inner schema.

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
// Type-erased — for heterogeneous containers
func Optional(inner AnySchemaLike, params...any) *OptionalSchema[any]
func Nullable(inner AnySchemaLike, params...any) *NullableSchema[any]
func Nullish(inner AnySchemaLike, params...any) *OptionalSchema[any]
func NonOptional(inner AnySchemaLike, params...any) *NonOptionalSchema[any]
func Readonly(inner AnySchemaLike, params...any) *ReadonlySchema[any]

// Typed — inner type is preserved
func OptionalOf[T any](inner Schema[T], params...any) *OptionalSchema[T]
func NullableOf[T any](inner Schema[T], params...any) *NullableSchema[T]
func NullishOf[T any](inner Schema[T], params...any) *OptionalSchema[T]
func NonOptionalOf[T any](inner Schema[T], params...any) *NonOptionalSchema[T]
func ReadonlyOf[T any](inner Schema[T], params...any) *ReadonlySchema[T]
```

Optional params follow the usual pattern: string message, `ErrorMap`, or `Params`.

:::tip Fluent or package-level
`z.String.Optional` and `z.OptionalOf(z.String)` are the same thing — the fluent method just calls the generic constructor. Reach for the erased `z.Optional(...)` when you are holding an `AnySchemaLike` and the inner type is not known statically.
:::
