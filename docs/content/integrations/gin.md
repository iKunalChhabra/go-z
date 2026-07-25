# Gin integration (`zgin`)

First-class request validation for [Gin](https://gin-gonic.com).

```bash
# zgin is a separate module, so Gin only enters your build if you ask for it
go get github.com/iKunalChhabra/go-z/zgin
```

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/iKunalChhabra/go-z/z"
    "github.com/iKunalChhabra/go-z/zgin"
)
```

## BindJSON

Typed JSON body → `Schema[T]`:

```go
type CreateUser struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

var createUser = z.ToStruct[CreateUser](z.Object(z.Shape{
    "name":  z.String().Min(2),
    "email": z.String().Email(),
}))

r.POST("/users", func(c *gin.Context) {
    body, ok := zgin.BindJSON(c, createUser)
    if !ok {
        return // 400 + issues already written
    }
    c.JSON(200, body)
})
```

On failure: writes the default issues JSON shape and returns `(zero, false)`.

### Body handling

The JSON binders harden the read before the schema ever sees it:

| Behaviour | Default | Response when violated |
|---|---|---|
| Body size limit | `zgin.DefaultMaxBodyBytes` (1 MiB) | **413** `request body too large` |
| `Content-Type` must be a JSON type | on (an absent header is accepted) | **415** `unsupported Content-Type` |
| Trailing data after the JSON value | rejected | **400** `unexpected data after the JSON value` |
| Integers above 2^53 | kept exact as `int64` | — |

Every failure uses the same issue shape as a validation failure, so a client parses one format.

```go
body, ok := zgin.BindJSON(c, createUser, zgin.BindOptions{
    MaxBodyBytes:        4 << 20, // 4 MiB; negative means unlimited
    AllowAnyContentType: false,
    AllowTrailingData:   false,
})
```

`Validate` and `ValidateToStruct` take the same options.

:::warn The body is read once
Reading `c.Request.Body` yourself after a bind returns nothing: an HTTP body is a
one-shot reader. The binders cache the *decoded* value on the context, so a second
bind — chaining two `Validate` middlewares, or binding again in the handler — works
and re-validates the same data. If a handler needs the raw bytes as well, read and
buffer them before the bind (`c.Request.Body = io.NopCloser(bytes.NewReader(raw))`).
:::

:::info Exact integers
The body is decoded with `Decoder.UseNumber`, so a JSON integer larger than 2^53
reaches the schema as an `int64` instead of a rounded `float64` — which matters for
database identifiers. Numbers that fit exactly still arrive as `float64`, so
`z.Number()`, `z.Literal(42)` and discriminated unions behave as before.
:::

## BindJSONAny

Same as `BindJSON` for untyped `AnySchemaLike` (typical `Object` → `map[string]any`):

```go
body, ok := zgin.BindJSONAny(c, userSchema)
```

## BindQuery

URL query → schema. Values are coerced with `CoerceQueryValues` (single values unwrapped to `string`; multi-values stay `[]string`). Pair with `z.Coerce.*` for numbers/bools:

```go
var listQuery = z.ToStruct[ListQuery](z.Object(z.Shape{
    "page":  z.Coerce.Number().Gte(1),
    "limit": z.Default(z.Coerce.Number().Gte(1).Lte(100), float64(20)),
    "q":     z.Optional(z.String()),
}))

r.GET("/items", func(c *gin.Context) {
    q, ok := zgin.BindQuery(c, listQuery)
    if !ok {
        return
    }
    // ...
})
```

:::info Single-value array parameters
`?tag=a` carries one value and `?tag=a&tag=b` carries two, so a schema expecting
an array would reject the first. `BindQuery` consults the schema: a field declared
as `Array`, `Set` or `Tuple` always receives a slice, whatever the query contained.
`zgin.CoerceQueryValuesFor(schema, values)` applies the same rule outside
`BindQuery`; `CoerceQueryValues` is the schema-unaware form.
:::

## BindURI

Gin path params → schema. All values are **strings**; use `z.Coerce.*` for numeric/bool fields:

```go
var idParam = z.ToStruct[IDParam](z.Object(z.Shape{
    "id": z.Int(z.Params{Coerce: true}), // or z.String().UUID()
}))

r.GET("/users/:id", func(c *gin.Context) {
    p, ok := zgin.BindURI(c, idParam)
    if !ok {
        return
    }
    _ = p
})
```

:::info CoerceQueryValues
`zgin.CoerceQueryValues(map[string][]string) map[string]any` is public if you need the same conversion outside BindQuery.
:::

## Validate + Get / GetAs

Middleware that parses the JSON body, stores the result, and continues:

```go
const ContextKey = "go-z:value" // zgin.ContextKey

