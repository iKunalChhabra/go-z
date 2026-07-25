# Record

`zod.Record(keySchema, valueSchema)` ports `z.record(key, value)`. Validates `map[string]any` (and other string-keyed maps). Output is `map[string]any`.

```go
scores := zod.Record(zod.String(), zod.Number())

out := scores.MustParse(map[string]any{
    "alice": 10,
    "bob":   20,
})
```

## Open records (string keys)

When the key schema has **no** `Internals.Values` set (e.g. plain `String()`), every enumerable key is validated independently:

```go
schema := zod.Record(zod.String(), zod.String())
schema.MustParse(map[string]any{"any": "key"})

res := schema.SafeParse(map[string]any{"a": 1})
// Value type error — Path: ["a"]

res = schema.SafeParse([]any{})
// Expected: "record"
```

### Invalid keys → `invalid_key`

If the key schema fails, go-zod emits `invalid_key` with nested issues (`Origin: "record"`), path set to the key:

```go
// Example: keys must be emails
schema := zod.Record(zod.String().Email(), zod.Number())
res := schema.SafeParse(map[string]any{
    "not-an-email": 1,
})
// Code: invalid_key
// Origin: "record"
// Path: ["not-an-email"]
// Issues: nested invalid_format email
```

### Loose keys

`.Loose()` keeps entries whose keys fail validation (pass through raw), instead of emitting `invalid_key`:

```go
schema := zod.Record(zod.String().Email(), zod.Number()).Loose()
got := schema.MustParse(map[string]any{
    "ok@x.co": 1,
    "bad":     2, // kept despite invalid key
})
```

## Enum / literal keys (exhaustive)

When the key schema defines `Internals.Values` (Enum, Literal, Null, …), the record becomes **exhaustive**:

1. Every expected key must be present (missing → value schema sees `Missing`).
2. Extra keys → `unrecognized_keys`.

```go
key := zod.Enum("ok")
schema := zod.Record(key, zod.String())

schema.MustParse(map[string]any{"ok": "yes"})

res := schema.SafeParse(map[string]any{"ok": "yes", "no": "x"})
// Code: unrecognized_keys, Keys: ["no"]

res = schema.SafeParse(map[string]any{})
// Missing required key "ok" → invalid_type at path ["ok"]
```

Works well with `Object.Keyof()`:

```go
shape := zod.Object(zod.Shape{
    "width":  zod.Number(),
    "height": zod.Number(),
})
dims := zod.Record(shape.Keyof(), zod.Number())
dims.MustParse(map[string]any{"width": 1, "height": 2})
```

## Nested paths

```go
schema := zod.Record(zod.String(), zod.Object(zod.Shape{
    "n": zod.String(),
}))
res := schema.SafeParse(map[string]any{
    "x": map[string]any{},
})
// Path: ["x", "n"]
```

## API surface

```go
func Record(keySchema, valueSchema AnySchemaLike, params ...any) *RecordSchema
func (r *RecordSchema) Loose() *RecordSchema
```
