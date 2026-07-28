# Checks & refinements

Checks are composable validation units. After a schema’s bare type parse succeeds (or at least runs), attached checks inspect — and sometimes mutate — the payload. Fluent helpers like `Min` and `Email` are thin wrappers that attach check objects.

## The Check object

```go
type Check struct {
	Name     string              // e.g. "min_length", "greater_than"
	Fn       CheckFn             // func(p *Payload)
	Error    ErrorMap            // check-level messages
	Abort    bool                // stop later checks on failure
	When     func(p *Payload) bool
	OnAttach []func(in *Internals)
}

type CheckFn func(p *Payload)
```

Checks append issues with `payload.AddIssue(ch.Issue(...))`. `Check.Issue` wires the check’s error map and **continue** semantics:

- `Abort: false` (default) → issue gets `continue: true` → later checks may still run
- `Abort: true` → issue aborts the remaining chain (`{ abort: true }` semantics)

## How Min / Email attach checks

```go
func (s *StringSchema) Min(n int, params ...any) *StringSchema {
	return newString(s.def.withChecks(MinLength(n, params...)))
}

func (s *StringSchema) Email(params ...any) *StringSchema {
	return newString(s.def.withChecks(FormatEmail(params...)))
}
```

`MinLength` builds a `*Check`:

```go
func MinLength(minimum int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "min_length",
		Error: p.Error,
		Abort: p.Abort,
		When:  hasLength, // only run for strings / []any
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				// stash minimum into schema bag for metadata consumers
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				curr, _ := in.Bag["minimum"].(int)
				if minimum > curr {
					in.Bag["minimum"] = minimum
				}
			},
		},
	}
	ch.Fn = func(payload *z.Payload) {
		n, ok := lengthOf(payload.Value)
		if !ok || n >= minimum {
			return
		}
		payload.AddIssue(ch.Issue(z.Issue{
			Code:      z.IssueTooSmall,
			Origin:    "string",
			Minimum:   minimum,
			Inclusive: true,
			Input:     payload.Value,
		}))
	}
	return ch
}
```

:::tip Bag metadata
`OnAttach` runs when the check is attached to a schema (construction time). It can record hints like `minimum` / `maximum` / format on `Internals.Bag` — the same idea as `inst._zod.bag`.
:::

## Attaching raw checks

Every schema type that supports composition exposes `.Check(...)`:

```go
onlyHello := z.String().Check(&z.Check{
	Name: "only_hello",
	Fn: func(p *z.Payload) {
		if p.Value != "hello" {
			p.AddIssue(z.Issue{
				Code:    z.IssueCustom,
				Message: "say hello",
				Input:   p.Value,
			})
		}
	},
})

_, err := onlyHello.Parse("hi")
// custom: say hello
```

Prefer high-level helpers (`Min`, `Email`, `Refine`, …) when they exist; use raw `Check` for one-off rules or library extensions.

## Standalone check constructors

Every fluent method (`Min`, `Email`, `Gt`, …) is a thin wrapper over an exported constructor that returns a `*Check`. Reach for these when building checks dynamically, attaching the same check to several schemas, or writing your own fluent helpers:

```go
s := z.String().Check(z.MinLength(3), z.Trim())
n := z.Number().Check(z.Gte(0), z.MultipleOf(0.5))
```

### Number checks

| Constructor | Fluent equivalent | Rule |
|---|---|---|
| `Gt(value float64, ...)` | `.Gt` | `>` |
| `Gte(value float64, ...)` | `.Gte` / `.Min` | `≥` |
| `Lt(value float64, ...)` | `.Lt` | `<` |
| `Lte(value float64, ...)` | `.Lte` / `.Max` | `≤` |
| `GreaterThan(value any, inclusive bool, ...)` | — | generic form of the above |
| `LessThan(value any, inclusive bool, ...)` | — | generic form of the above |
| `MultipleOf(divisor float64, ...)` | `.MultipleOf` | float divisibility |
| `MultipleOfInt64(divisor int64, ...)` | `.MultipleOf` | exact int64 divisibility |
| `MultipleOfUint64(divisor uint64, ...)` | `.MultipleOf` | exact uint64 divisibility |
| `MultipleOfBigInt(divisor *big.Int, ...)` | `.MultipleOf` | big.Int divisibility |
| `NumberFormat(format string, ...)` | `.Int`, `.Safe`, … | e.g. `"safeint"`, `"int32"`, `"float64"` |

