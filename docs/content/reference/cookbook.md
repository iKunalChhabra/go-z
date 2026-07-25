# Cookbook

Copy-paste recipes for common go-zod patterns.

## 1. Signup form

```go
var Signup = z.SuperRefine(z.Object(z.Shape{
    "email":           z.String().Email(),
    "password":        z.String().Min(8).Max(128),
    "confirmPassword": z.String(),
    "acceptTerms":     z.Literal(true),
}), func(v any, ctx *z.RefinementCtx) {
    m := v.(map[string]any)
    if m["password"] != m["confirmPassword"] {
        ctx.AddIssue(z.Issue{
            Code:    z.IssueCustom,
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

envSchema := z.Object(z.Shape{
    "PORT":     z.Coerce.Number().Gte(1).Lte(65535),
    "DEBUG":    z.Default(z.Coerce.Bool(), false),
    "DATABASE": z.String().Min(1),
    "APP_ENV":  z.Default(z.Enum("dev", "staging", "prod"), "dev"),
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

var pageQuery = z.ToStruct[PageQuery](z.Object(z.Shape{
    "page":  z.Default(z.Coerce.Number().Gte(1), float64(1)),
    "limit": z.Default(z.Coerce.Number().Gte(1).Lte(100), float64(20)),
    "q":     z.Optional(z.String().Max(200)),
}))

// Gin:
// q, ok := zgin.BindQuery(c, pageQuery)
```

## 4. Nested address

```go
address := z.Object(z.Shape{
    "line1":   z.String().Min(1),
    "line2":   z.Optional(z.String()),
    "city":    z.String().Min(1),
    "zip":     z.String().Min(5).Max(10),
    "country": z.String().Length(2),
})

profile := z.Object(z.Shape{
    "name":    z.String().Min(2),
    "email":   z.String().Email(),
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

var parseProfile = z.ToStruct[Profile](profile)
```

## 5. Recursive comments

```go
var Comment z.AnySchemaLike
Comment = z.Lazy(func() z.AnySchemaLike {
    return z.Object(z.Shape{
        "id":      z.String().UUID(),
        "author":  z.String().Min(1),
        "body":    z.String().Min(1).Max(5000),
        "replies": z.Default(z.Array(Comment), []any{}),
    })
})

thread := z.Object(z.Shape{
    "title":    z.String().Min(1),
    "comments": z.Array(Comment),
})
```

## 6. Discriminated union webhooks

```go
webhooks := z.DiscriminatedUnion("type", []z.AnySchemaLike{
    z.Object(z.Shape{
        "type": z.Literal("user.created"),
        "data": z.Object(z.Shape{
            "userId": z.String().UUID(),
            "email":  z.String().Email(),
        }),
    }),
    z.Object(z.Shape{
        "type": z.Literal("user.deleted"),
        "data": z.Object(z.Shape{
            "userId": z.String().UUID(),
        }),
    }),
    z.Object(z.Shape{
        "type":   z.Literal("ping"),
        "sentAt": z.String().ISODateTime(),
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
var userBody = z.Object(z.Shape{
    "name":  z.String().Min(2).Max(100),
    "email": z.String().Email(),
})

var userID = z.Object(z.Shape{
    "id": z.String().UUID(),
})

func RegisterUserRoutes(r *gin.Engine) {
    g := r.Group("/users")

    g.POST("", zgin.Validate(userBody), func(c *gin.Context) {
        body, _ := zgin.Get(c)
        c.JSON(201, body)
    })

    g.GET("/:id", func(c *gin.Context) {
        p, ok := zgin.BindURI(c, z.ToStruct[struct {
            ID string `json:"id"`
        }](userID))
        if !ok {
            return
        }
        c.JSON(200, gin.H{"id": p.ID})
    })

    g.PUT("/:id", func(c *gin.Context) {
        if _, ok := zgin.BindURI(c, z.ToStruct[struct {
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
        if _, ok := zgin.BindURI(c, z.ToStruct[struct {
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
var importRow = z.Object(z.Shape{
    "email": z.String().Email(),
    "name":  z.String().Min(1),
    "role":  z.Default(z.Enum("user", "admin"), "user"),
})

func ImportUsers(ctx context.Context, rows []any) ([]any, error) {
    out, err := z.ParseParallelSlice(ctx, importRow, rows, z.ParallelOpts{
        Workers:  runtime.NumCPU(),
        MinChunk: 64,
    })
    if err != nil {
        if zerr, ok := err.(*z.ZodError); ok {
            // log z.Prettify(zerr) or return Flatten for the client
            _ = zerr
        }
        return out, err
    }
    return out, nil
}
```

## 9. Partial update (PATCH)

```go
base := z.Object(z.Shape{
    "name":  z.String().Min(2),
    "email": z.String().Email(),
    "bio":   z.String().Max(500),
})

// All fields optional for PATCH
patch := base.Partial()

// Or pick a subset
emailOnly := base.Pick("email")
```

## 10. Catch bad query flags

```go
flag := z.Catch(z.Coerce.Bool(), false)
// "?verbose=nope" → false instead of 400
```
