# String

`z.String()` is the Go port of `z.string`. It accepts Go `string` values, runs fluent checks, and returns a typed `string` from `Parse` / `SafeParse`.

```go
import "github.com/iKunalChhabra/go-z/z"

schema := z.String().Min(1).Max(100)

s, err := schema.Parse("hello")
// s == "hello", err == nil
```

:::tip Module
All examples use `import "github.com/iKunalChhabra/go-z/z"` (alias `z`) to match `z` convention.
:::

## Basics

Without checks, only the type is validated:

```go
schema := z.String()

schema.MustParse("ok")

res := schema.SafeParse(123)
// res.Success == false
// res.Error.Issues[0].Code == z.IssueInvalidType
// res.Error.Issues[0].Expected == "string"
// Message: "Invalid input: expected string, received number"
```

## Length checks

Length is measured in **UTF-16 code units** (same as JavaScript `string.length`), not Go `len(s)` bytes or rune count.

| Method | Meaning | Issue on failure |
|--------|---------|------------------|
| `Min(n)` | length ≥ n | `too_small`, `Origin: "string"` |
| `Max(n)` | length ≤ n | `too_big`, `Origin: "string"` |
| `Length(n)` | length == n | `too_small` / `too_big` with `Exact: true` |
| `NonEmpty` | alias of `Min(1)` | same as `Min` |

```go
minFive := z.String().Min(5, "min5")
maxFive := z.String().Max(5, "max5")
justFive := z.String().Length(5)
nonempty := z.String().NonEmpty("nonempty")

minFive.MustParse("12345")
maxFive.MustParse("1234")
justFive.MustParse("12345")
nonempty.MustParse("x")

res := minFive.SafeParse("1234")
// Message: "min5"

res = z.String().Min(5).SafeParse("hi")
// Message: "Too small: expected string to have >=5 characters"

res = z.String().Max(2).SafeParse("hello")
// Message: "Too big: expected string to have <=2 characters"

res = z.String().Length(3).SafeParse("ab")
// Issue.Exact == true, Code == too_small
```

## Content checks

| Method | Format field | Notes |
|--------|--------------|-------|
| `Regex(re)` | `"regex"` | Sets `Pattern` (JS-style `/…/` literal) |
| `Includes(sub)` | `"includes"` | Optional start index: `Includes(sub, 2)` |
| `StartsWith(prefix)` | `"starts_with"` | Sets `Prefix` |
| `EndsWith(suffix)` | `"ends_with"` | Sets `Suffix` |
| `Uppercase` | `"uppercase"` | **Check**, not transform |
| `Lowercase` | `"lowercase"` | **Check**, not transform |

All emit `invalid_format` with `Origin: "string"` (except where noted).

```go
import "regexp"

re := z.String().Regex(regexp.MustCompile(`^moo+$`))
re.MustParse("moooo")

res := re.SafeParse("boooo")
// Code: invalid_format, Format: "regex", Pattern: "/^moo+$/"
// Message: "Invalid string: must match pattern /^moo+$/"

custom := z.String().Regex(regexp.MustCompile(`^moo+$`), "Custom error message")
res = custom.SafeParse("boooo")
// Message: "Custom error message"
```

```go
includes := z.String().Includes("includes")
includes.MustParse("XincludesXX")
_ = includes.SafeParse("XincludeXX") // fail

// Start searching from byte index 2
from2 := z.String().Includes("includes", 2)
from2.MustParse("XXXincludesXX")
_ = from2.SafeParse("XincludesXX") // fail — match starts before index 2

starts := z.String().StartsWith("startsWith")
ends := z.String().EndsWith("endsWith")
starts.MustParse("startsWithX")
ends.MustParse("XendsWith")

res := z.String().StartsWith("ab").SafeParse("x")
// Message: `Invalid string: must start with "ab"`

res = z.String().EndsWith("yz").SafeParse("x")
// Message: `Invalid string: must end with "yz"`
```

```go
z.String().Uppercase().MustParse("ABC")
res := z.String().Uppercase().SafeParse("Abc")
// Message: "Invalid uppercase"

z.String().Lowercase().MustParse("abc")
_ = z.String().Lowercase().SafeParse("Abc") // fail
```

