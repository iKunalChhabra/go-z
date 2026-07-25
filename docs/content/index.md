<div class="hero">
<p class="eyebrow">Zod v4 · native Go · Gin-ready</p>
<h1>go-zod</h1>
<p class="lede">Schema-first validation for Go — same patterns as Zod, same issue taxonomy, fluent API, built for concurrency.</p>
<p class="hero-actions">
<a class="btn btn-primary" href="#/guide/installation">Get started</a>
<a class="btn" href="#/guide/quickstart">Quickstart</a>
<a class="btn" href="#/integrations/gin">Gin guide</a>
</p>
</div>

go-zod is not a thin wrapper around struct tags. It ports Zod’s actual architecture: a `ParsePayload` that accumulates issues, composable checks with `when` / `abort` / `continue`, eleven issue codes byte-compatible with Zod’s JSON, and an error-map chain that resolves messages from check → parse → custom → locale.

```go
package main

import (
	"fmt"

	"github.com/iKunalChhabra/go-zod"
)

func main() {
	user := zod.Object(zod.Shape{
		"name":  zod.String().Min(2).Max(100),
		"email": zod.String().Email(),
		"age":   zod.Optional(zod.Int().Gte(0).Lt(150)),
	})

	data, err := user.Parse(map[string]any{
		"name":  "Ada",
		"email": "ada@example.com",
	})
	if err != nil {
		fmt.Println(zod.Prettify(err.(*zod.ZodError)))
		return
	}
	fmt.Println(data)
}
```

:::tip Install in one line
```bash
go get github.com/iKunalChhabra/go-zod
```
Requires **Go 1.22+**. Gin users also pull in `github.com/iKunalChhabra/go-zod/zgin` (optional).
:::

## Why teams pick go-zod

- **Zod-shaped API** — `String().Min(5).Email()`, `Object(Shape{...})`, `Optional` / `Nullable` / `Default` / `Catch`
- **Accumulate, don’t abort early** — issues collect through containers; paths are prefixed exactly like Zod
- **11 issue codes** — `invalid_type`, `too_small`, `too_big`, `invalid_format`, and the rest match Zod v4 JSON
- **Locales** — `en`, `es`, `fr`, `de`, `ja`, `pt`, `zh` out of the box
- **Concurrency-safe** — schemas are immutable after construction; `Parse` is lock-free and `-race` clean
- **Gin-ready** — `zgin.Validate` / `zgin.BindJSONAny` for middleware and one-liner handlers
- **Parallel arrays** — `ParseParallelSlice` for large lists (~2.5× faster on multi-core vs sequential)
- **Struct edge** — `ToStruct[T]` when you want typed Go structs after JSON-shaped parse

## Start here

| Page | What you’ll learn |
|---|---|
| [Installation](#/guide/installation) | `go get`, module path, verify with a tiny `main` |
| [Quickstart](#/guide/quickstart) | Object schema → `Parse` / `SafeParse` → errors → Gin teaser |
| [Why go-zod?](#/guide/why) | Architecture: payload, checks, issue codes vs struct tags |
| [Comparison](#/guide/comparison) | Honest table vs validator, ozzo, zog, and TypeScript Zod |

## Core concepts

| Page | What you’ll learn |
|---|---|
| [Schemas & parsing](#/guide/parsing) | `Parse`, `MustParse`, `SafeParse`, `ParseCtx`, immutability |
| [Issues & ZodError](#/guide/errors) | Issue fields, `Flatten` / `Format` / `Treeify` / `Prettify` |
| [Error maps & locales](#/guide/error-maps) | Custom messages, `Configure`, `Locale("es")` |
| [Checks & refinements](#/guide/checks) | `Check`, `Abort`, `When`, `OnAttach` |
| [Missing vs nil](#/guide/missing-nil) | Zod’s `undefined` vs JSON `null` in Go |
| [Immutability & concurrency](#/guide/concurrency) | Lock-free parse, pooling, sharing schemas safely |

## A taste of the fluent API

```go
schema := zod.String().Min(5).Email()

s, err := schema.Parse("hi@example.com")
// err is *zod.ZodError when validation fails

res := schema.SafeParse("nope")
if !res.Success {
	fmt.Println(res.Error.Issues[0].Code) // invalid_format
}
```

Objects produce `map[string]any`; arrays produce `[]any`. That is intentional — go-zod is **JSON-model first**, then typed at the edges with generics (`Schema[T]`) and optional `ToStruct[T]`.

```go
parsed, err := zod.ToStruct[User](userSchema).Parse(input)
```

## Gin in one breath

```go
import "github.com/iKunalChhabra/go-zod/zgin"

r.POST("/users", zgin.Validate(user), func(c *gin.Context) {
	body, _ := zgin.Get(c) // already parsed & validated
	c.JSON(200, body)
})
```

:::info Same soul as Zod
If you know Zod, you already know go-zod. The names, issue codes, and error utilities (`flattenError` → `Flatten`, `prettifyError` → `Prettify`) are deliberate ports — not vague inspiration.
:::

## What’s next

1. Follow the [Quickstart](#/guide/quickstart) and validate a real request body.
2. Read [Missing vs nil](#/guide/missing-nil) before you model optional fields — Go’s single `nil` is not Zod’s `undefined` / `null` pair.
3. Skim [Comparison](#/guide/comparison) if you’re migrating from `go-playground/validator` or TypeScript Zod.

Welcome. Let’s make invalid input someone else’s problem.
