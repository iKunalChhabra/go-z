# Migration from TypeScript Zod

go-zod ports Zod v4’s architecture and issue taxonomy. The biggest differences come from **Go’s type system** and the **JSON-first** data model.

## Quick mapping

| TypeScript Zod | go-zod |
|---|---|
| `import { z } from "zod"` | `import "github.com/iKunalChhabra/go-zod"` as `zod` |
| `z.string()` | `zod.String()` |
| `z.number()` / `z.int()` | `zod.Number()` / `zod.Int()` |
| `z.boolean()` | `zod.Bool()` |
| `z.date()` | `zod.Time()` |
| `z.bigint()` | `zod.BigInt()` |
| `z.literal("x")` | `zod.Literal("x")` |
| `z.enum(["a","b"])` | `zod.Enum("a", "b")` |
| `z.object({ name: z.string() })` | `zod.Object(zod.Shape{"name": zod.String()})` |
| `z.array(z.string())` | `zod.Array(zod.String())` |
| `z.tuple([a,b])` | `zod.Tuple(a, b)` |
| `z.record(z.string(), z.number())` | `zod.Record(zod.String(), zod.Number())` |
| `z.union([a,b])` | `zod.Union([]zod.AnySchemaLike{a,b})` or `zod.UnionOf(a,b)` |
| `z.discriminatedUnion("t", […])` | `zod.DiscriminatedUnion("t", []zod.AnySchemaLike{…})` |
| `z.intersection(a,b)` | `zod.Intersection(a,b)` |
| `z.lazy(() => …)` | `zod.Lazy(func() zod.AnySchemaLike { … })` |
| `z.string().optional()` | `zod.Optional(zod.String())` |
| `z.string().nullable()` | `zod.Nullable(zod.String())` |
| `z.string().nullish()` | `zod.Nullish(zod.String())` |
| `z.string().default("x")` | `zod.Default(zod.String(), "x")` |
| `z.string().catch("x")` | `zod.Catch(zod.String(), "x")` |
| `z.string().prefault("x")` | `zod.Prefault(zod.String(), "x")` |
| `a.pipe(b)` | `zod.Pipe(a, b)` |
| `.transform(fn)` | `zod.Transform` / `zod.TransformTo[T]` |
| `z.preprocess(fn, s)` | `zod.Preprocess(fn, s)` |
| `.refine(fn)` | `zod.Refine(schema, fn, …)` |
| `.superRefine(fn)` | `zod.SuperRefine(schema, fn)` |
| `z.custom<T>(fn)` | `zod.Custom(fn)` |
| `z.coerce.string()` | `zod.Coerce.String()` |
| `schema.parse(data)` | `schema.Parse(data)` → `(T, error)` |
| `schema.safeParse(data)` | `schema.SafeParse(data)` |
| `z.infer<typeof schema>` | see below |
| `flattenError` / `treeifyError` / `prettifyError` | `zod.Flatten` / `Treeify` / `Prettify` |

## Wrappers are package functions

```ts
// TypeScript
z.string().min(5).optional().default("x")
```

```go
// Go — wrap outward
zod.Default(zod.Optional(zod.String().Min(5)), "x")
```

Go methods cannot introduce new type parameters, so `Optional`, `Default`, `Pipe`, `Transform`, `ToStruct`, etc. are top-level functions.

## Objects produce maps

```ts
type User = z.infer<typeof UserSchema> // { name: string, … }
```

```go
data, err := userSchema.Parse(input) // map[string]any
user, err := zod.ToStruct[User](userSchema).Parse(input)
```

:::info z.infer limitations
There is no `z.infer` equivalent that derives a Go struct from a schema at compile time. Define your struct yourself and bind with `ToStruct[T]`, or stay on `map[string]any`.
:::

Generics give you `Schema[T]` at the edges (`String` → `string`, `Object` → `map[string]any`, `ToStruct[User]` → `User`), but nested object inference is not automatic.

## Missing vs undefined / null

| Zod / JSON | go-zod |
|---|---|
| `undefined` / omitted key | `zod.Missing` |
| `null` | `nil` |

`Optional` ≠ `Nullable`. See [Missing vs nil](#/guide/missing-nil).

## Errors

```ts
try { schema.parse(x) } catch (e) {
  if (e instanceof z.ZodError) console.log(e.issues)
}
```

```go
_, err := schema.Parse(x)
if err != nil {
    zerr := err.(*zod.ZodError)
    _ = zerr.Issues // same codes / JSON fields as Zod v4
}
```

`safeParse` → `SafeParse` with `Success` / `Data` / `Error`.

## Not in v0

| Zod feature | Status |
|---|---|
| Codecs / `.encode()` / `.decode()` | **Not in v0** — use `Pipe` / `Transform` |
| JSON Schema export | Roadmap |
| `z.function` / `z.promise` | Out of scope |
| Template literals | Roadmap |
| Async `parseAsync` | Unneeded — Go is sync |
| Mini functional API | Not ported |
| All 50+ locales | `en es fr de ja pt zh` shipped |

## Gin instead of HTTP adapters

Zod doesn’t ship a framework binder. go-zod provides `github.com/iKunalChhabra/go-zod/zgin` — see [Gin](#/integrations/gin).

## Mental model checklist

1. Build schemas once; they are immutable and concurrency-safe.
2. Prefer `Object` + `ToStruct` over trying to mirror every TS inferred type.
3. Use `DiscriminatedUnion` for tagged variants.
4. Use `Lazy` for recursive trees (assign to `var` first).
5. Expect richer failure-path cost than tag validators — you get Zod-shaped issues.
