# Template literal

`TemplateLiteral` validates strings assembled from literal text and other schemas — `z.templateLiteral([...])`. Parts are concatenated into a single anchored pattern at construction time, so parsing is one regex match.

```go
schema := z.TemplateLiteral([]any{
    "https://",
    z.String().Regex(regexp.MustCompile(`\w+`)),
    ".",
    z.Enum("com", "net"),
})

schema.Parse("https://example.com")   // ok
schema.SafeParse("https://example.org") // fails: "org" is not in the enum
schema.SafeParse("http://example.com")  // fails: wrong scheme
```

## Parts

A part is either a literal value or a schema:

| Part | Contributes |
|---|---|
| `string` | the escaped literal text |
| `int`, `int64`, `float64`, `bool`, `nil` | its literal rendering (`1`, `true`, `null`) |
| `*regexp.Regexp` | the pattern as written |
| a schema with `Values` (`Literal`, `Enum`) | an alternation of its members |
| `z.String()` | any characters |
| `z.Number()` / `z.Int64()` | a JSON number |
| `z.Bool()` | `true\|false` |
| `z.Optional(...)` | the inner pattern, made optional |
| `z.Nullable(...)` | the inner pattern or `null` |
| a nested `TemplateLiteral` | its pattern, inlined |

```go
version := z.TemplateLiteral([]any{"v", z.Number(), z.Literal("-beta").Optional()})

version.Parse("v1")      // ok
version.Parse("v1-beta") // ok
```

:::warn Unsupported parts panic at definition time
A part that has no pattern — an object, an array, a schema whose shape cannot be expressed as a regex — panics while the schema is built, like any other invalid schema definition. It cannot fail at request time.
:::

## Combining with coercion

Template literals validate strings, so coerce first when the input may not be one:

```go
id := z.Pipe(z.Coerce.String(), z.TemplateLiteral([]any{"id-", z.Number()}))
id.Parse("id-42") // "id-42"
```

## Accessors

```go
schema.Parts() // a copy of the parts slice
```

The compiled pattern is also published on `Internals.Pattern`, which is how nested template literals inline each other and how [JSON Schema](#/api/json-schema) emits a `pattern` for the string.

## Signatures

```go
func TemplateLiteral(parts []any, params ...any) *TemplateLiteralSchema

func (s *TemplateLiteralSchema) Parts() []any
func (s *TemplateLiteralSchema) Check(checks ...*Check) *TemplateLiteralSchema
```

Params follow the usual pattern: string message, `ErrorMap`, or `Params`.

## Related

- [String formats](#/api/string-formats) — built-in format checks
- [Literal & Enum](#/api/literal-enum) — the member sets template parts read
- [JSON Schema](#/api/json-schema) — template literals become `pattern`
