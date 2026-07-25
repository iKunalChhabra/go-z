# Discriminated union

Object unions keyed by a shared literal field — Zod’s `z.discriminatedUnion`.

## DiscriminatedUnion

```go
schema := zod.DiscriminatedUnion("role", []zod.AnySchemaLike{
    zod.Object(zod.Shape{
        "role":  zod.Literal("admin"),
        "perms": zod.Array(zod.String()),
    }),
    zod.Object(zod.Shape{
        "role":    zod.Literal("guest"),
        "session": zod.String().UUID(),
    }),
})

schema.Parse(map[string]any{
    "role":  "admin",
    "perms": []any{"read", "write"},
})
```

### Fast path

At construction, go-zod builds a **discriminator map** from each option’s `PropValues` (literal / enum value sets on the discriminator key):

```
discMap["admin"] → admin object schema
discMap["guest"] → guest object schema
```

At parse time:

1. Input must be an object (`map[string]any`); otherwise `invalid_type` expected `"object"`.
2. Read `obj[discriminator]` (absent key → `Missing`).
3. If the value is in the map → run **only that option** (no linear try-all).
4. Otherwise → discriminator error (or union fallback).

:::tip Performance
Discriminated unions avoid trying every branch. Prefer them over plain `Union` whenever variants share a stable tag field.
:::

## Discriminator errors

Unknown discriminator values produce `invalid_union` with:

| Field | Content |
|---|---|
| `Code` | `"invalid_union"` |
| `Discriminator` | the key name |
| `Values` | known discriminator literals |
| `Path` | `[]any{discriminator}` |
| `Errors` | empty `[][]Issue{}` |

`EnLocale` formats this as:

```text
Invalid discriminator value. Expected 'admin' | 'guest'
```

```go
res := schema.SafeParse(map[string]any{"role": "superuser"})
// res.Error.Issues[0].Message ≈ "Invalid discriminator value. Expected 'admin' | 'guest'"
```

## Construction panics

Invalid option lists panic at build time (same as Zod):

| Condition | Panic message pattern |
|---|---|
| Option has no values for the discriminator | `Invalid discriminated union option at index "…"` |
| Duplicate discriminator value across options | `Duplicate discriminator value "…"` |

Options may be wrapped in `Lazy` — the builder unwraps lazy schemas when collecting PropValues.

## DiscUnionParams

```go
schema := zod.DiscriminatedUnion("type", options, zod.DiscUnionParams{
    Error:         myErrorMap,
    UnionFallback: true, // on unknown tag, try plain Union instead of disc error
})
```

| Field | Default | Meaning |
|---|---|---|
| `Error` | nil | Schema-level error map |
| `UnionFallback` | false | If true, unknown discriminators fall through to [Union](#/api/union) try-all |

You can also pass a string message, `ErrorMap`, or `Params` as optional trailing args.

## PropValues aggregation

The disc-union’s internals merge `PropValues` across options so nested containers / further disc unions can inspect them. Object fields with literal `Values` populate PropValues automatically at object build time.

## Signatures

```go
func DiscriminatedUnion(discriminator string, options []AnySchemaLike, params ...any) *DiscriminatedUnionSchema

type DiscUnionParams struct {
    Error         ErrorMap
    UnionFallback bool
}

type DiscriminatedUnionSchema struct {
    Discriminator string
    Options       []AnySchemaLike
    // ...
}
func (s *DiscriminatedUnionSchema) Check(checks ...*Check) *DiscriminatedUnionSchema
```
