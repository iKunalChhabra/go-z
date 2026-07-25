# Record

`z.Record(keySchema, valueSchema)` ports `z.record(key, value)`. Validates `map[string]any` (and other string-keyed maps). Output is `map[string]any`.

```go
scores:= z.Record(z.String, z.Number)

out:= scores.MustParse(map[string]any{
    "alice": 10,
    "bob":   20,
})
```

## Open records (string keys)

When the key schema has **no** `Internals.Values` set (e.g. plain `String`), every enumerable key is validated independently:

```go
schema:= z.Record(z.String, z.String)
schema.MustParse(map[string]any{"any": "key"})

res:= schema.SafeParse(map[string]any{"a": 1})
// Value type error — Path: ["a"]

res = schema.SafeParse([]any{})
// Expected: "record"
```

### Invalid keys → `invalid_key`

If the key schema fails, go-z emits `invalid_key` with nested issues (`Origin: "record"`), path set to the key:

```go
// Example: keys must be emails
schema:= z.Record(z.String.Email, z.Number)
res:= schema.SafeParse(map[string]any{
    "not-an-email": 1,
})
// Code: invalid_key
// Origin: "record"
// Path: ["not-an-email"]
// Issues: nested invalid_format email
```

### Loose keys

`.Loose` keeps entries whose keys fail validation (pass through raw), instead of emitting `invalid_key`:

```go
schema:= z.Record(z.String.Email, z.Number).Loose
got:= schema.MustParse(map[string]any{
    "ok@x.co": 1,
    "bad":     2, // kept despite invalid key
})
```

## Enum / literal keys (exhaustive)

When the key schema defines `Internals.Values` (Enum, Literal, Null, …), the record becomes **exhaustive**:

1. Every expected key must be present (missing → value schema sees `Missing`).
2. Extra keys → `unrecognized_keys`.

```go
key:= z.Enum("ok")
schema:= z.Record(key, z.String)

schema.MustParse(map[string]any{"ok": "yes"})

res:= schema.SafeParse(map[string]any{"ok": "yes", "no": "x"})
// Code: unrecognized_keys, Keys: ["no"]

res = schema.SafeParse(map[string]any{})
// Missing required key "ok" → invalid_type at path ["ok"]
```

Works well with `Object.Keyof`:

```go
shape:= z.Object(z.Shape{
    "width":  z.Number,
    "height": z.Number,
})
dims:= z.Record(shape.Keyof, z.Number)
dims.MustParse(map[string]any{"width": 1, "height": 2})
```

## Nested paths

```go
schema:= z.Record(z.String, z.Object(z.Shape{
    "n": z.String,
}))
res:= schema.SafeParse(map[string]any{
    "x": map[string]any{},
})
// Path: ["x", "n"]
```

## API surface

```go
func Record(keySchema, valueSchema AnySchemaLike, params...any) *RecordSchema
func (r *RecordSchema) Loose *RecordSchema
```