r.POST("/users", zgin.Validate(userSchema), func(c *gin.Context) {
    body, ok := zgin.Get(c) // any
    if !ok {
        return
    }
    c.JSON(200, body)
})
```

Typed end-to-end with `ToStruct`:

```go
type User struct {
    Name  string  `json:"name"`
    Email string  `json:"email"`
    Age   float64 `json:"age"`
}

r.POST("/users", zgin.ValidateToStruct[User](userSchema), func(c *gin.Context) {
    user, ok := zgin.GetAs[User](c)
    if !ok {
        return
    }
    c.JSON(200, user)
})
```

`Validate` uses `BindJSONAny` under the hood. On failure it aborts; on success it `c.Set(zgin.ContextKey, v)` and calls `c.Next`.

## AbortWithError & Options

```go
zgin.AbortWithError(c, zerr, zgin.Options{
    Status: http.StatusUnprocessableEntity, // default 400
    Format: zgin.FormatFlatten,             // default FormatIssues
})
```

### ErrorFormat

| Constant | Body shape |
|---|---|
| `FormatIssues` (default) | `{"success":false,"error":{"issues":[...]}}` |
| `FormatFlatten` | `formErrors` + `fieldErrors` via `z.Flatten` |
| `FormatTree` | nested tree via `z.Treeify` |
| `FormatPretty` | string via `z.Prettify` |

See [HTTP error shapes](#/integrations/http-errors) for JSON examples.

## Full router example

```go
package main

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/iKunalChhabra/go-z/z"
    "github.com/iKunalChhabra/go-z/zgin"
)

type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   float64 `json:"age"`
}

var userShape = z.Object(z.Shape{
    "name":  z.String().Min(2).Max(100),
    "email": z.String().Email(),
    "age":   z.Optional(z.Int().Gte(0).Lt(150)),
})

var createUser = z.ToStruct[User](userShape)

var listQuery = z.Object(z.Shape{
    "page":  z.Default(z.Coerce.Number().Gte(1), float64(1)),
    "limit": z.Default(z.Coerce.Number().Gte(1).Lte(100), float64(20)),
})

var idURI = z.Object(z.Shape{
    "id": z.String().UUID(),
})

func main() {
    r := gin.Default()

    // Middleware style
    r.POST("/users", zgin.Validate(userShape), func(c *gin.Context) {
        body, _ := zgin.Get(c)
        c.JSON(http.StatusCreated, body)
    })

    // Typed bind style
    r.POST("/users/typed", func(c *gin.Context) {
        u, ok := zgin.BindJSON(c, createUser)
        if !ok {
            return
        }
        c.JSON(http.StatusCreated, u)
    })

    r.GET("/users", func(c *gin.Context) {
        q, ok := zgin.BindQuery(c, z.ToStruct[struct {
            Page  float64 `json:"page"`
            Limit float64 `json:"limit"`
        }](listQuery))
        if !ok {
            return
        }
        c.JSON(http.StatusOK, gin.H{"page": q.Page, "limit": q.Limit})
    })

    r.GET("/users/:id", func(c *gin.Context) {
        p, ok := zgin.BindURI(c, z.ToStruct[struct {
            ID string `json:"id"`
        }](idURI))
        if !ok {
            return
        }
        c.JSON(http.StatusOK, gin.H{"id": p.ID})
    })

    // Custom error format for a group
    api := r.Group("/api")
    api.POST("/import", func(c *gin.Context) {
        body, ok := zgin.BindJSONAny(c, userShape)
        if !ok {
            // BindJSONAny already aborted with FormatIssues.
            // For a custom format, parse manually:
            return
        }
        c.JSON(http.StatusOK, body)
    })

    _ = r.Run(":8080")
}
```

:::tip Manual parse + custom format
```go
v, err := schema.Parse(data)
if err != nil {
    zgin.AbortWithError(c, err.(*z.Error), zgin.Options{
        Format: zgin.FormatTree,
        Status: http.StatusUnprocessableEntity,
    })
    return
}
```
:::

## API surface

| Symbol | Role |
|---|---|
| `BindJSON[T]` | JSON body + typed schema |
| `BindJSONAny` | JSON body + `AnySchemaLike` |
| `BindQuery[T]` | Query string + coerce |
| `BindURI[T]` | Path params |
| `Validate` | Middleware |
| `Get` | Read middleware value |
| `ContextKey` | `"go-z:value"` |
| `AbortWithError` | Write schema-shaped error |
| `Options` / `ErrorFormat` | Status + renderer |
| `BindOptions` | Body limit, Content-Type policy, trailing data |
| `DefaultMaxBodyBytes` | 1 MiB body cap used when `BindOptions` is omitted |
| `CoerceQueryValues` | `url.Values` → `map[string]any` |
| `CoerceQueryValuesFor` | the same, with array fields always slices |
