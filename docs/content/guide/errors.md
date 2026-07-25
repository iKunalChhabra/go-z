# Issues & ZodError

When parsing fails, go-zod returns a `*ZodError` whose `Issues` slice mirrors Zod v4’s issue objects. This page covers the error type, issue fields, issue codes, and the formatting helpers: `Flatten`, `Format`, `Treeify`, `Prettify`, and `ToDotPath`.

## ZodError

```go
type ZodError struct {
	Issues []Issue
}
```

`Error()` returns pretty-printed JSON of the issues (Zod’s `ZodError.message` behavior):

```go
_, err := zod.String().Min(5).Parse("hi")
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

Always type-assert when you need structured data:

```go
zerr, ok := err.(*zod.ZodError)
if !ok {
	// unexpected — schema.Parse should only return *ZodError
}
for _, iss := range zerr.Issues {
	fmt.Println(iss.Code, iss.Path, iss.Message)
}
```

`SafeParse` already exposes `*ZodError` on the result:

```go
res := schema.SafeParse(input)
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
	Origin        string     // "string" | "number" | "array" | ...
	Minimum       any        // too_small
	Maximum       any        // too_big
	Inclusive     bool
	Exact         bool
	Format        string     // invalid_format ("email", "uuid", ...)
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

Only fields relevant to the code are set. JSON tags match Zod v4 so you can marshal issues to clients that already speak Zod.

### Paths

Paths are `[]any` segments — strings for object keys, ints for array indices:

```go
// Issue at user.addresses[1].zip
iss.Path // []any{"user", "addresses", 1, "zip"}

fmt.Println(zod.ToDotPath(iss.Path))
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
res := zod.String().Email().SafeParse("nope")
fmt.Println(res.Error.Issues[0].Code)    // invalid_format
fmt.Println(res.Error.Issues[0].Format)  // email
```

## Formatting helpers

All helpers accept `*ZodError`. They are pure functions — safe to call from any goroutine.

### Prettify

Human-readable multi-line string. Issues sorted by path length; each line is `✖ message` plus optional `→ at path`.

```go
schema := zod.Object(zod.Shape{
	"name":  zod.String().Min(2),
	"email": zod.String().Email(),
})

_, err := schema.Parse(map[string]any{
	"name":  "A",
	"email": "bad",
})
fmt.Println(zod.Prettify(err.(*zod.ZodError)))
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
flat := zod.Flatten(zerr)
// flat.FormErrors  []string
// flat.FieldErrors map[string][]string
```

Example:

```go
err := &zod.ZodError{Issues: []zod.Issue{
	{Code: zod.IssueCustom, Message: "Must be equal", Path: []any{}},
	{Code: zod.IssueTooSmall, Message: "Too small", Path: []any{"name"}},
}}

flat := zod.Flatten(err)
fmt.Println(flat.FormErrors)           // [Must be equal]
fmt.Println(flat.FieldErrors["name"])  // [Too small]
```

Use `FlattenMap` when you want custom mapped values instead of messages:

```go
codes := zod.FlattenMap(zerr, func(iss zod.Issue) string {
	return string(iss.Code)
})
```

### Format

Nested map with `"_errors"` arrays at each node — Zod’s `formatError`.

```go
formatted := zod.Format(zerr)
```

Example shape:

```go
zerr := &zod.ZodError{Issues: []zod.Issue{
	{Code: zod.IssueUnrecognizedKeys, Path: []any{}, Message: `Unrecognized key: "extra"`},
	{Code: zod.IssueInvalidType, Path: []any{"username"}, Message: "expected string"},
	{Code: zod.IssueInvalidType, Path: []any{"favoriteNumbers", 1}, Message: "expected number"},
	{Code: zod.IssueInvalidType, Path: []any{"nesting", "a"}, Message: "expected string"},
}}

fmt.Printf("%#v\n", zod.Format(zerr))
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

tree := zod.Treeify(zerr)
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
zod.ToDotPath([]any{"user", "emails", 0, "address"})
// "user.emails[0].address"

zod.ToDotPath([]any{"weird.key"})
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

	"github.com/iKunalChhabra/go-zod"
)

func main() {
	schema := zod.Object(zod.Shape{
		"name": zod.String().Min(2),
		"tags": zod.Array(zod.String().Min(1)).Min(1),
	})

	_, err := schema.Parse(map[string]any{
		"name": "A",
		"tags": []any{"", "ok"},
	})
	zerr := err.(*zod.ZodError)

	fmt.Println("--- Prettify ---")
	fmt.Println(zod.Prettify(zerr))

	fmt.Println("--- Flatten ---")
	b, _ := json.MarshalIndent(zod.Flatten(zerr), "", "  ")
	fmt.Println(string(b))

	fmt.Println("--- Format ---")
	b, _ = json.MarshalIndent(zod.Format(zerr), "", "  ")
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
out, err := schema.ParseCtx(input, &zod.ParseCtx{ReportInput: true})
if err != nil {
	iss := err.(*zod.ZodError).Issues[0]
	fmt.Printf("bad value: %#v\n", iss.Input)
}
```

Avoid enabling this for passwords or tokens in production responses.

## Related

- [Error maps & locales](#/guide/error-maps) — customize `Message`  
- [Checks & refinements](#/guide/checks) — how checks produce issues  
- [HTTP error shapes](#/integrations/http-errors) — wiring formatters into APIs  
