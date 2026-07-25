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
| `z.String().Regex(re)` | that pattern |
| `z.String().Email()` and other formats | the format's pattern |
| `z.Number()` / `z.Float64()` | a JSON number |
| `z.Int()` / `z.Int64()` / `z.Int32()` | an integer |
| `z.Uint32()` | an unsigned integer |
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
A template validates by matching **one** regular expression, so every part has to
contribute a pattern. Two kinds of part cannot, and both panic while the schema is
built rather than being silently ignored at request time:

- A schema with no pattern at all — an object, an array, a map.
- A schema carrying a check with no pattern equivalent: `z.String().Min(5)`,
  `z.Int().Gte(10)`, a `Refine`, or a transform like `Trim`. Length and range are
  not expressible in the composed pattern, so a template that accepted them would
  quietly ignore them.

```go
z.TemplateLiteral([]any{"id-", z.String().Regex(digits)})  // ok — the pattern composes
z.TemplateLiteral([]any{"id-", z.String().Min(5)})         // panics: Min has no pattern
```

Validate that field with its own schema instead, or express the constraint in the
regex (`^\d{5,}$`).
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
