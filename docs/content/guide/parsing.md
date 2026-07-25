# Schemas & parsing

Everything in go-z is a schema value. You build it with fluent constructors, then call `Parse`, `MustParse`, `SafeParse`, or `ParseCtx`. This page covers the parse surface, the `Schema[T]` / `AnySchemaLike` split, and how fluent methods stay immutable.

## The parse surface

Every concrete schema embeds a typed base that implements:

```go
type Schema[T any] interface {
	Parse(data any) (T, error)
	MustParse(data any) T
	SafeParse(data any) SafeParseResult[T]
	Internals() *Internals
}
```

Plus `ParseCtx` on the concrete types for per-call options.

### Parse

Validates `data` and returns the typed output or a `*z.Error`.

```go
schema := z.String().Min(3)

s, err := schema.Parse("hey")
if err != nil {
	zerr := err.(*z.Error)
	fmt.Println(zerr.Issues[0].Message)
	return
}
fmt.Println(s) // "hey"
```

Object schemas return `map[string]any`:

```go
obj := z.Object(z.Shape{
	"id": z.String().UUID(),
})

m, err := obj.Parse(map[string]any{
	"id": "550e8400-e29b-41d4-a716-446655440000",
})
// m["id"] is the validated string
```

### MustParse

Same as `Parse`, but panics with `*z.Error` on failure. Use for tests, `init`, or trusted fixtures — not request handlers.

```go
email := z.String().Email().MustParse("ada@example.com")

// panics:
// z.String().Email().MustParse("nope")
```

### SafeParse

Never returns an `error` value. Mirrors `safeParse` result:

```go
type SafeParseResult[T any] struct {
	Success bool
	Data    T
	Error   *Error
}
```

```go
res := z.Int().Gte(0).SafeParse(-1)
if !res.Success {
	fmt.Println(res.Error.Issues[0].Code) // too_small
	return
}
fmt.Println(res.Data)
```

:::tip Prefer SafeParse in branching handlers
When both success and failure are expected control flow (HTTP 200 vs 400), `SafeParse` keeps the happy path free of type assertions on `error`.
:::

### ParseCtx

Per-parse options — `ParseContext`:

```go
type ParseCtx struct {
	Error       ErrorMap // per-parse error map
	ReportInput bool     // keep offending input on finalized issues
}
```

```go
ctx := &z.ParseCtx{
	ReportInput: true,
	Error: func(iss *z.Issue) string {
		if iss.Code == z.IssueTooSmall {
			return "needs more characters"
		}
		return "" // defer to next map in the chain
	},
}

_, err := z.String().Min(5).ParseCtx("hi", ctx)
zerr := err.(*z.Error)
fmt.Println(zerr.Issues[0].Message) // needs more characters
fmt.Println(zerr.Issues[0].Input)   // "hi"  (because ReportInput)
```

By default, `FinalizeIssue` clears `Input` unless `ReportInput` is true — matching privacy-minded default.

## Schema[T] vs AnySchemaLike

### Schema[T]

The typed public surface. `String()` is roughly `Schema[string]`; `Object(...)` is `Schema[map[string]any]`. Wrappers keep the inner type: `String().Optional()` is `Schema[*string]` (nil when absent) and `String().Default("x")` is `Schema[string]`. Only the type-erased constructors — `z.Optional(anySchemaLike)` — widen to `any`.

```go
var asString z.Schema[string] = z.String().Min(1)
out, err := asString.Parse("x")
```

### AnySchemaLike

Type-erased view used by containers that hold heterogeneous children:

```go
type AnySchemaLike interface {
	Internals() *Internals
}
```

```go
shape := z.Shape{
	"name": z.String(),          // *StringSchema
	"age":  z.Optional(z.Int()), // *OptionalSchema[any]
	"meta": z.Record(z.String(), z.Unknown()),
}
user := z.Object(shape)
```

You can always go from typed schema → `AnySchemaLike` because every schema exposes `Internals`. Containers only need the untyped core to run child parses and prefix paths.

## What happens inside Parse

Simplified:

```text
Parse(data)
  └─ AcquirePayload(data)          // pooled Payload
  └─ internals.Run(payload, ctx)
        ├─ type parse (normalize / invalid_type)
        └─ runChecks (Min, Email, …)   // skipped if zero checks
  └─ if issues → FinalizeIssue each → *Error
  └─ else assert payload.Value.(T) → return
  └─ ReleasePayload
```

Issues accumulate. Containers call `RunChild`, which merges child issues and prepends path segments (`"email"`, `0`, …).

## Immutable fluent clones

Fluent methods never mutate the receiver. They clone the definition and append checks:

```go
base := z.String()
email := base.Email()
longEmail := email.Min(5)

// base is still a plain string schema (no email check)
_, err1 := base.Parse("not-an-email") // ok — no format check

// email requires format only
_, err2 := email.Parse("a@b.c") // may pass format

// longEmail requires format AND min length
_, err3 := longEmail.Parse("a@b.c") // too_small
```

Under the hood, helpers like `Min` do:

```go
func (s *StringSchema) Min(n int, params ...any) *StringSchema {
	return newString(s.def.withChecks(MinLength(n, params...)))
}
```

`withChecks` copies the `Def` and appends to a new `Checks` slice. That is why package-level schemas are safe to share and extend:

```go
var Name = z.String().Min(1).Max(100)

// Local specialization without mutating Name
adminName := Name.Min(3) // additional min (stricter bag metadata + extra check)
```

:::info Clone cost is construction-time
You pay for cloning when you *build* schemas (usually once at startup). Hot-path `Parse` does not clone the schema — it only acquires a pooled `Payload`.
:::

## Params on fluent methods

Most constraint methods accept variadic params — a string message, an `ErrorMap`, or `Params{}`:

```go
z.String().Min(5, "too short")

z.String().Min(5, z.Params{
	Error: z.MessageFromString("too short"),
	Abort: true, // stop later checks if this fails
})

z.String().Email(func(iss *z.Issue) string {
	return "email looks wrong"
})
```

See [Error maps & locales](#/guide/error-maps) and [Checks](#/guide/checks).

## Working with JSON

`encoding/json` unmarshals objects into `map[string]any` and arrays into `[]any` — exactly what object/array schemas expect:

```go
var input any
_ = json.Unmarshal(body, &input)

out, err := User.Parse(input)
```

JSON numbers arrive as `float64`, and each numeric schema converts them to its own output type: `z.Number()` keeps `float64`, `z.Int()` gives an `int`, `z.Int64()` an `int64`. The conversion must be exact, so `3.7` fails `z.Int()` instead of being truncated.

## Package-level type changers

Go methods cannot introduce new type parameters, so transforms live as functions:

```go
// Conceptual — Transform / Pipe / ToStruct are package-level
typed := z.ToStruct[User](userSchema)
u, err := typed.Parse(input)
```

## Common patterns

### Parse once at the boundary

```go
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var input any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}
	data, err := User.Parse(input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   err.(*z.Error),
		})
		return
	}
	_ = data
}
```

### Specialize shared schemas

```go
var Password = z.String().Min(8).Max(128)

signup := z.Object(z.Shape{
	"password": Password,
})

// Stricter admin password without mutating Password
adminSignup := z.Object(z.Shape{
	"password": Password.Min(16),
})
```

### MustParse for fixtures

```go
var fixture = User.MustParse(map[string]any{
	"name":  "Ada",
	"email": "ada@example.com",
})
```

## Related

- [Issues & Error](#/guide/errors)
- [Missing vs nil](#/guide/missing-nil)
- [Immutability & concurrency](#/guide/concurrency)
- [Checks & refinements](#/guide/checks)
