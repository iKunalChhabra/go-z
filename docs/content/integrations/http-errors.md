# HTTP error shapes

How `*z.Error` is rendered for APIs — used by `zgin.AbortWithError` and available as pure helpers.

## Issues (default)

`zgin.FormatIssues` / raw `zerr.Issues`:

```json
{
  "success": false,
  "error": {
    "issues": [
      {
        "code": "too_small",
        "path": ["name"],
        "message": "Too small: expected string to have >=2 characters",
        "origin": "string",
        "minimum": 2,
        "inclusive": true
      }
    ]
  }
}
```

`Error.Error` stringifies issues as indented JSON (`Error.message`).

## Flatten

```go
flat:= z.Flatten(zerr)
// zgin.Options{Format: zgin.FormatFlatten}
```

```json
{
  "success": false,
  "error": {
    "formErrors": ["Invalid input"],
    "fieldErrors": {
      "email": ["Invalid email address"],
      "age": ["Too big: expected number to be <=150"]
    }
  }
}
```

| Field | Meaning |
|---|---|
| `formErrors` | Issues with empty `path` |
| `fieldErrors` | Grouped by **first** path segment (string key) |

Custom mapper: `z.FlattenMap(err, func(iss Issue) U)`.

:::tip Forms
Flatten is ideal for HTML / SPA forms that bind errors to top-level field names.
:::

## Tree

```go
tree:= z.Treeify(zerr)
// zgin.Options{Format: zgin.FormatTree}
```

```json
{
  "success": false,
  "error": {
    "errors": [],
    "properties": {
      "address": {
        "errors": [],
        "properties": {
          "zip": {
            "errors": ["Too small: expected string to have >=5 characters"]
          }
        }
      },
      "tags": {
        "errors": [],
        "items": [
          null,
          { "errors": ["Invalid input: expected string, received number"] }
        ]
      }
    }
  }
}
```

String keys → `properties`; numeric segments → `items` (sparse array). Nested `invalid_union` / `invalid_key` / `invalid_element` issues are traversed too.

Custom mapper: `z.TreeifyMap`.

## Pretty

```go
s:= z.Prettify(zerr)
// zgin.Options{Format: zgin.FormatPretty}
```

HTTP body:

```json
{
  "success": false,
  "error": "✖ Too small: expected string to have >=2 characters\n  → at name\n✖ Invalid email address\n  → at email"
}
```

Issues are sorted by path length, then formatted as:

```text
✖ <message>
  → at <dotPath>
```

`z.ToDotPath(path)` produces paths like `address.zip`, `tags[1]`, `["weird.key"]`.

## Format (nested `_errors`)

Also available (not a zgin format constant):

```go
m:= z.Format(zerr)
```

```json
{
  "_errors": [],
  "name": { "_errors": ["Too small: …"] },
  "address": {
    "_errors": [],
    "zip": { "_errors": ["…"] }
  }
}
```

## Choosing a format

| Format | Best for |
|---|---|
| Issues | schema-compatible clients, debugging |
| Flatten | Simple form field binding |
| Tree | Nested objects / arrays in UI |
| Pretty | Logs, human-readable 400 pages |

```go
zgin.AbortWithError(c, zerr, zgin.Options{
    Status: http.StatusUnprocessableEntity,
    Format: zgin.FormatTree,
})
```

## Helpers

```go
func Flatten(err *Error) FlattenedError
func FlattenMap[U any](err *Error, mapper func(Issue) U) FlattenedErrorU[U]
func Treeify(err *Error) *ErrorTree
func TreeifyMap[U any](err *Error, mapper func(Issue) U) *ErrorTreeU[U]
func Format(err *Error) map[string]any
func FormatMap[U any](err *Error, mapper func(Issue) U) map[string]any
func Prettify(err *Error) string
func ToDotPath(path []any) string
```
