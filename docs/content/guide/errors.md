# Issues & errors

When parsing fails, go-z returns a `*z.Error` whose `Issues` slice carries one entry per problem. This page covers the error type, issue fields, issue codes, and the formatting helpers: `Flatten`, `Format`, `Treeify`, `Prettify`, and `ToDotPath`.

## The Error type

```go
type Error struct {
	Issues []Issue
}
```

`Error` returns pretty-printed JSON of the issues:

```go
_, err:= z.String.Min(5).Parse("hi")
fmt.Println(err)
```

```json
[
  {
    "code": "too_small",
    "path": [],
    "message": "Too small: expected string to have >=5 characters",
    "origin": "string",
    "minimum": 5,
    "inclusive": true
  }
]
```

Use `AsError` when you need structured data. It unwraps through
`fmt.Errorf("%w", …)`, so it keeps working once the error crosses a layer that
adds context — a bare type assertion does not:

```go
zerr, ok:= z.AsError(err)
if !ok {
	// not a validation error
}
for _, iss:= range zerr.Issues {
	fmt.Println(iss.Code, iss.Path, iss.Message)
}
```

`z.IsError(err)` is the boolean-only form, and `errors.As(err, &zerr)` works
directly too.

`SafeParse` already exposes `*z.Error` on the result:

```go
res:= schema.SafeParse(input)
if !res.Success {
	_ = res.Error.Issues
}
```

## Issue fields

```go
type Issue struct {
	Code    IssueCode // e.g. "too_small"
	Path    []any     // ["user","email"] or ["items", 0]
	Message string    // finalized human message
	Input   any       // cleared unless ParseCtx.ReportInput

	// Populated depending on Code:
	Expected      string     // invalid_type
	Origin        string     // "string" | "number" | "array" |...
	Minimum       any        // too_small
	Maximum       any        // too_big
	Inclusive     bool
	Exact         bool
	Format        string     // invalid_format ("email", "uuid",...)
	Pattern       string
	Prefix        string
	Suffix        string
	Includes      string
	Algorithm     string
	Divisor       float64    // not_multiple_of
	Keys          []string   // unrecognized_keys
	Errors        [][]Issue  // invalid_union
	Discriminator string
	Issues        []Issue    // invalid_key / invalid_element
	Key           any
	Values        []any      // invalid_value
	Params        map[string]any // custom
}
```

Only fields relevant to the code are set. JSON tags are stable, so issues marshal directly to clients.

### Paths

Paths are `[]any` segments — strings for object keys, ints for array indices:

```go
// Issue at user.addresses[1].zip
iss.Path // []any{"user", "addresses", 1, "zip"}

fmt.Println(z.ToDotPath(iss.Path))
// user.addresses[1].zip
```

## Issue codes

| Code | Constant | Typical cause |
|---|---|---|
| `invalid_type` | `IssueInvalidType` | Wrong Go/JSON type (`expected` vs received) |
| `too_big` | `IssueTooBig` | Above max length / value / size |
| `too_small` | `IssueTooSmall` | Below min length / value / size |
| `invalid_format` | `IssueInvalidFormat` | Email, UUID, regex, URL, … |
| `not_multiple_of` | `IssueNotMultipleOf` | Number not divisible by divisor |
| `unrecognized_keys` | `IssueUnrecognizedKeys` | Extra object keys (strict) |
| `invalid_union` | `IssueInvalidUnion` | No union member matched |
| `invalid_key` | `IssueInvalidKey` | Record/map key failed |
| `invalid_element` | `IssueInvalidElement` | Map/set element failed |
| `invalid_value` | `IssueInvalidValue` | Literal / enum mismatch |
| `custom` | `IssueCustom` | Refine / custom checks |

```go
res:= z.String.Email.SafeParse("nope")
fmt.Println(res.Error.Issues[0].Code)    // invalid_format
fmt.Println(res.Error.Issues[0].Format)  // email
```

## Formatting helpers

All helpers accept `*z.Error`. They are pure functions — safe to call from any goroutine.

### Prettify

Human-readable multi-line string. Issues sorted by path length; each line is `✖ message` plus optional `→ at path`.

```go
schema:= z.Object(z.Shape{
	"name":  z.String.Min(2),
	"email": z.String.Email,
})

_, err:= schema.Parse(map[string]any{
	"name":  "A",
	"email": "bad",
})
fmt.Println(z.Prettify(err.(*z.Error)))
```

```text
✖ Too small: expected string to have >=2 characters
  → at name
✖ Invalid email address
  → at email
```

Great for CLI tools and server logs.

### Flatten

Splits root-level issues (`formErrors`) from first-segment field errors (`fieldErrors`) — classic form UX.

