# Pipe, Transform & Preprocess

Compose schemas and value transforms. Because Go methods cannot introduce new type parameters, output-changing ops are **package-level functions**.

## Pipe

```go
// string → trimmed string → non-empty
trimmed := zod.Transform(zod.String(), func(v any, _ *zod.RefinementCtx) (any, error) {
    return strings.TrimSpace(v.(string)), nil
})
schema := zod.Pipe(trimmed, zod.String().Min(1))
```

Runs schema **A**, then schema **B** on A’s output. If A produces any issues, B is skipped and the payload is aborted.

| Trait | Source |
|---|---|
| OptIn | from A |
| OptOut | from B |
| Values / PropValues | copied from A |

```go
p := zod.Pipe(a, b)
p.In()  // a
p.Out() // b
```

## Transform

```go
schema := zod.Transform(zod.String(), func(v any, ctx *zod.RefinementCtx) (any, error) {
    s := v.(string)
    if s == "" {
        ctx.AddMessage("empty")
        return nil, nil
    }
    return strings.ToUpper(s), nil
})
```

Runs the inner schema, then `fn`. Failures:

- Returned `error` → `custom` issue with `err.Error()` as message.
- Issues added via `RefinementCtx` → parse fails (value not replaced).

If the inner schema already has issues, the transform is skipped (`Aborted`).

### TransformTo

Typed convenience when the output type is known:

```go
schema := zod.TransformTo[int](zod.String(), func(v any) (int, error) {
    return strconv.Atoi(v.(string))
})

n, err := schema.Parse("42") // n is int
```

Returns `Schema[Out]`. Prefer this over untyped `Transform` at API boundaries.

## Preprocess

```go
schema := zod.Preprocess(func(v any) any {
    if s, ok := v.(string); ok {
        return strings.TrimSpace(s)
    }
    return v
}, zod.String().Email())
```

Applies `fn` to the **input**, then parses with `schema`. Ports `$ZodPreprocess` (implemented as a pipe subtype in Zod). OptIn/OptOut follow the target schema.

## OverwriteSchema

In-place value rewrite that keeps the same schema “slot” (Zod’s `.overwrite()` / `$ZodCheckOverwrite`):

```go
schema := zod.OverwriteSchema(zod.String(), func(v any) any {
    return strings.TrimSpace(v.(string))
})
```

Runs as a check after the inner parse — distinct from `Overwrite` in `checks_string.go` (string helper). Named `OverwriteSchema` to avoid the clash.

:::info Transform vs OverwriteSchema vs Preprocess
- **Preprocess** — mutate input *before* type validation.
- **OverwriteSchema** — mutate value *after* successful inner parse (check).
- **Transform** — full pipe into a new output; can change type / add contextual issues.
:::

## Signatures

```go
func Pipe(a, b AnySchemaLike) *PipeSchema
func Transform(inner AnySchemaLike, fn func(any, *RefinementCtx) (any, error)) *TransformSchema
func TransformTo[Out any](inner AnySchemaLike, fn func(any) (Out, error)) Schema[Out]
func Preprocess(fn func(any) any, schema AnySchemaLike) *PreprocessSchema
func OverwriteSchema(inner AnySchemaLike, fn func(any) any) *CheckedSchema
```

:::warn encode / decode not in v0
Bidirectional codecs (`z.codec`, `.encode()`, `.decode()`) are on the roadmap, not shipped in v0. Use `Pipe` / `Transform` / `Preprocess` for one-way pipelines.
:::
