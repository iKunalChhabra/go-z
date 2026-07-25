<div class="hero">
<p class="eyebrow">schema-first · native Go · Gin-ready</p>
<h1>go-z</h1>
<p class="lede">Schema-first validation for Go — fluent schemas, structured issues, built for concurrency.</p>
<p class="hero-actions">
<a class="btn btn-primary" href="#/guide/installation">Get started</a>
<a class="btn" href="#/guide/quickstart">Quickstart</a>
<a class="btn" href="#/integrations/gin">Gin guide</a>
</p>
</div>

go-z is not a thin wrapper around struct tags. It is built on a real parse pipeline: a `ParsePayload` that accumulates issues, composable checks with `when` / `abort` / `continue`, eleven issue codes, and an error-map chain that resolves messages from check → parse → custom → locale.

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
		"age":   z.Optional(z.Int().Gte(0).Lt(150)),
	})

	data, err := user.Parse(map[string]any{
		"name":  "Ada",
		"email": "ada@example.com",
	})
	if err != nil {
		fmt.Println(z.Prettify(err.(*z.Error)))
		return
	}
	fmt.Println(data)
}
```

:::tip Install in one line
```bash
go get github.com/iKunalChhabra/go-z/z
```
Requires **Go 1.26+**. Gin users install the separate `github.com/iKunalChhabra/go-z/zgin` module.
:::

## Why teams pick go-z

- **Fluent schema API** — `String.Min(5).Email`, `Object(Shape{...})`, `Optional` / `Nullable` / `Default` / `Catch`
- **Accumulate, don’t abort early** — issues collect through containers; container paths are prefixed automatically
- **11 issue codes** — `invalid_type`, `too_small`, `too_big`, `invalid_format`, and the rest, with stable JSON field names
- **Locales** — `en`, `es`, `fr`, `de`, `ja`, `pt`, `zh` out of the box
- **Concurrency-safe** — schemas are immutable after construction; `Parse` is lock-free and `-race` clean
- **Gin-ready** — `zgin.Validate` / `zgin.BindJSONAny` for middleware and one-liner handlers
- **Parallel arrays** — `ParseParallelSlice` for large lists (~2.2× faster at 10k elements on 4 cores)
- **Struct edge** — `ToStruct[T]` when you want typed Go structs after JSON-shaped parse

## Start here

| Page | What you’ll learn |
|---|---|
| [Installation](#/guide/installation) | `go get`, module path, verify with a tiny `main` |
| [Quickstart](#/guide/quickstart) | Object schema → `Parse` / `SafeParse` → errors → Gin teaser |
| [Why go-z?](#/guide/why) | Architecture: payload, checks, issue codes vs struct tags |
| [Comparison](#/guide/comparison) | Honest table vs validator, ozzo, and zog |

## Core concepts

| Page | What you’ll learn |
|---|---|
| [Schemas & parsing](#/guide/parsing) | `Parse`, `MustParse`, `SafeParse`, `ParseCtx`, immutability |
| [Issues & errors](#/guide/errors) | Issue fields, `Flatten` / `Format` / `Treeify` / `Prettify` |
| [Error maps & locales](#/guide/error-maps) | Custom messages, `Configure`, `Locale("es")` |
| [Checks & refinements](#/guide/checks) | `Check`, `Abort`, `When`, `OnAttach` |
| [Missing vs nil](#/guide/missing-nil) | JS `undefined` vs JSON `null` in Go |
| [Immutability & concurrency](#/guide/concurrency) | Lock-free parse, pooling, sharing schemas safely |

## A taste of the fluent API

```go
schema := z.String().Min(5).Email()

s, err := schema.Parse("hi@example.com")
// err is *z.Error when validation fails

res := schema.SafeParse("nope")
if !res.Success {
	fmt.Println(res.Error.Issues[0].Code) // invalid_format
}
```

Objects produce `map[string]any`; arrays produce `[]any`. That is intentional — go-z is **JSON-model first**, then typed at the edges with generics (`Schema[T]`) and optional `ToStruct[T]`.

```go
parsed, err := z.ToStruct[User](userSchema).Parse(input)
```

## Gin in one breath

```go
import "github.com/iKunalChhabra/go-z/zgin"

r.POST("/users", zgin.Validate(user), func(c *gin.Context) {
	body, _ := zgin.Get(c) // already parsed & validated
	c.JSON(200, body)
})
```

:::info Familiar by design
If you have used a schema-first validator in TypeScript, you already know go-z. The names, issue codes, and error utilities (`Flatten`, `Format`, `Treeify`, `Prettify`) follow the same model.
:::

## What’s next

1. Follow the [Quickstart](#/guide/quickstart) and validate a real request body.
2. Read [Missing vs nil](#/guide/missing-nil) before you model optional fields — Go’s single `nil` is not the `undefined` / `null` pair.
3. Skim [Comparison](#/guide/comparison) if you’re migrating from `go-playground/validator`.

Welcome. Let’s make invalid input someone else’s problem.

---

<p class="meta">go-z is MIT licensed, by <a href="https://github.com/iKunalChhabra">Kunal Chhabra</a>. Portions are derived from <a href="https://github.com/colinhacks/zod">Zod</a> (MIT, © 2025 Colin McDonnell). This is an independent project, not affiliated with or endorsed by Zod.</p>
