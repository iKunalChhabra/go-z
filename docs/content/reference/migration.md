# Migration from TypeScript Zod

go-zod ports Zod v4’s architecture and issue taxonomy. The biggest differences come from **Go’s type system** and the **JSON-first** data model.

## Quick mapping

| TypeScript Zod | go-zod |
|---|---|
| `import { z } from "zod"` | `import z "github.com/iKunalChhabra/go-zod"` |
| `z.string()` | `z.String()` |
| `z.number()` / `z.int()` | `z.Number()` / `z.Int()` |
| `z.boolean()` | `z.Bool()` |
| `z.date()` | `z.Time()` |
| `z.bigint()` | `z.BigInt()` |
| `z.literal("x")` | `z.Literal("x")` |
| `z.enum(["a","b"])` | `z.Enum("a", "b")` |
| `z.object({ name: z.string() })` | `z.Object(z.Shape{"name": z.String()})` |
| `z.array(z.string())` | `z.Array(z.String())` |
| `z.tuple([a,b])` | `z.Tuple(a, b)` |
| `z.record(z.string(), z.number())` | `z.Record(z.String(), z.Number())` |
| `z.union([a,b])` | `z.Union([]z.AnySchemaLike{a,b})` or `z.UnionOf(a,b)` |
| `z.discriminatedUnion("t", […])` | `z.DiscriminatedUnion("t", []z.AnySchemaLike{…})` |
| `z.intersection(a,b)` | `z.Intersection(a,b)` |
| `z.lazy(() => …)` | `z.Lazy(func() z.AnySchemaLike { … })` |
| `z.string().optional()` | `z.Optional(z.String())` |
| `z.string().nullable()` | `z.Nullable(z.String())` |
| `z.string().nullish()` | `z.Nullish(z.String())` |
| `z.string().default("x")` | `z.Default(z.String(), "x")` |
| `z.string().catch("x")` | `z.Catch(z.String(), "x")` |
| `z.string().prefault("x")` | `z.Prefault(z.String(), "x")` |
| `a.pipe(b)` | `z.Pipe(a, b)` |
| `.transform(fn)` | `z.Transform` / `z.TransformTo[T]` |
| `z.preprocess(fn, s)` | `z.Preprocess(fn, s)` |
| `.refine(fn)` | `z.Refine(schema, fn, …)` |
| `.superRefine(fn)` | `z.SuperRefine(schema, fn)` |
| `z.custom<T>(fn)` | `z.Custom(fn)` |
| `z.coerce.string()` | `z.Coerce.String()` |
| `schema.parse(data)` | `schema.Parse(data)` → `(T, error)` |
| `schema.safeParse(data)` | `schema.SafeParse(data)` |
| `z.infer<typeof schema>` | see below |
| `flattenError` / `treeifyError` / `prettifyError` | `z.Flatten` / `Treeify` / `Prettify` |

## Wrappers are package functions

```ts
// TypeScript
z.string().min(5).optional().default("x")
```

```go
// Go — wrap outward
z.Default(z.Optional(z.String().Min(5)), "x")
```

Go methods cannot introduce new type parameters, so `Optional`, `Default`, `Pipe`, `Transform`, `ToStruct`, etc. are top-level functions.

## Objects produce maps

```ts
type User = z.infer<typeof UserSchema> // { name: string, … }
```

```go
data, err := userSchema.Parse(input) // map[string]any
user, err := z.ToStruct[User](userSchema).Parse(input)
```

:::info z.infer limitations
There is no `z.infer` equivalent that derives a Go struct from a schema at compile time. Define your struct yourself and bind with `ToStruct[T]`, or stay on `map[string]any`.
:::

Generics give you `Schema[T]` at the edges (`String` → `string`, `Object` → `map[string]any`, `ToStruct[User]` → `User`), but nested object inference is not automatic.

## Missing vs undefined / null

| Zod / JSON | go-zod |
|---|---|
| `undefined` / omitted key | `z.Missing` |
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
    zerr := err.(*z.ZodError)
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
