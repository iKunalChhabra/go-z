# Cookbook

Copy-paste recipes for common go-zod patterns.

## 1. Signup form

```go
var Signup = zod.SuperRefine(zod.Object(zod.Shape{
    "email":           zod.String().Email(),
    "password":        zod.String().Min(8).Max(128),
    "confirmPassword": zod.String(),
    "acceptTerms":     zod.Literal(true),
}), func(v any, ctx *zod.RefinementCtx) {
    m := v.(map[string]any)
    if m["password"] != m["confirmPassword"] {
        ctx.AddIssue(zod.Issue{
            Code:    zod.IssueCustom,
            Message: "passwords must match",
            Path:    []any{"confirmPassword"},
        }.WithContinue())
    }
})

_, err := Signup.Parse(map[string]any{
    "email":           "ada@example.com",
    "password":        "correct-horse",
    "confirmPassword": "correct-horse",
    "acceptTerms":     true,
})
_ = err
```

:::tip Path on confirm
`Path: []any{"confirmPassword"}` lets `Flatten` / forms highlight the right field.
:::

## 2. Env config with Coerce

```go
import "os"

envSchema := zod.Object(zod.Shape{
    "PORT":     zod.Coerce.Number().Gte(1).Lte(65535),
    "DEBUG":    zod.Default(zod.Coerce.Bool(), false),
    "DATABASE": zod.String().Min(1),
    "APP_ENV":  zod.Default(zod.Enum("dev", "staging", "prod"), "dev"),
})

func loadEnv() (map[string]any, error) {
    raw := map[string]any{
        "PORT":     os.Getenv("PORT"),
        "DEBUG":    os.Getenv("DEBUG"),
        "DATABASE": os.Getenv("DATABASE_URL"),
        "APP_ENV":  os.Getenv("APP_ENV"),
    }
    // Empty optional-ish: drop blank DEBUG so Default applies
    if raw["DEBUG"] == "" {
        delete(raw, "DEBUG")
    }
    if raw["APP_ENV"] == "" {
        delete(raw, "APP_ENV")
    }
    return envSchema.Parse(raw)
}
```

## 3. Pagination query

```go
type PageQuery struct {
    Page  float64 `json:"page"`
    Limit float64 `json:"limit"`
    Q     string  `json:"q"`
}

var pageQuery = zod.ToStruct[PageQuery](zod.Object(zod.Shape{
    "page":  zod.Default(zod.Coerce.Number().Gte(1), float64(1)),
    "limit": zod.Default(zod.Coerce.Number().Gte(1).Lte(100), float64(20)),
    "q":     zod.Optional(zod.String().Max(200)),
}))

// Gin:
// q, ok := zgin.BindQuery(c, pageQuery)
```

## 4. Nested address

```go
address := zod.Object(zod.Shape{
    "line1":   zod.String().Min(1),
    "line2":   zod.Optional(zod.String()),
    "city":    zod.String().Min(1),
    "zip":     zod.String().Min(5).Max(10),
    "country": zod.String().Length(2),
})

profile := zod.Object(zod.Shape{
    "name":    zod.String().Min(2),
    "email":   zod.String().Email(),
    "address": address,
})

type Address struct {
    Line1   string  `json:"line1"`
    Line2   *string `json:"line2"`
    City    string  `json:"city"`
    Zip     string  `json:"zip"`
    Country string  `json:"country"`
}
type Profile struct {
    Name    string  `json:"name"`
    Email   string  `json:"email"`
    Address Address `json:"address"`
}

var parseProfile = zod.ToStruct[Profile](profile)
```

## 5. Recursive comments

```go
var Comment zod.AnySchemaLike
Comment = zod.Lazy(func() zod.AnySchemaLike {
    return zod.Object(zod.Shape{
        "id":      zod.String().UUID(),
        "author":  zod.String().Min(1),
        "body":    zod.String().Min(1).Max(5000),
        "replies": zod.Default(zod.Array(Comment), []any{}),
    })
})

thread := zod.Object(zod.Shape{
    "title":    zod.String().Min(1),
    "comments": zod.Array(Comment),
})
```

## 6. Discriminated union webhooks

```go
webhooks := zod.DiscriminatedUnion("type", []zod.AnySchemaLike{
    zod.Object(zod.Shape{
        "type": zod.Literal("user.created"),
        "data": zod.Object(zod.Shape{
            "userId": zod.String().UUID(),
            "email":  zod.String().Email(),
        }),
    }),
    zod.Object(zod.Shape{
        "type": zod.Literal("user.deleted"),
        "data": zod.Object(zod.Shape{
            "userId": zod.String().UUID(),
        }),
    }),
    zod.Object(zod.Shape{
        "type":   zod.Literal("ping"),
        "sentAt": zod.String().ISODateTime(),
    }),
})

func handleWebhook(raw any) error {
    evt, err := webhooks.Parse(raw)
    if err != nil {
        return err
    }
    m := evt.(map[string]any)
    switch m["type"] {
    case "user.created":
        // ...
    case "user.deleted":
        // ...
    case "ping":
        // ...
    }
    return nil
}
```

## 7. Gin middleware CRUD

```go
var userBody = zod.Object(zod.Shape{
    "name":  zod.String().Min(2).Max(100),
    "email": zod.String().Email(),
})

var userID = zod.Object(zod.Shape{
    "id": zod.String().UUID(),
})

func RegisterUserRoutes(r *gin.Engine) {
    g := r.Group("/users")

    g.POST("", zgin.Validate(userBody), func(c *gin.Context) {
        body, _ := zgin.Get(c)
        c.JSON(201, body)
    })

    g.GET("/:id", func(c *gin.Context) {
        p, ok := zgin.BindURI(c, zod.ToStruct[struct {
            ID string `json:"id"`
        }](userID))
        if !ok {
            return
        }
        c.JSON(200, gin.H{"id": p.ID})
    })

    g.PUT("/:id", func(c *gin.Context) {
        if _, ok := zgin.BindURI(c, zod.ToStruct[struct {
            ID string `json:"id"`
        }](userID)); !ok {
            return
        }
        body, ok := zgin.BindJSONAny(c, userBody)
        if !ok {
            return
        }
        c.JSON(200, body)
    })

    g.DELETE("/:id", func(c *gin.Context) {
        if _, ok := zgin.BindURI(c, zod.ToStruct[struct {
            ID string `json:"id"`
        }](userID)); !ok {
            return
        }
        c.Status(204)
    })
}
```

## 8. Parallel batch import

```go
var importRow = zod.Object(zod.Shape{
    "email": zod.String().Email(),
    "name":  zod.String().Min(1),
    "role":  zod.Default(zod.Enum("user", "admin"), "user"),
})

func ImportUsers(ctx context.Context, rows []any) ([]any, error) {
    out, err := zod.ParseParallelSlice(ctx, importRow, rows, zod.ParallelOpts{
        Workers:  runtime.NumCPU(),
        MinChunk: 64,
    })
    if err != nil {
        if zerr, ok := err.(*zod.ZodError); ok {
            // log zod.Prettify(zerr) or return Flatten for the client
            _ = zerr
        }
        return out, err
    }
    return out, nil
}
```

## 9. Partial update (PATCH)

```go
base := zod.Object(zod.Shape{
    "name":  zod.String().Min(2),
    "email": zod.String().Email(),
    "bio":   zod.String().Max(500),
})

// All fields optional for PATCH
patch := base.Partial()

// Or pick a subset
emailOnly := base.Pick("email")
```

## 10. Catch bad query flags

```go
flag := zod.Catch(zod.Coerce.Bool(), false)
// "?verbose=nope" → false instead of 400
```