```go
flat:= z.Flatten(zerr)
// flat.FormErrors  []string
// flat.FieldErrors map[string][]string
```

Example:

```go
err:= &z.Error{Issues: []z.Issue{
	{Code: z.IssueCustom, Message: "Must be equal", Path: []any{}},
	{Code: z.IssueTooSmall, Message: "Too small", Path: []any{"name"}},
}}

flat:= z.Flatten(err)
fmt.Println(flat.FormErrors)           // [Must be equal]
fmt.Println(flat.FieldErrors["name"])  // [Too small]
```

Use `FlattenMap` when you want custom mapped values instead of messages:

```go
codes:= z.FlattenMap(zerr, func(iss z.Issue) string {
	return string(iss.Code)
})
```

### Format

Nested map with `"_errors"` arrays at each node — a nested form.

```go
formatted:= z.Format(zerr)
```

Example shape:

```go
zerr:= &z.Error{Issues: []z.Issue{
	{Code: z.IssueUnrecognizedKeys, Path: []any{}, Message: `Unrecognized key: "extra"`},
	{Code: z.IssueInvalidType, Path: []any{"username"}, Message: "expected string"},
	{Code: z.IssueInvalidType, Path: []any{"favoriteNumbers", 1}, Message: "expected number"},
	{Code: z.IssueInvalidType, Path: []any{"nesting", "a"}, Message: "expected string"},
}}

fmt.Printf("%#v\n", z.Format(zerr))
```

Conceptual JSON:

```json
{
  "_errors": ["Unrecognized key: \"extra\""],
  "username": { "_errors": ["expected string"] },
  "favoriteNumbers": {
    "_errors": [],
    "1": { "_errors": ["expected number"] }
  },
  "nesting": {
    "_errors": [],
    "a": { "_errors": ["expected string"] }
  }
}
```

`Format` also expands nested `invalid_union` / `invalid_key` / `invalid_element` issues.

### Treeify

Typed tree with `Errors`, `Properties`, and `Items` — better when you want structure without `map[string]any`:

```go
type ErrorTree struct {
	Errors     []string
	Properties map[string]*ErrorTree
	Items      []*ErrorTree
}

tree:= z.Treeify(zerr)
if tree.Properties["username"] != nil {
	fmt.Println(tree.Properties["username"].Errors)
}
```

Array indices land in `Items`:

```go
// path ["tags", 2] → tree.Properties["tags"].Items[2].Errors
```

### ToDotPath

Converts a path slice to a JS-like path string:

```go
z.ToDotPath([]any{"user", "emails", 0, "address"})
// "user.emails[0].address"

z.ToDotPath([]any{"weird.key"})
// `["weird.key"]`
```

`Prettify` uses `ToDotPath` for the `→ at …` lines.

:::tip Pick a formatter
| Need | Use |
|---|---|
| Logs / CLI | `Prettify` |
| Flat HTML forms | `Flatten` |
| Nested JSON for SPAs | `Format` or `Treeify` |
| Single path label | `ToDotPath` |
:::

## End-to-end example

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/iKunalChhabra/go-z/z"
)

func main {
	schema:= z.Object(z.Shape{
		"name": z.String.Min(2),
		"tags": z.Array(z.String.Min(1)).Min(1),
	})

	_, err:= schema.Parse(map[string]any{
		"name": "A",
		"tags": []any{"", "ok"},
	})
	zerr:= err.(*z.Error)

	fmt.Println("--- Prettify ---")
	fmt.Println(z.Prettify(zerr))

	fmt.Println("--- Flatten ---")
	b, _:= json.MarshalIndent(z.Flatten(zerr), "", "  ")
	fmt.Println(string(b))

	fmt.Println("--- Format ---")
	b, _ = json.MarshalIndent(z.Format(zerr), "", "  ")
	fmt.Println(string(b))
}
```

Possible Prettify output:

```text
✖ Too small: expected string to have >=2 characters
  → at name
✖ Too small: expected string to have >=1 characters
  → at tags[0]
```

## Reporting input in issues

By default finalized issues omit `Input`. Opt in per parse:

```go
out, err:= schema.ParseCtx(input, &z.ParseCtx{ReportInput: true})
if err != nil {
	iss:= err.(*z.Error).Issues[0]
	fmt.Printf("bad value: %#v\n", iss.Input)
}
```

Avoid enabling this for passwords or tokens in production responses.

## Related

- [Error maps & locales](#/guide/error-maps) — customize `Message`
- [Checks & refinements](#/guide/checks) — how checks produce issues
- [HTTP error shapes](#/integrations/http-errors) — wiring formatters into APIs
