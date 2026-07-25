# Gin integration (`zgin`)

First-class request validation for [Gin](https://gin-gonic.com).

```bash
go get github.com/iKunalChhabra/go-zod
# zgin is a subpackage of the same module
```

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/iKunalChhabra/go-zod"
    "github.com/iKunalChhabra/go-zod/zgin"
)
```

## BindJSON

Typed JSON body → `Schema[T]`:

```go
type CreateUser struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

var createUser = zod.ToStruct[CreateUser](zod.Object(zod.Shape{
    "name":  zod.String().Min(2),
    "email": zod.String().Email(),
}))

r.POST("/users", func(c *gin.Context) {
    body, ok := zgin.BindJSON(c, createUser)
    if !ok {
        return // 400 + Zod issues already written
    }
    c.JSON(200, body)
})
```

On failure: writes **400** with the default issues JSON shape and returns `(zero, false)`.

## BindJSONAny

Same as `BindJSON` for untyped `AnySchemaLike` (typical `Object` → `map[string]any`):

```go
body, ok := zgin.BindJSONAny(c, userSchema)
```

## BindQuery

URL query → schema. Values are coerced with `CoerceQueryValues` (single values unwrapped to `string`; multi-values stay `[]string`). Pair with `zod.Coerce.*` for numbers/bools:

```go
var listQuery = zod.ToStruct[ListQuery](zod.Object(zod.Shape{
    "page":  zod.Coerce.Number().Gte(1),
    "limit": zod.Default(zod.Coerce.Number().Gte(1).Lte(100), float64(20)),
    "q":     zod.Optional(zod.String()),
}))

r.GET("/items", func(c *gin.Context) {
    q, ok := zgin.BindQuery(c, listQuery)
    if !ok {
        return
    }
    // ...
})
```

## BindURI

Gin path params → schema. All values are **strings**; use `zod.Coerce.*` for numeric/bool fields:

```go
var idParam = zod.ToStruct[IDParam](zod.Object(zod.Shape{
    "id": zod.Coerce.Number().Int(), // or String().UUID()
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

## Validate + Get

Middleware that parses the JSON body, stores the result, and continues:

```go
const ContextKey = "zod:value" // zgin.ContextKey

r.POST("/users", zgin.Validate(userSchema), func(c *gin.Context) {
    body, ok := zgin.Get(c)
    if !ok {
        return
    }
    c.JSON(200, body)
})
```

`Validate` uses `BindJSONAny` under the hood. On failure it aborts; on success it `c.Set(zgin.ContextKey, v)` and calls `c.Next()`.

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
| `FormatFlatten` | `formErrors` + `fieldErrors` via `zod.Flatten` |
| `FormatTree` | nested tree via `zod.Treeify` |
| `FormatPretty` | string via `zod.Prettify` |

See [HTTP error shapes](#/integrations/http-errors) for JSON examples.

## Full router example

```go
package main

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/iKunalChhabra/go-zod"
    "github.com/iKunalChhabra/go-zod/zgin"
)

type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   float64 `json:"age"`
}

var userShape = zod.Object(zod.Shape{
    "name":  zod.String().Min(2).Max(100),
    "email": zod.String().Email(),
    "age":   zod.Optional(zod.Int().Gte(0).Lt(150)),
})

var createUser = zod.ToStruct[User](userShape)

var listQuery = zod.Object(zod.Shape{
    "page":  zod.Default(zod.Coerce.Number().Gte(1), float64(1)),
    "limit": zod.Default(zod.Coerce.Number().Gte(1).Lte(100), float64(20)),
})

var idURI = zod.Object(zod.Shape{
    "id": zod.String().UUID(),
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
        q, ok := zgin.BindQuery(c, zod.ToStruct[struct {
            Page  float64 `json:"page"`
            Limit float64 `json:"limit"`
        }](listQuery))
        if !ok {
            return
        }
        c.JSON(http.StatusOK, gin.H{"page": q.Page, "limit": q.Limit})
    })

    r.GET("/users/:id", func(c *gin.Context) {
        p, ok := zgin.BindURI(c, zod.ToStruct[struct {
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
    zgin.AbortWithError(c, err.(*zod.ZodError), zgin.Options{
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
| `ContextKey` | `"zod:value"` |
| `AbortWithError` | Write Zod-shaped error |
| `Options` / `ErrorFormat` | Status + renderer |
| `CoerceQueryValues` | `url.Values` → `map[string]any` |
