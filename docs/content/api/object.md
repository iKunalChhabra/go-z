# Object

`zod.Object(shape)` ports `z.object(...)`. Output is `map[string]any` (JSON object model). Default unknown-key mode is **strip**.

```go
user := zod.Object(zod.Shape{
    "name":  zod.String().Min(1),
    "email": zod.String().Email(),
    "age":   zod.Optional(zod.Int().Gte(0)),
})

out, err := user.Parse(map[string]any{
    "name":  "Ada",
    "email": "ada@example.com",
    "age":   36,
})
```

## Shape

`zod.Shape` is `map[string]AnySchemaLike`. Keys are validated in sorted order for stable issue ordering.

```go
schema := zod.Object(zod.Shape{
    "id":   zod.String().UUID(),
    "tags": zod.Array(zod.String()),
})

schema.Shape() // defensive copy of the shape map
```

Non-map inputs fail with `Expected: "object"`. Typed Go maps with string keys are accepted via reflection.

## Unknown keys: strip / strict / loose / catchall

| Mode | Method | Behavior |
|------|--------|----------|
| Strip (default) | `Strip()` | Drop unknown keys silently |
| Strict | `Strict()` | `unrecognized_keys` issue |
| Loose | `Loose()` / `Passthrough()` | Keep unknown keys as-is |
| Catchall | `Catchall(schema)` | Validate extras with `schema` |

```go
base := zod.Object(zod.Shape{"points": zod.String()})

// Strip (default)
got := base.MustParse(map[string]any{"points": "2314", "unknown": "asdf"})
// got == map[string]any{"points": "2314"}

// Strict
res := base.Strict().SafeParse(map[string]any{"points": "2314", "unknown": "asdf"})
// Code: unrecognized_keys
// Keys: ["unknown"]

// Loose
got = base.Loose().MustParse(map[string]any{"points": "2314", "unknown": "asdf"})
// got["unknown"] == "asdf"

// Catchall
schema := zod.Object(zod.Shape{"name": zod.String()}).Catchall(zod.String())
schema.MustParse(map[string]any{"name": "Foo", "extra": "ok"})
res = schema.SafeParse(map[string]any{"name": "Foo", "bad": 1})
// path: ["bad"], invalid_type string

// Catchall(Never) ≡ Strict
_ = zod.Object(zod.Shape{"id": zod.String()}).Catchall(zod.Never())
```

:::tip Catchall overrides Strict
`.Strict().Catchall(String())` validates extras as strings — catchall wins.
:::

## Path prefixes

Field issues are scoped under the property name:

```go
schema := zod.Object(zod.Shape{
    "user": zod.Object(zod.Shape{
        "email": zod.String().Email(),
    }),
})

res := schema.SafeParse(map[string]any{
    "user": map[string]any{"email": "nope"},
})
// Issues[0].Path == []any{"user", "email"}
```

## Optionality & OptIn

Wrap fields with `zod.Optional(...)` so absent keys are skipped (`Internals.OptIn`). Present-but-wrong values still error. `nil` is **not** optional — use `Nullable` / `Nullish`.

```go
schema := zod.Object(zod.Shape{
    "name": zod.String(),
    "bio":  zod.Optional(zod.String()),
})

schema.MustParse(map[string]any{"name": "Ada"})
// bio omitted from output

res := schema.SafeParse(map[string]any{})
// path ["name"] — required field missing
```

## Pick / Omit / Partial / Required

```go
full := zod.Object(zod.Shape{
    "a": zod.String(),
    "b": zod.Number(),
    "c": zod.Bool(),
})

full.Pick("a", "c")  // shape {a, c}
full.Omit("b")       // shape {a, c}

// All fields optional
full.Partial()
// Only "b" optional
full.Partial("b")

// Force presence (NonOptional wrapper)
full.Partial().Required()
full.Partial().Required("a")
```

## Extend / Merge / Keyof

```go
base := zod.Object(zod.Shape{"id": zod.String()})
extended := base.Extend(zod.Shape{"name": zod.String()})
// {id, name} — incoming keys win on conflict

a := zod.Object(zod.Shape{"a": zod.String()}).Strict()
b := zod.Object(zod.Shape{"b": zod.Number()}).Loose()
merged := a.Merge(b)
// shape {a,b}; adopts b's loose mode + catchall

keys := base.Extend(zod.Shape{"name": zod.String()}).Keyof()
// Enum of property names: "id" | "name"
keys.MustParse("id")
_ = keys.SafeParse("nope")
```

## Empty object

```go
empty := zod.Object(zod.Shape{})
empty.MustParse(map[string]any{})
empty.MustParse(map[string]any{"x": 1}) // strip → {}
```

## Custom messages

```go
schema := zod.Object(zod.Shape{"n": zod.Number()}, "object required")
res := schema.SafeParse("x")
// Message: "object required"

strict := zod.Object(zod.Shape{"n": zod.Number()}).Strict("no extras")
res = strict.SafeParse(map[string]any{"n": 1, "x": 2})
// Message: "no extras"
```

## API surface

```go
type Shape map[string]AnySchemaLike
func Object(shape Shape, params ...any) *ObjectSchema

func (s *ObjectSchema) Strict / Loose / Passthrough / Strip / Catchall(...)
func (s *ObjectSchema) Shape() Shape
func (s *ObjectSchema) Pick / Omit(keys ...string) *ObjectSchema
func (s *ObjectSchema) Extend(shape Shape) *ObjectSchema
func (s *ObjectSchema) Merge(other *ObjectSchema) *ObjectSchema
func (s *ObjectSchema) Partial / Required(keys ...string) *ObjectSchema
func (s *ObjectSchema) Keyof() *EnumSchema
```
