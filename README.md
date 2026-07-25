# go-z

[![Test](https://github.com/iKunalChhabra/go-z/actions/workflows/test.yml/badge.svg)](https://github.com/iKunalChhabra/go-z/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/iKunalChhabra/go-z/z.svg)](https://pkg.go.dev/github.com/iKunalChhabra/go-z/z)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

Schema-first validation for Go. Define a schema once, parse anything into it, and get
structured errors back — with the ergonomics of a fluent schema API and the
performance of hand-written Go.

> **Inspired by [Zod](https://github.com/colinhacks/zod), not affiliated with it.**
> go-z is an independent project, not endorsed by or sponsored by Zod or its
> authors. Portions are derived from Zod under the MIT licence — see [NOTICE](./NOTICE).

**[Documentation](https://ikunalchhabra.github.io/go-z/)** ·
[Quickstart](https://ikunalchhabra.github.io/go-z/#/guide/quickstart) ·
[API reference](https://ikunalchhabra.github.io/go-z/#/reference/cheatsheet) ·
[Benchmarks](./BENCHMARKS.md)

## Install

```bash
go get github.com/iKunalChhabra/go-z/z
```

Requires **Go 1.26+**.

The core package has a single dependency, [`golang.org/x/text`](https://pkg.go.dev/golang.org/x/text),
used for Unicode NFC normalisation in `Normalize()`. Importing the core downloads
nothing else and compiles nothing else. The optional `zgin` package additionally
needs [Gin](https://github.com/gin-gonic/gin); because it lives in this module,
Gin appears in the module graph even if you never import it, but it is never
downloaded, compiled, or linked unless you do.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/iKunalChhabra/go-z/z"
)

func main() {
	user := z.Object(z.Shape{
		"name":  z.String().Min(2).Max(100),
		"email": z.String().Email(),
		"age":   z.Int().Gte(0).Lt(150).Optional(),
	})

	data, err := user.Parse(map[string]any{
		"name":  "Ada",
		"email": "ada@example.com",
	})
	if err != nil {
		zerr, _ := z.AsError(err)
		fmt.Println(z.Prettify(zerr))
		return
	}
	fmt.Println(data) // map[email:ada@example.com name:Ada]
}
```

The package is named `z` and lives at `/z`, so a plain import already reads the
way you would want — no alias required:

```go
import "github.com/iKunalChhabra/go-z/z"

z.String().Min(5).Email()
```

## Features

- **Fluent schema API.** `z.String().Min(5).Email()`, `z.Object(z.Shape{…})`,
  `Optional` / `Nullable` / `Default` / `Catch` / `Pipe` / `Transform` / `Refine`.
- **Typed edges.** `String().Optional().Parse(v)` returns `(*string, error)`;
  `String.Default("x").Parse(v)` returns `(string, error)`. Generic wrappers keep
  the inner type instead of collapsing to `any`.
- **Structured errors.** Eleven issue codes with paths and stable JSON field
  names, plus `Flatten` / `Format` / `Treeify` / `Prettify`.
- **Bidirectional codecs.** `z.Decode` / `z.Encode` with direction-aware defaults.
- **JSON Schema export.** `z.ToJSONSchema` for OpenAPI and client-side validators.
- **i18n.** Error maps and seven locales (`en es fr de ja pt zh`).
- **Concurrency-safe.** Schemas are immutable after construction; `Parse` is
  lock-free and `-race` clean. `ParseParallelSlice` fans large slices across cores.
- **Gin integration.** `zgin.Validate`, `zgin.BindJSON`, typed `zgin.GetAs[T]`.

## Usage

### Safe parsing

```go
res := z.String().Email().SafeParse("nope")
if !res.Success {
	fmt.Println(res.Error.Issues[0].Code) // invalid_format
}
```

### Objects, unions, recursion

```go
var Category z.AnySchemaLike
Category = z.Lazy(func() z.AnySchemaLike {
	return z.Object(z.Shape{
		"name":     z.String().Min(1),
		"children": z.Array(Category).Default([]any{}),
	})
})

userOrGuest := z.DiscriminatedUnion("role", []z.AnySchemaLike{
	z.Object(z.Shape{"role": z.Literal("admin"), "perms": z.Array(z.String())}),
	z.Object(z.Shape{"role": z.Literal("guest"), "session": z.String().UUID()}),
})
```

### Structs

```go
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

parsed, err := z.ToStruct[User](user).Parse(input) // parsed is a User
```

### Codecs

```go
isoDate := z.Codec(z.String().ISODateTime(), z.Time(), z.CodecTx{
	Decode: func(v any, _ *z.RefinementCtx) (any, error) {
		return time.Parse(time.RFC3339Nano, v.(string))
	},
	Encode: func(v any, _ *z.RefinementCtx) (any, error) {
		return v.(time.Time).UTC.Format(time.RFC3339Nano), nil
	},
})

t, _ := z.Decode(isoDate, "2024-01-15T10:30:00Z") // time.Time
s, _ := z.Encode(isoDate, t)                      // ISO string
```

### Gin

```go
import "github.com/iKunalChhabra/go-z/zgin"

r.POST("/users", zgin.Validate(user), func(c *gin.Context) {
	body, _ := zgin.Get(c) // already parsed and validated
	c.JSON(200, body)
})
```

Failed validation writes structured issues automatically:

```json
{"success":false,"error":{"issues":[{"code":"too_small","path":["name"],"message":"Too small: expected string to have >=2 characters"}]}}
```

`Flatten`, `Treeify`, and `Prettify` renderers are available via `zgin.Options`.

### Concurrency

```go
schema := z.String().Email() // build once, share freely
go func() { schema.Parse(a) }()
go func() { schema.Parse(b) }()

out, err := z.ParseParallelSlice(ctx, itemSchema, items, z.ParallelOpts{})
```

## Performance

4-core Xeon, Go 1.26, median of six runs. Full methodology and tables in
[BENCHMARKS.md](./BENCHMARKS.md).

| Scenario | go-z | go-playground/validator | Oudwins/zog |
|---|---:|---:|---:|
| Flat object | **416 ns** | 607 ns | 1295 ns |
| Nested object | **978 ns** | 1090 ns | 2783 ns |
| String formats (email + uuid + url) | **890 ns** | 1099 ns | 1810 ns |
| Array of 10k (parallel) | **2.75 ms** | 6.09 ms | 12.9 ms |

Email, UUID, and the ISO date/time formats use hand-written matchers rather than
backtracking regexes; each is differential-tested against the regex it replaced over
hundreds of thousands of random inputs. Building structured issues makes the
**failure** path slower than tag-based validators — see BENCHMARKS.md.

## Design notes

- **Untyped core, typed edge.** The engine runs on `any`; `Schema[T]` is
  the generic boundary. `Optional` and `Nullable` yield `*T` (nil means absent or
  null); every other wrapper yields `T`. Each wrapper has a type-erased constructor
  for heterogeneous containers (`Optional(anySchema)`) and a typed one (`OptionalOf`,
  `DefaultOf`, `RefineOf`, …) that works with every schema type.
- **JSON model first.** Objects produce `map[string]any` and arrays `[]any`; use
  `ToStruct[T]` when you want a struct. Numeric schemas produce the Go type they
  are named after — `Int` an `int`, `Uint32` a `uint32`, `Number` a `float64` —
  converting the incoming JSON number only when the conversion is exact.
- **`Missing` is not `nil`.** `Missing` means an absent key (JS `undefined`); `nil`
  is JSON `null`. `Optional` accepts Missing, `Nullable` accepts nil, `Nullish` both.
- **Params are checked at definition time.** An unsupported params type panics while
  the schema is built — at startup, never during request handling.
- **Object field order.** `Object(Shape)` reports issues in sorted key order because
  Go maps are unordered; `ObjectOrdered([]Field{…})` preserves definition order.

## Project layout

Every package is a directory; none of them is special.

```
z/               core package — import "github.com/iKunalChhabra/go-z/z"
  schema_*.go      schema types (string, number, object, union, codec, …)
  checks_*.go      composable checks
  fluent*.go       mid-chain Optional/Default/Refine on concrete schemas
  matchers.go      hand-written format matchers
  errorutils.go    Flatten / Format / Treeify / Prettify
  jsonschema.go    ToJSONSchema
  locale_*.go      i18n error maps
  parallel.go      ParseParallelSlice
  tostruct.go      cached reflect decode
zgin/            Gin binding and middleware
bench/           comparative benchmarks (separate module)
docs/            documentation site
```

## Status

Implemented: primitives and string formats, objects and collections, unions / xor /
discriminated unions / intersection / lazy, wrappers, codecs, `ToJSONSchema`,
template literals, coercion, error utilities, seven locales, Gin, struct binding,
and parallel parsing. Behavioural parity is tracked in `z/parity_*_test.go` — see
[PARITY.md](./PARITY.md).

Not implemented: `fromJSONSchema`, and the JavaScript-only surface (`z.function()`,
`z.promise()`, `z.symbol()`, `z.file()`). Async parsing is unnecessary in Go.

## Contributing

Issues and pull requests are welcome. Before submitting:

```bash
go test -race./...
go vet./...
gofmt -l.
```

## Author

**Kunal Chhabra** ([@iKunalChhabra](https://github.com/iKunalChhabra))

## Licence and attribution

MIT — Copyright (c) 2026 Kunal Chhabra. See [LICENSE](./LICENSE).

Portions of this project are derived from [Zod](https://github.com/colinhacks/zod)
(MIT, Copyright (c) 2025 Colin McDonnell): string-format patterns, locale message
text, the issue taxonomy, and behavioural test cases ported from Zod's test suite.
Zod's licence is reproduced in full in [NOTICE](./NOTICE).
