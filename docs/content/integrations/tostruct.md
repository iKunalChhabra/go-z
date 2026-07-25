# Struct binding (`ToStruct`)

Decode validated `map[string]any` into typed Go structs with a **cached reflection plan**.

## ToStruct

```go
type User struct {
    Name  string  `json:"name"`
    Email string  `json:"email"`
    Age   float64 `json:"age"`
}

schema:= z.ToStruct[User](z.Object(z.Shape{
    "name":  z.String.Min(2),
    "email": z.String.Email,
    "age":   z.Optional(z.Int.Gte(0)),
}))

user, err:= schema.Parse(map[string]any{
    "name":  "Ada",
    "email": "ada@example.com",
    "age":   36.0,
})
// user is User
```

Pipeline:

1. Run the inner schema (must produce `map[string]any` on success).
2. Decode into `T` with the cached plan.
3. Return `Schema[T]`.

:::warn T must be a non-pointer struct
`ToStruct[*User]` and non-struct `T` panic at construction. Use value structs; pointer fields inside the struct are fine.
:::

## json tags

Field names come from `` `json:"…"` `` tags (same rules as `encoding/json`):

| Tag | Behavior |
|---|---|
| `` `json:"name"` `` | Map key `name` |
| `` `json:"name,omitempty"` `` | Key `name` (omitempty ignored on decode) |
| `` `json:"-"` `` | Skipped |
| (no tag) | Field name as key |
| unexported fields | Skipped |

Nested structs, pointers, and slices of structs are supported. `time.Time` accepts `time.Time` or RFC3339 / RFC3339Nano strings. JSON numbers (`float64`) coerce into int/uint/float fields.

```go
type Address struct {
    City string `json:"city"`
    Zip  string `json:"zip"`
}
type Person struct {
    Name    string   `json:"name"`
    Address Address  `json:"address"`
    Tags    []string `json:"tags"`
    Born    *time.Time `json:"born"`
}
```

## Cached plans

Plans are built once and stored in process-wide maps:

| Cache | Key | Used by |
|---|---|---|
| `toStructPlans` | `(schema Internals pointer, reflect.Type)` | `ToStruct` |
| `decodePlans` | `reflect.Type` | `DecodeStruct` + shared with ToStruct |

Hot-path decode walks the plan only — **no per-parse type walks** of the full struct definition. Reuse the same `ToStruct` schema across requests (immutable, concurrency-safe).

:::tip Build schemas at init
```go
var createUser = z.ToStruct[User](userObject) // once
// then BindJSON / Parse in handlers
```
:::

## DecodeStruct

Decode **without** schema validation:

```go
user, err:= z.DecodeStruct[User](map[string]any{
    "name":  "Ada",
    "email": "ada@example.com",
})
```

Uses the type-keyed plan cache. Useful when data is already trusted or validated elsewhere.

## Error behavior

| Failure | Issue |
|---|---|
| Inner schema fails | Inner issues (unchanged) |
| Output not an object / not already `T` | `invalid_type` expected `"object"` |
| Field assign failure | `custom` with decode error message |

If `p.Value` is already type `T` after the inner run, ToStruct short-circuits and keeps it.

## Signatures

```go
func ToStruct[T any](schema AnySchemaLike) Schema[T]
func DecodeStruct[T any](data map[string]any) (T, error)
```

## With Gin

```go
var createUser = z.ToStruct[User](userObject)

r.POST("/users", func(c *gin.Context) {
    u, ok:= zgin.BindJSON(c, createUser)
    if !ok {
        return
    }
    c.JSON(200, u)
})
```
