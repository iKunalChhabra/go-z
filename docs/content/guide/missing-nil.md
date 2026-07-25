# Missing vs nil

JavaScript has two “empty” values: `undefined` (absent) and `null` (present, null). Go has one: `nil`. go-z restores distinction with an explicit sentinel — `z.Missing` — so optional keys and nullable fields behave correctly.

## The two empties

| Concept | JS | go-z | Typical JSON |
|---|---|---|---|
| Absent | `undefined` | `z.Missing` | key omitted from object |
| Null | `null` | `nil` | `"field": null` |
| Present value | `"ada"` / `42` | same | normal JSON value |

```go
z.IsMissing(z.Missing) // true
z.IsMissing(nil)         // false
z.IsMissing("")          // false
```

:::warn Never use nil for “key omitted”
If you pass `nil` into a field schema, that means JSON `null`, not absence. Object parsing is what injects `Missing` for omitted keys.
:::

## Optional vs Nullable vs Nullish

```go
// Optional — accepts Missing; rejects nil
opt:= z.Optional(z.String)

// Nullable — accepts nil; rejects Missing (as invalid for the inner type)
nul:= z.Nullable(z.String)

// Nullish — Optional(Nullable(...)) — accepts Missing OR nil OR string
both:= z.Nullish(z.String)
```

### Behavior table (standalone parse)

| Input | `String` | `Optional(String)` | `Nullable(String)` | `Nullish(String)` |
|---|---|---|---|---|
| `"hi"` | ✅ | ✅ | ✅ | ✅ |
| `Missing` | ❌ `invalid_type` | ✅ → `Missing` | ❌ | ✅ → `Missing` |
| `nil` | ❌ `invalid_type` | ❌ | ✅ → `nil` | ✅ → `nil` |
| `42` | ❌ | ❌ | ❌ | ❌ |

The arrows above describe the **raw JSON model**, which is what `ParseAny` returns and what object parsing passes around. The typed `Parse` on `Optional` / `Nullable` folds both `Missing` and `nil` into a nil `*T`:

```go
v, _:= z.String.Optional.Parse(z.Missing)   // (*string)(nil)
raw, _:= z.String.Optional.ParseAny(z.Missing) // z.Missing — sentinel preserved
```

```go
fmt.Println(z.Optional(z.String).SafeParse(z.Missing).Success) // true
fmt.Println(z.Optional(z.String).SafeParse(nil).Success)         // false

fmt.Println(z.Nullable(z.String).SafeParse(nil).Success)         // true
fmt.Println(z.Nullable(z.String).SafeParse(z.Missing).Success) // false

fmt.Println(z.Nullish(z.String).SafeParse(z.Missing).Success)  // true
fmt.Println(z.Nullish(z.String).SafeParse(nil).Success)          // true
```

## Object absent keys

When an object schema parses a `map[string]any`, **missing keys** are passed to field schemas as `Missing` (not as Go’s zero value, not as `nil`).

```go
schema:= z.Object(z.Shape{
	"name":  z.String,
	"nickname": z.Optional(z.String),
})

out, err:= schema.Parse(map[string]any{
	"name": "Ada",
	// nickname omitted
})
// err == nil
// out["name"] == "Ada"
// "nickname" key typically absent from output (optional / OptOut semantics)
```

Present but null:

```go
_, err:= schema.Parse(map[string]any{
	"name":     "Ada",
	"nickname": nil, // JSON null
})
// fails — Optional(String) does not accept nil
```

Allow both omit and null:

```go
schema:= z.Object(z.Shape{
	"name":     z.String,
	"nickname": z.Nullish(z.String),
})

schema.MustParse(map[string]any{"name": "Ada"})
schema.MustParse(map[string]any{"name": "Ada", "nickname": nil})
schema.MustParse(map[string]any{"name": "Ada", "nickname": "Addie"})
```

### OptIn / OptOut (internals)

