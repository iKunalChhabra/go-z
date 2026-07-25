# Codec (encode / decode)

A codec is a **bidirectional** schema: it validates and converts in both directions. Decoding runs input → output; encoding runs output → input.

```go
isoDate := z.Codec(z.String().ISODateTime(), z.Time(), z.CodecTx{
    Decode: func(v any, _ *z.RefinementCtx) (any, error) {
        return time.Parse(time.RFC3339Nano, v.(string))
    },
    Encode: func(v any, _ *z.RefinementCtx) (any, error) {
        return v.(time.Time).UTC.Format(time.RFC3339Nano), nil
    },
})

t, err := z.Decode(isoDate, "2024-01-15T10:30:00Z") // time.Time
s, err := z.Encode(isoDate, t)                      // "2024-01-15T10:30:00Z"
```

`Decode` validates with the **input** schema, runs your transform, then validates with the **output** schema. `Encode` does the reverse.

## Direction

Both directions run the same schema tree, so a codec nested anywhere inside an object or array works in both directions:

```go
event := z.Object(z.Shape{
    "name": z.String(),
    "at":   isoDate,
})

parsed, _ := z.Decode(event, map[string]any{"name": "launch", "at": "2024-01-15T10:30:00Z"})
// parsed["at"] is a time.Time

raw, _ := z.Encode(event, parsed)
// raw["at"] is an ISO string again
```

Direction-aware behavior across the library:

| Schema | Decode | Encode |
|---|---|---|
| `Pipe(a, b)` | a then b | b then a (reversed) |
| `Default` / `Prefault` | substitutes for absent input | **skipped** — encoding must not invent values |
| `Catch` | replaces failures with the fallback | **skipped** — errors surface |
| `Transform` | runs | **panics** — a one-way transform cannot be reversed |

:::warn Transform is one-way
`z.Transform` has no inverse, so encoding through one panics. Use a `Codec` when you need the value to travel in both directions.
:::

## Safe variants

```go
res := z.SafeDecode(isoDate, "not-a-date")
if !res.Success {
    fmt.Println(res.Error.Issues[0].Code) // invalid_format
}

res = z.SafeEncode(isoDate, time.Now)
```

`Decode` and `Encode` return `(any, error)`; the safe forms return `SafeParseResult[any]`.

## InvertCodec

Swaps the two sides, so decode becomes encode:

```go
dateToISO := z.InvertCodec(isoDate)

s, _ := z.Decode(dateToISO, time.Now) // now a string
```

## JSONStringCodec

Ships with the common "field holds a JSON string" case:

```go
payload := z.JSONStringCodec(z.Object(z.Shape{
    "retries": z.Int().Gte(0),
}))

cfg, _ := z.Decode(payload, `{"retries":3}`) // map[string]any
raw, _ := z.Encode(payload, cfg)             // `{"retries":3}`
```

Invalid JSON produces an `invalid_format` issue with `format: "json"` rather than a Go error.

## Reading the direction inside a check

`ParseCtx.Direction` carries the current direction; `ctx.IsEncode()` is the readable form. Custom schemas and checks can branch on it:

```go
if ctx.IsEncode() {
    // encoding: skip anything that would fabricate data
}
```

## Accessors

```go
isoDate.In  // z.String().ISODateTime()
isoDate.Out // z.Time()
```

## Signatures

```go
type CodecTx struct {
    Decode func(any, *RefinementCtx) (any, error)
    Encode func(any, *RefinementCtx) (any, error)
}

func Codec(in, out AnySchemaLike, tx CodecTx) *CodecSchema
func InvertCodec(c *CodecSchema) *CodecSchema
func JSONStringCodec(schema AnySchemaLike) *CodecSchema

func Decode(schema AnySchemaLike, data any) (any, error)
func Encode(schema AnySchemaLike, data any) (any, error)
func SafeDecode(schema AnySchemaLike, data any) SafeParseResult[any]
func SafeEncode(schema AnySchemaLike, data any) SafeParseResult[any]
```

`Decode` on a schema without codecs behaves exactly like `Parse`, so it is safe to call generically.

## Related

- [Pipe & Transform](#/api/pipe-transform) — one-way composition
- [Time](#/api/time) — the `time.Time` edge
- [Struct binding](#/integrations/tostruct) — the other way to leave the JSON model
