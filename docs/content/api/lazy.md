# Lazy & recursive schemas

Defer schema construction for recursive and mutually-recursive types — Zod’s `z.lazy`.

## Lazy

```go
var Category zod.AnySchemaLike
Category = zod.Lazy(func() zod.AnySchemaLike {
    return zod.Object(zod.Shape{
        "name":     zod.String().Min(1),
        "children": zod.Default(zod.Array(Category), []any{}),
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
var Comment zod.AnySchemaLike
Comment = zod.Lazy(func() zod.AnySchemaLike {
    return zod.Object(zod.Shape{
        "id":      zod.String().UUID(),
        "body":    zod.String().Min(1),
        "replies": zod.Default(zod.Array(Comment), []any{}),
    })
})
```

Mutual recursion:

```go
var A, B zod.AnySchemaLike
A = zod.Lazy(func() zod.AnySchemaLike {
    return zod.Object(zod.Shape{
        "type": zod.Literal("a"),
        "b":    zod.Optional(B),
    })
})
B = zod.Lazy(func() zod.AnySchemaLike {
    return zod.Object(zod.Shape{
        "type": zod.Literal("b"),
        "a":    zod.Optional(A),
    })
})
```

## Inner()

```go
lazy := zod.Lazy(func() zod.AnySchemaLike { return zod.String().Min(1) })
inner := lazy.Inner() // resolves getter if needed; returns the memoized schema
```

`Inner()` also copies OptIn / OptOut / Values / PropValues / Pattern from the inner schema onto the Lazy’s `Internals`, so containers that inspect traits (discriminated unions, objects) see the real inner metadata — Zod’s `defineLazy` getters.

:::tip DiscriminatedUnion + Lazy
Disc-union construction unwraps `*LazySchema` when collecting discriminator values. You can nest lazy object options safely.
:::

## Check clones share state

```go
base := zod.Lazy(func() zod.AnySchemaLike { return zod.String() })
cloned := base.Check(myCheck)
// base and cloned share the same memoized inner schema
```

## Signatures

```go
func Lazy(fn func() AnySchemaLike) *LazySchema

func (s *LazySchema) Inner() AnySchemaLike
func (s *LazySchema) Check(checks ...*Check) *LazySchema
```