Optional wrappers set `OptIn` / `OptOut` on schema internals so object parsing knows a key may be omitted. You rarely touch these flags directly — `Optional`, `Default`, and friends set them for you.

| Wrapper | Absent key on object | Output presence |
|---|---|---|
| required field | fails | always present if parse ok |
| `Optional` | ok | may be absent |
| `Default` | ok → substitute default | output present |
| `Nullable` alone | fails (Missing ≠ null) | present (maybe `nil`) |
| `Nullish` | ok | may be absent or `nil` |

## Default and Catch

### Default — fill Missing

```go
schema:= z.Object(z.Shape{
	"role": z.Default(z.String, "user"),
})

out, _:= schema.Parse(map[string]any{})
fmt.Println(out["role"]) // "user"
```

`Default` substitutes when input is `Missing` and does **not** re-run the default through the inner schema. Use `Prefault` when the default should be parsed/validated as input.

`nil` still fails a `Default(String,...)` field unless you also wrap with `Nullable`.

### Catch — recover from failures

```go
schema:= z.Catch(z.String.Email, "fallback@example.com")

out, err:= schema.Parse("not-an-email")
// err == nil, out == "fallback@example.com"
```

## Encoding from Go structs

`encoding/json` cannot express `Missing`. Common patterns:

```go
type PatchUser struct {
	Name  *string `json:"name,omitempty"`  // omitempty ≈ absent when nil pointer
	Bio   *string `json:"bio"`             // nil pointer → null if you marshal carefully
}
```

When building maps by hand for tests:

```go
input:= map[string]any{
	"name": "Ada",
	// omit nickname key entirely for Missing
}

inputWithNull:= map[string]any{
	"name":     "Ada",
	"nickname": nil, // null
}
```

After `json.Unmarshal`, omitted keys are simply not in the map — object schemas turn that into `Missing` for you. Explicit `null` becomes Go `nil`.

## NonOptional

Strip optionality — reject `Missing` after the inner schema runs:

```go
inner:= z.Optional(z.String)
req:= z.NonOptional(inner)

_, err:= req.Parse(z.Missing)
// invalid_type, expected "nonoptional"
```

## Decision guide

```text
Can the JSON key be omitted?
  └─ no  → plain schema (String, Int,...)
  └─ yes →
        Can the value be JSON null?
          ├─ no  → Optional(schema)
          ├─ yes → Nullish(schema)
          └─ always provide a fallback when omitted?
                → Default(schema, value)  (and Nullable/Nullish if null is also allowed)
```

:::info Mental model
Ask two questions for every field: “May it be absent?” and “May it be null?” In go-z that’s `.Optional` / `.Nullable` / `.Nullish` — just remember `Missing` is not `nil`.
:::

## Full example

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/iKunalChhabra/go-z/z"
)

func main {
	schema:= z.Object(z.Shape{
		"email": z.String.Email,
		"age":   z.Optional(z.Int.Gte(0)),
		"note":  z.Nullish(z.String.Max(200)),
		"role":  z.Default(z.String, "member"),
	})

	cases:= []string{
		`{"email":"a@b.co"}`,
		`{"email":"a@b.co","age":31}`,
		`{"email":"a@b.co","note":null}`,
		`{"email":"a@b.co","note":"hi"}`,
		`{"email":"a@b.co","age":null}`, // fails — age is Optional, not Nullable
	}

	for _, raw:= range cases {
		var input any
		_ = json.Unmarshal([]byte(raw), &input)
		out, err:= schema.Parse(input)
		if err != nil {
			fmt.Println(raw, "→", z.Prettify(err.(*z.Error)))
			continue
		}
		b, _:= json.Marshal(out)
		fmt.Println(raw, "→", string(b))
	}
}
```

## Related

- [Quickstart](#/guide/quickstart) — Optional / Default in a user object
- [Optional & Nullable](#/api/optional) — API details
- [Default, Prefault & Catch](#/api/defaults) — fallback wrappers
