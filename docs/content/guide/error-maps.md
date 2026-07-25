# Error maps & locales

Every issue eventually gets a `Message`. go-zod resolves that string through Zod’s precedence chain: explicit message → producing check/schema map → per-parse map → global custom map → locale → `"Invalid input"`.

## ErrorMap type

```go
// Return "" to defer to the next map in the chain.
type ErrorMap func(iss *Issue) string
```

### MessageFromString

Shorthand for a fixed message — Zod’s `z.string().min(5, "too short")`:

```go
msg := z.MessageFromString("too short")

schema := z.String().Min(5, msg)
// or simply:
schema = z.String().Min(5, "too short")
```

Empty string → `nil` map (no override).

## Params.Error on schemas and checks

Fluent methods accept a string, an `ErrorMap`, or `Params`:

```go
z.String().Min(8, z.Params{
	Error: z.MessageFromString("password too short"),
})

z.String().Email(z.Params{
	Error: func(iss *z.Issue) string {
		if iss.Format == "email" {
			return "That doesn't look like an email"
		}
		return ""
	},
})
```

The map is attached to issues **produced by that check/schema**. Returning `""` lets the chain continue.

Schema-level maps (on constructors) cover type errors from that schema’s own parse:

```go
z.String(z.Params{
	Error: z.MessageFromString("expected a string value"),
})
```

## Per-parse: ParseCtx.Error

Override messages for a single call without changing the schema:

```go
ctx := &z.ParseCtx{
	Error: func(iss *z.Issue) string {
		switch iss.Code {
		case z.IssueTooSmall:
			return "Please write a bit more"
		case z.IssueInvalidFormat:
			return "Format check failed"
		default:
			return ""
		}
	},
}

_, err := z.String().Min(5).Email().ParseCtx("x", ctx)
fmt.Println(err.(*z.ZodError).Issues[0].Message)
// Please write a bit more
```

## Global config: CustomError & LocaleError

```go
type Config struct {
	CustomError ErrorMap // third link — before locale
	LocaleError ErrorMap // lowest priority; default EnLocale
}
```

`Configure` atomically replaces the global config and returns the previous one. Safe for concurrent use.

```go
prev := z.Configure(z.Config{
	CustomError: func(iss *z.Issue) string {
		if iss.Code == z.IssueUnrecognizedKeys {
			return "Please remove unknown fields"
		}
		return ""
	},
	LocaleError: z.Locale("es"),
})

// ... later, restore
z.Configure(prev)
```

:::warn Global = process-wide
`Configure` affects every parse in the process. Prefer `Params.Error` or `ParseCtx.Error` for request-scoped customization. Use global config for app-wide locale and a few universal overrides.
:::

## Resolution order (detailed)

For each raw issue, `FinalizeIssue` does:

1. If `iss.Message` is already set → keep it  
2. Else call the producing schema/check `ErrorMap`  
3. Else call `ParseCtx.Error` (if any)  
4. Else call `Config.CustomError` (if any)  
5. Else call `Config.LocaleError`, or `EnLocale` if unset  
6. Else `"Invalid input"`  

```go
// Demonstration of deferral
z.Configure(z.Config{
	CustomError: func(iss *z.Issue) string {
		// Only handle custom codes; defer everything else
		if iss.Code == z.IssueCustom {
			return "Business rule failed"
		}
		return ""
	},
	LocaleError: z.EsLocale,
})
```

## Built-in locales

| Locale | Function | `Locale(name)` keys |
|---|---|---|
| English (default) | `EnLocale` | `en`, `eng`, `english` |
| Spanish | `EsLocale` | `es`, `spa`, `spanish`, `español`, … |
| French | `FrLocale` | `fr`, `fra`, `french`, … |
| German | `DeLocale` | `de`, `deu`, `german`, `deutsch` |
| Japanese | `JaLocale` | `ja`, `jpn`, `japanese` |
| Portuguese | `PtLocale` | `pt`, `por`, `portuguese`, … |
| Chinese (Simplified) | `ZhLocale` | `zh`, `zh-cn`, `chinese`, `cn`, … |

Unknown names fall back to `EnLocale`.

### Locale(name)

```go
z.Configure(z.Config{
	LocaleError: z.Locale("ja"),
})

_, err := z.String().Min(5).Parse("hi")
fmt.Println(err.(*z.ZodError).Issues[0].Message)
// Japanese too_small message
```

Or pick a function directly:

```go
z.Configure(z.Config{LocaleError: z.FrLocale})
```

### Side-by-side messages

Same issue, different locales:

```go
iss := &z.Issue{
	Code:      z.IssueTooSmall,
	Origin:    "string",
	Minimum:   5,
	Inclusive: true,
	Input:     "hi",
}

fmt.Println(z.EnLocale(iss))
// Too small: expected string to have >=5 characters

fmt.Println(z.EsLocale(iss))
// Demasiado pequeño: se esperaba que la cadena tuviera >=5 caracteres
// (wording follows the Spanish locale templates)

fmt.Println(z.Locale("de")(iss))
// German template
```

:::tip Accept-Language → Locale
In HTTP handlers, map the first supported tag to `Locale` and pass it via `ParseCtx.Error` wrapping, or set locale once per worker if the process is single-tenant per language.
:::

## Practical patterns

### Field-level product copy

```go
password := z.String().
	Min(8, "Use at least 8 characters").
	Max(128, "Password is too long")
```

### Shared brand voice via CustomError

```go
z.Configure(z.Config{
	LocaleError: z.EnLocale,
	CustomError: func(iss *z.Issue) string {
		switch iss.Code {
		case z.IssueInvalidType:
			return "We couldn't read this field — check the type"
		default:
			return "" // keep locale messages for too_small, email, etc.
		}
	},
})
```

### Per-request locale without global mutation

```go
func parseWithLocale(schema z.AnySchemaLike, data any, lang string) (any, error)
{
	locale := z.Locale(lang)
	// Wrap as ParseCtx.Error — runs before global locale
	ctx := &z.ParseCtx{
		Error: func(iss *z.Issue) string {
			return locale(iss)
		},
	}
	// Concrete schemas expose ParseCtx; for AnySchemaLike run via a typed helper
	// or call the concrete schema you already have:
	return schema.(interface {
		ParseCtx(any, *z.ParseCtx) (any, error)
	}).ParseCtx(data, ctx)
}
```

Simpler when you already hold a concrete schema:

```go
out, err := userSchema.ParseCtx(input, &z.ParseCtx{
	Error: z.Locale("pt"),
})
```

Because a per-parse map that always returns a string **short-circuits** the chain, this replaces locale messages for that call. To only override some codes, return `""` for the rest:

```go
lang := z.Locale("pt")
ctx := &z.ParseCtx{
	Error: func(iss *z.Issue) string {
		if iss.Code == z.IssueCustom {
			return "Regra de negócio falhou"
		}
		return lang(iss) // or "" to fall through to global locale
	},
}
```

### Restore config in tests

```go
func TestSpanish(t *testing.T) {
	prev := z.Configure(z.Config{LocaleError: z.EsLocale})
	t.Cleanup(func() { z.Configure(prev) })

	_, err := z.String().Min(5).Parse("x")
	if err == nil {
		t.Fatal("expected error")
	}
}
```

## Related

- [Issues & ZodError](#/guide/errors) — formatters once you have messages  
- [Checks & refinements](#/guide/checks) — attaching maps on check construction  
- [Schemas & parsing](#/guide/parsing) — `ParseCtx` options  