### String checks

| Constructor | Fluent equivalent | Rule |
|---|---|---|
| `MinLength(minimum int, ...)` | `.Min` | length ≥ |
| `MaxLength(maximum int, ...)` | `.Max` | length ≤ |
| `LengthEquals(length int, ...)` | `.Length` | exact length |
| `Regex(pattern *regexp.Regexp, ...)` | `.Regex` | pattern match |
| `Includes(includes string, ...)` | `.Includes` | substring |
| `StartsWith(prefix string, ...)` | `.StartsWith` | prefix |
| `EndsWith(suffix string, ...)` | `.EndsWith` | suffix |
| `LowerCase(...)` | `.Lowercase` | must be lowercase |
| `UpperCase(...)` | `.Uppercase` | must be uppercase |
| `Overwrite(tx func(string) string)` | — | generic transform |
| `Trim()` | `.Trim` | transform: trim space |
| `ToLowerCase()` | `.ToLowerCase` | transform: lowercase |
| `ToUpperCase()` | `.ToUpperCase` | transform: uppercase |
| `NormalizeNFC()` | `.Normalize` | transform: Unicode NFC |

### Format checks

Each [string format](#/api/string-formats) has a constructor too: `FormatEmail`, `FormatURL`, `FormatHttpURL`, `FormatHostname`, `FormatHash(alg)`, `FormatUUID` / `FormatUUIDv4` / `FormatUUIDv6` / `FormatUUIDv7`, `FormatGUID`, `FormatNanoID`, `FormatCUID`, `FormatCUID2`, `FormatULID`, `FormatKSUID`, `FormatXID`, `FormatBase64`, `FormatBase64URL`, `FormatHex`, `FormatJWT`, `FormatE164`, `FormatEmoji`, `FormatIPv4`, `FormatIPv6`, `FormatCIDRv4`, `FormatCIDRv6`, `FormatMAC`, `FormatISODate`, `FormatISOTime`, `FormatISODateTime`, `FormatISODuration`. They accept the same option structs as their fluent counterparts (`URLOpts`, `JWTOpts`, `MACOpts`, `ISOTimeOpts`, `ISODateTimeOpts`).

All constructors take trailing `params ...any` for custom messages / abort — see [Error maps](#/guide/error-maps).

## Abort semantics

Without abort, multiple failing checks can all contribute issues (subject to the continue flags):

```go
schema := z.String().Min(10).Email()

res := schema.SafeParse("x")
// typically too_small; email may or may not also run depending on abort state
// from prior issues (type errors abort; non-aborting check failures continue)
```

Force a hard stop after a specific check:

```go
schema := z.String().
	Min(5, z.Params{Abort: true, Error: z.MessageFromString("too short")}).
	Email() // skipped if Min fails with Abort

_, err := schema.Parse("x")
zerr := err.(*z.Error)
fmt.Println(len(zerr.Issues)) // 1 — email not evaluated
fmt.Println(zerr.Issues[0].Message) // too short
```

## When gates

`When` decides whether the check runs. Length checks use a gate so they don’t fire on wrong types (the type parse already reported `invalid_type`):

```go
ch := &z.Check{
	Name: "even_length",
	When: func(p *z.Payload) bool {
		_, ok := p.Value.(string)
		return ok
	},
	Fn: func(p *z.Payload) {
		s := p.Value.(string)
		if len(s)%2 != 0 {
			p.AddIssue(z.Issue{Code: z.IssueCustom, Message: "length must be even"})
		}
	},
}

schema := z.String().Check(ch)
```

### Continue vs When

The run loop tracks abort state:

- Ordinary checks are **skipped** once the payload is aborted (an issue without `continue: true`).
- Checks with **`When` set** still run unless the payload was **explicitly** aborted (`continue: false` / `Abort` on a prior issue). They skip themselves when `When` returns false.

That ports subtle `when` behavior: gated checks can still refine after soft failures, but a hard abort wins.

```go
// Pseudocode of runChecks
isAborted := payload.aborted(0)
for _, ch := range checks {
	if ch.When != nil {
		if payload.explicitlyAborted(0) {
			continue
		}
		if !ch.When(payload) {
			continue
		}
	} else if isAborted {
		continue
	}
	ch.Fn(payload)
	// update isAborted if new aborting issues appeared
}
```

## OnAttach

Hooks for schema metadata at attach time:

```go
ch := &z.Check{
	Name: "brand_min",
	OnAttach: []func(in *z.Internals){
		func(in *z.Internals) {
			if in.Bag == nil {
				in.Bag = map[string]any{}
			}
			in.Bag["brand"] = "acme"
		},
	},
	Fn: func(p *z.Payload) { /* ... */ },
}

s := z.String().Check(ch)
fmt.Println(s.Internals().Bag["brand"]) // acme
```

Multiple checks may write the same bag keys; helpers like `MinLength` keep the **strictest** minimum.

## Zero-check fast path

Schemas with no checks set `Run == Parse`. Adding the first check composes `parse → runChecks`. That’s why a bare `z.String()` stays cheap, and why you should build schemas once rather than re-fluent them inside hot loops.

```go
// Construction (once)
schema := z.String().Min(1).Email()

// Hot path — no cloning, no OnAttach
for _, item := range requests {
	_, _ = schema.Parse(item)
}
```

## Overwrite checks

A check may mutate `p.Value` (“overwrite” checks — trim, normalize, etc.). Later checks see the new value:

```go
trim := &z.Check{
	Name: "trim",
	Fn: func(p *z.Payload) {
		if s, ok := p.Value.(string); ok {
			p.Value = strings.TrimSpace(s)
		}
	},
}

schema := z.String().Check(trim).Min(3)
out, err := schema.Parse("  hi  ")
// after trim, length is 2 → too_small
_ = out
_ = err
```

(Prefer dedicated helpers when the package provides them — e.g. string trim/format utilities — but the mechanism is always a `Check`.)

## Refine-style custom rules

For value-level predicates, use the package refine helpers (see API pages) or a custom check with `IssueCustom`:

```go
func MultipleOfThree() *z.Check {
	ch := &z.Check{Name: "multiple_of_three"}
	ch.Fn = func(p *z.Payload) {
		n, ok := p.Value.(float64)
		if !ok {
			return
		}
		if int(n)%3 != 0 {
			p.AddIssue(ch.Issue(z.Issue{
				Code:   z.IssueCustom,
				Params: map[string]any{"rule": "multiple_of_three"},
				Input:  p.Value,
			}))
		}
	}
	return ch
}

var schema = z.Number().Check(MultipleOfThree())
```

Pair with an error map for a friendly message:

```go
ch := MultipleOfThree()
ch.Error = z.MessageFromString("must be a multiple of 3")
```

## Putting it together

```go
package main

import (
	"fmt"

	"github.com/iKunalChhabra/go-z/z"
)

func main() {
	schema := z.String().
		Min(3, z.Params{Abort: true}).
		Email()

	for _, in := range []any{"x", "not-an-email", "ada@example.com"} {
		res := schema.SafeParse(in)
		if res.Success {
			fmt.Println("ok", res.Data)
			continue
		}
		for _, iss := range res.Error.Issues {
			fmt.Printf("%v → %s (%s)\n", in, iss.Code, iss.Message)
		}
	}
}
```

## Related

- [Schemas & parsing](#/guide/parsing) — when checks run in the pipeline
- [Error maps & locales](#/guide/error-maps) — `Params.Error` on checks
- [Refine & Custom](#/api/refine) — higher-level refine API
