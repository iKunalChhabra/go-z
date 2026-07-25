# Xor (exclusive union)

`Xor` accepts a value that matches **exactly one** option. A value matching two or more options fails.

```go
schema:= z.XorOf(z.String.Min(1), z.Number.Gte(0))

schema.Parse("hi")   // "hi"
schema.Parse(3.0)    // 3.0
schema.SafeParse(-1) // fails: no option matched
```

Compare with [Union](#/api/union), which returns the **first** match and does not care whether later options would also match.

## When exclusivity matters

Overlapping options are the interesting case:

```go
overlapping:= z.XorOf(z.String, z.String.Min(1))

overlapping.SafeParse("hello") // fails — both options match
overlapping.Parse("")          // "" — only the first option matches
```

With `z.Union` the same input would succeed on the first option. Reach for `Xor` when overlap indicates an ambiguous contract you would rather reject than silently resolve.

:::info A go-z addition
The TypeScript original has no `xor`. It exists here for contracts where "matches more than one shape" is a bug. Everything else about it — issue codes, error maps, wrappers — behaves like `Union`.
:::

## Errors

Zero matches produce an `invalid_union` issue carrying the per-option issue lists, exactly like `Union`:

```go
res:= z.XorOf(z.String, z.Number).SafeParse(true)
res.Error.Issues[0].Code      // invalid_union
res.Error.Issues[0].Errors    // [][]Issue — one slice per option
```

More than one match also produces an `invalid_union` issue, distinguishable by its **empty** `Errors` slice — there were no per-option failures to report:

```go
res:= z.XorOf(z.String, z.String.Min(1)).SafeParse("hello")
res.Error.Issues[0].Code           // invalid_union
len(res.Error.Issues[0].Errors)    // 0 — matched too many, not too few
```

Attach a message when the distinction matters to your callers:

```go
z.Xor(options, "value must match exactly one shape")
```

## Options accessor

```go
schema:= z.XorOf(z.String, z.Number)
len(schema.Options) // 2
```

## JSON Schema

`Xor` emits `oneOf`, while `Union` emits `anyOf` — the same distinction JSON Schema draws:

```go
js, _:= z.ToJSONSchema(z.XorOf(z.String, z.Number))
// js["oneOf"] = [{"type":"string"}, {"type":"number"}]
```

## Signatures

```go
func Xor(options []AnySchemaLike, params...any) *XorSchema
func XorOf(options...AnySchemaLike) *XorSchema
```

`Xor` takes the params list (string message, `ErrorMap`, or `Params`); `XorOf` is the variadic convenience without params.

## Related

- [Union](#/api/union) — first match wins
- [Discriminated union](#/api/discriminated-union) — O(1) dispatch on a literal tag
- [JSON Schema](#/api/json-schema) — `oneOf` vs `anyOf`
