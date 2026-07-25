# Quickstart

Build a real object schema, parse success and failure paths, use defaults, and peek at Gin. About ten minutes from zero to a validated request body.

## 1. Define an object schema

```go
package main

import (
	"fmt"

	"github.com/iKunalChhabra/go-zod"
)

var User = zod.Object(zod.Shape{
	"name":  zod.String().Min(2).Max(100),
	"email": zod.String().Email(),
	"age":   zod.Optional(zod.Int().Gte(0).Lt(150)),
	"tags":  zod.Default(zod.Array(zod.String()).Max(10), []any{}),
})
```

What this says:

| Key | Schema | Meaning |
|---|---|---|
| `name` | `String().Min(2).Max(100)` | Required string, length bounds |
| `email` | `String().Email()` | Required, email format check |
| `age` | `Optional(Int()...)` | Key may be absent (`Missing`); if present, must be an int in range |
| `tags` | `Default(Array(...), []any{})` | Absent key → `[]any{}`; if present, array of strings |

Objects parse into `map[string]any`. That matches JSON and keeps the core untyped like Zod’s runtime.

## 2. Parse (throws… as a Go error)

```go
func main() {
	input := map[string]any{
		"name":  "Ada Lovelace",
		"email": "ada@example.com",
		// age omitted — Optional
		// tags omitted — Default fills []any{}
	}

	data, err := User.Parse(input)
	if err != nil {
		zerr := err.(*zod.ZodError)
		fmt.Println(zod.Prettify(zerr))
		return
	}

	fmt.Printf("%#v\n", data)
	// map[string]interface{}{"email":"ada@example.com", "name":"Ada Lovelace", "tags":[]interface{}{}}
}
```

`Parse` returns `(T, error)`. On failure the error is always `*zod.ZodError` with a slice of structured `Issue` values.

## 3. SafeParse (no error value)

Prefer a result object when you don’t want to type-assert the error:

```go
res := User.SafeParse(map[string]any{
	"name":  "A", // too short
	"email": "not-an-email",
})

if !res.Success {
	for _, iss := range res.Error.Issues {
		fmt.Printf("%s at %v: %s\n", iss.Code, iss.Path, iss.Message)
	}
	return
}

fmt.Println(res.Data)
```

Example issue lines:

```text
too_small at [name]: Too small: expected string to have >=2 characters
invalid_format at [email]: Invalid email address
```

:::tip Parse vs SafeParse
Use `Parse` when you’re writing `if err != nil` pipelines. Use `SafeParse` when you want Zod’s `{ success, data, error }` shape in one value — handy in handlers that branch on `Success`.
:::

## 4. Handle ZodError properly

```go
data, err := User.Parse(badInput)
if err != nil {
	zerr, ok := err.(*zod.ZodError)
	if !ok {
		panic(err) // should not happen for schema.Parse
	}

	// Human-readable CLI / logs
	fmt.Println(zod.Prettify(zerr))
	// ✖ Too small: expected string to have >=2 characters
	//   → at name
	// ✖ Invalid email address
	//   → at email

	// Form / field maps for UIs
	flat := zod.Flatten(zerr)
	_ = flat.FieldErrors["name"]

	// Nested tree for complex forms
	tree := zod.Treeify(zerr)
	_ = tree
}
```

See [Issues & ZodError](#/guide/errors) for `Format`, `Treeify`, `ToDotPath`, and issue field reference.

## 5. Optional, Default, and friends

```go
// Absent key OK; JSON null (nil) is NOT OK
opt := zod.Optional(zod.String())

// JSON null OK; absent key still fails on object fields unless also Optional
nul := zod.Nullable(zod.String())

// Absent OR null OR string
both := zod.Nullish(zod.String())

// Absent → substitute without re-validating the default through the inner schema
withDef := zod.Default(zod.String(), "anonymous")

// On parse failure, catch and return a fallback
caught := zod.Catch(zod.String().Email(), "fallback@example.com")
```

:::warn Missing ≠ nil
In go-zod, `zod.Missing` is Zod’s `undefined` (key absent). Go’s `nil` is JSON `null`. Choosing `Optional` vs `Nullable` vs `Nullish` matters — full guide: [Missing vs nil](#/guide/missing-nil).
:::

## 6. Coerce query/form strings

```go
n, err := zod.Coerce.Number().Parse("42")
// n == float64(42)

b, err := zod.Coerce.Bool().Parse("true")
// b == true

s, err := zod.Coerce.String().Parse(99)
// s == "99"
```

Useful for query params and form posts where everything arrives as strings.

## 7. Gin one-liner teaser

```go
import (
	"github.com/gin-gonic/gin"
	"github.com/iKunalChhabra/go-zod/zgin"
)

func mount(r *gin.Engine) {
	r.POST("/users", zgin.Validate(User), func(c *gin.Context) {
		body, _ := zgin.Get(c)
		c.JSON(200, body)
	})

	// Or without middleware:
	r.POST("/users2", func(c *gin.Context) {
		body, ok := zgin.BindJSONAny(c, User)
		if !ok {
			return // 400 + Zod issues already written
		}
		c.JSON(200, body)
	})
}
```

Default error body shape:

```json
{
  "success": false,
  "error": {
    "issues": [
      {
        "code": "too_small",
        "path": ["name"],
        "message": "Too small: expected string to have >=2 characters"
      }
    ]
  }
}
```

## Full runnable example

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/iKunalChhabra/go-zod"
)

func main() {
	schema := zod.Object(zod.Shape{
		"name":  zod.String().Min(2),
		"email": zod.String().Email(),
		"role":  zod.Default(zod.String(), "user"),
	})

	raw := []byte(`{"name":"Ada","email":"ada@example.com"}`)
	var input any
	if err := json.Unmarshal(raw, &input); err != nil {
		panic(err)
	}

	out, err := schema.Parse(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, zod.Prettify(err.(*zod.ZodError)))
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
```

Output:

```json
{
  "email": "ada@example.com",
  "name": "Ada",
  "role": "user"
}
```

## Where to go next

- [Schemas & parsing](#/guide/parsing) — `MustParse`, `ParseCtx`, `Schema[T]`, fluent clones
- [Error maps & locales](#/guide/error-maps) — custom messages and Spanish / Japanese / …
- [Checks & refinements](#/guide/checks) — how `Min` / `Email` attach under the hood
- [Comparison](#/guide/comparison) — when to choose go-zod over tag validators
