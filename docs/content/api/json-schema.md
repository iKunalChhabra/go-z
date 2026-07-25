# JSON Schema export

`ToJSONSchema` converts a schema into a JSON Schema document (`map[string]any`), ready to marshal into an OpenAPI spec or hand to a client validator.

```go
js, err:= z.ToJSONSchema(z.Object(z.Shape{
    "name":  z.String.Min(2),
    "email": z.String.Email,
    "age":   z.Optional(z.Int.Gte(0)),
}))

out, _:= json.MarshalIndent(js, "", "  ")
fmt.Println(string(out))
```

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "age": { "type": "integer", "minimum": 0, "maximum": 9007199254740991 },
    "email": { "type": "string", "format": "email" },
    "name": { "type": "string", "minLength": 2 }
  },
  "required": ["email", "name"]
}
```

Optional fields are simply absent from `required`. `Int` carries the JSON safe-integer bound, which is why `age` gets a `maximum` you did not write — the same thing `toJSONSchema` does for `z.int`.

## Options

```go
js, err:= z.ToJSONSchema(schema, z.ToJSONSchemaOpts{
    Target:          z.JSONSchemaDraft202012, // or Draft07, OpenAPI30
    Unrepresentable: "any",                   // or "throw" (default)
    IO:              "input",                 // or "output" (default)
    Metadata:        z.GlobalRegistry,        // where id/title/description come from
})
```

| Option | Default | Meaning |
|---|---|---|
| `Target` | `JSONSchemaDraft202012` | dialect; sets `$schema` (OpenAPI 3.0 omits it) |
| `Unrepresentable` | `"throw"` | what to do with types JSON Schema cannot express |
| `IO` | `"output"` | which side of a pipe or codec to emit |
| `Metadata` | `GlobalRegistry` | registry consulted for `id` / `title` / `description` / `deprecated` |

### Unrepresentable types

`BigInt`, `Time`, `Map`, `Set`, `NaN`, `Undefined`, and `Void` have no JSON Schema equivalent. The default throws:

```go
_, err:= z.ToJSONSchema(z.BigInt)
// error: unrepresentable type "bigint"
```

Pass `Unrepresentable: "any"` to emit an empty schema (`{}`) instead — the JSON Schema way of saying "anything".

### Input vs output

A pipe or codec has two sides. `IO` picks which one the document describes:

```go
p:= z.Pipe(z.String, z.Number)

z.ToJSONSchema(p, z.ToJSONSchemaOpts{IO: "input"})  // {"type": "string"}
z.ToJSONSchema(p, z.ToJSONSchemaOpts{IO: "output"}) // {"type": "number"}
```

Use `input` when documenting what a client should send, `output` when documenting what your handler produces.

## Metadata

Register a description or title and it flows into the document:

```go
email:= z.Describe(z.String.Email, "Primary contact address")
js, _:= z.ToJSONSchema(email)
// js["description"] == "Primary contact address"
```

`id`, `title`, `description`, and `deprecated` are read from the registry entry.

## Type coverage

| Schema | JSON Schema |
|---|---|
| `String` (+ format, min, max, regex) | `type: string` with `format` / `minLength` / `maxLength` / `pattern` |
| `Number`, `Int`, `Int64` | `type: number` / `integer` with `minimum` / `maximum` / `multipleOf` |
| `Bool`, `Null`, `Any`, `Unknown`, `Never` | `boolean`, `null`, `{}`, `{}`, `not: {}` |
| `Literal`, `Enum` | `const` / `enum` |
| `Object` | `properties`, `required`, `additionalProperties` (from `.Strict` / `.Catchall`) |
| `Array`, `Tuple` | `items` / `prefixItems` with `minItems` / `maxItems` |
| `Record` | `type: object` with `additionalProperties` |
| `Union` | `anyOf` |
| `Xor` | `oneOf` |
| `Intersection` | `allOf` |
| `Optional`, `Default`, `Catch`, `Readonly`, wrappers | the inner schema |
| `Nullable` | `type: [T, "null"]` |
| `TemplateLiteral` | `type: string` with `pattern` |

Recursive schemas built with `Lazy` are inlined once and then cut off, so a self-referencing type does not loop.

## Signatures

```go
func ToJSONSchema(schema AnySchemaLike, opts...ToJSONSchemaOpts) (map[string]any, error)

type ToJSONSchemaOpts struct {
    Target          JSONSchemaTarget          // JSONSchemaDraft202012 | JSONSchemaDraft07 | JSONSchemaOpenAPI30
    Metadata        *Registry[map[string]any]
    Unrepresentable string                    // "throw" | "any"
    IO              string                    // "output" | "input"
}
```

:::info Export only
There is no `fromJSONSchema`. Going the other way — generating go-z schemas from a JSON Schema document — is not implemented.
:::

## Related

- [Registries & metadata](#/reference/cheatsheet) — where titles and descriptions live
- [Xor](#/api/xor) — `oneOf` vs `anyOf`
- [Codec](#/api/codec) — the input/output distinction