:::info Uppercase vs ToUpperCase
`Uppercase` / `Lowercase` **reject** mixed case. `ToUpperCase` / `ToLowerCase` **rewrite** the value. Use checks when you want validation; use overwrites when you want normalization.
:::

## Overwrite transforms

These mutate the parsed string in place (“overwrite” checks). They do not produce issues on their own.

| Method | Effect |
|--------|--------|
| `Trim` | `strings.TrimSpace` |
| `ToLowerCase` | Unicode lowercasing |
| `ToUpperCase` | Unicode uppercasing |
| `Normalize` | Unicode NFC |

```go
got, err := z.String().Trim().Min(2).Parse(" 12 ")
// got == "12", err == nil

// Order matters: checks run in attachment order.
got, err = z.String().Min(2).Trim().Parse(" 1 ")
// Trim runs after Min — Min sees " 1 " (length 3) → success, then trim → "1"

_, err = z.String().Trim().Min(2).Parse(" 1 ")
// Trim first → "1", then Min(2) fails

got = z.String().ToLowerCase().MustParse("ASDF") // "asdf"
got = z.String().ToUpperCase().MustParse("asdf") // "ASDF"
got = z.String().Normalize().MustParse("e\u0301") // NFC "é"
```

## Coercion

Pass `z.Params{Coerce: true}` (or use [`z.Coerce.String()`](/api/coerce)) to stringify common primitives before the type check:

```go
s := z.String(z.Params{Coerce: true})

s.MustParse(123)   // "123"
s.MustParse(true)  // "true"
s.MustParse(nil)   // "null"
s.MustParse(12.5)  // "12.5"
```

See [Coercion](/api/coerce) for the full conversion table.

## Params & custom messages

Fluent methods accept trailing params:

- `string` → fixed error message
- `z.ErrorMap` / `func(*z.Issue) string` → dynamic message
- `z.Params` → `{ Error, Abort, Coerce }`

```go
schema := z.String().Min(5, "too short").Max(20, "too long")

res := schema.SafeParse("hi")
// Message: "too short"

// Abort stops later checks that lack a When gate
s := z.String().
    Email(z.Params{Abort: true}).
    Regex(regexp.MustCompile(`^x$`))
res = s.SafeParse("not-email")
// Only the email issue is reported
```

Constructor-level params:

```go
schema := z.String("must be a string")
res := schema.SafeParse(42)
// Message: "must be a string"
```

## Chaining with formats

Length and case checks compose with [string formats](/api/string-formats):

```go
schema := z.String().Email().Min(10).Lowercase()
schema.MustParse("longemail@example.com")
_ = schema.SafeParse("ort@e.co")              // too short
_ = schema.SafeParse("EMAIL@EXAMPLE.COM")     // not lowercase
```

## API surface

```go
func String(params ...any) *StringSchema

func (s *StringSchema) Min(n int, params ...any) *StringSchema
func (s *StringSchema) Max(n int, params ...any) *StringSchema
func (s *StringSchema) Length(n int, params ...any) *StringSchema
func (s *StringSchema) NonEmpty(params ...any) *StringSchema
func (s *StringSchema) Regex(pattern *regexp.Regexp, params ...any) *StringSchema
func (s *StringSchema) Includes(value string, params ...any) *StringSchema
func (s *StringSchema) StartsWith(value string, params ...any) *StringSchema
func (s *StringSchema) EndsWith(value string, params ...any) *StringSchema
func (s *StringSchema) Uppercase(params ...any) *StringSchema
func (s *StringSchema) Lowercase(params ...any) *StringSchema
func (s *StringSchema) Trim() *StringSchema
func (s *StringSchema) ToLowerCase() *StringSchema
func (s *StringSchema) ToUpperCase() *StringSchema
func (s *StringSchema) Normalize() *StringSchema
func (s *StringSchema) Check(checks ...*Check) *StringSchema
// Plus format methods: Email, URL, UUID, … — see String formats
```
