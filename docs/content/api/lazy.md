# Lazy & recursive schemas

Defer schema construction for recursive and mutually-recursive types — `z.lazy`.

## Lazy

```go
var Category z.AnySchemaLike
Category = z.Lazy(func z.AnySchemaLike {
    return z.Object(z.Shape{
        "name":     z.String.Min(1),
        "children": z.Default(z.Array(Category), []any{}),
    })
})

Category.Parse(map[string]any{
    "name": "root",
    "children": []any{
        map[string]any{"name": "child"},
    },
})
```

The getter runs **once** (`sync.Once`) and is memoized. Subsequent parses reuse the same inner schema instance — important for recursive identity and for `Check` clones that share `lazyState`.

:::warn Nil getter / nil return
`Lazy(nil)` panics. A getter that returns `nil` panics on first resolve: `"Lazy getter returned nil schema"`.
:::

## Recursive trees

Typical pattern: declare a `var`, assign `Lazy`, close over the var inside the getter.

```go
var Comment z.AnySchemaLike
Comment = z.Lazy(func z.AnySchemaLike {
    return z.Object(z.Shape{
        "id":      z.String.UUID,
        "body":    z.String.Min(1),
        "replies": z.Default(z.Array(Comment), []any{}),
    })
})
```

Mutual recursion:

```go
var A, B z.AnySchemaLike
A = z.Lazy(func z.AnySchemaLike {
    return z.Object(z.Shape{
        "type": z.Literal("a"),
        "b":    z.Optional(B),
    })
})
B = z.Lazy(func z.AnySchemaLike {
    return z.Object(z.Shape{
        "type": z.Literal("b"),
        "a":    z.Optional(A),
    })
})
```

## Inner

```go
lazy:= z.Lazy(func z.AnySchemaLike { return z.String.Min(1) })
inner:= lazy.Inner // resolves getter if needed; returns the memoized schema
```

`Inner` also copies OptIn / OptOut / Values / PropValues / Pattern from the inner schema onto the Lazy’s `Internals`, so containers that inspect traits (discriminated unions, objects) see the real inner metadata — `defineLazy` getters.

:::tip DiscriminatedUnion + Lazy
Disc-union construction unwraps `*LazySchema` when collecting discriminator values. You can nest lazy object options safely.
:::

## Check clones share state

```go
base:= z.Lazy(func z.AnySchemaLike { return z.String })
cloned:= base.Check(myCheck)
// base and cloned share the same memoized inner schema
```

## Signatures

```go
func Lazy(fn func AnySchemaLike) *LazySchema

func (s *LazySchema) Inner AnySchemaLike
func (s *LazySchema) Check(checks...*Check) *LazySchema
```
