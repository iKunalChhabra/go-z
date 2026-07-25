# Map & Set

Go has no built-in JS `Map` / `Set`. go-z provides stand-ins that accept common Go containers and mirror issue taxonomy (`Origin: "map"` / `"set"`).

## Map

`z.Map(keySchema, valueSchema)` accepts:

- `map[any]any`
- `map[string]any`
- other `reflect.Map` kinds

Output is always `map[any]any`.

```go
schema:= z.Map(z.String, z.String)

out:= schema.MustParse(map[any]any{"first": "foo", "second": "bar"})
out = schema.MustParse(map[string]any{"a": "b"})

res:= schema.SafeParse([]any{})
// Expected: "map"
```

### Size checks

| Method | Rule | Origin |
|--------|------|--------|
| `Min(n)` | entries ≥ n | `"map"` |
| `Max(n)` | entries ≤ n | `"map"` |
| `Size(n)` | entries == n | `"map"` |
| `NonEmpty` | `Min(1)` | `"map"` |

```go
minTwo:= z.Map(z.String, z.String).Min(2)
minTwo.MustParse(map[any]any{"a": "b", "c": "d"})
res:= minTwo.SafeParse(map[any]any{"a": "b"})
// too_small, Origin: "map"

maxTwo:= z.Map(z.String, z.String).Max(2)
_ = maxTwo.SafeParse(map[any]any{"a": "b", "c": "d", "e": "f"})

justTwo:= z.Map(z.String, z.String).Size(2)
justTwo.MustParse(map[any]any{"a": "b", "c": "d"})

_ = z.Map(z.String, z.String).NonEmpty.SafeParse(map[any]any{})
```

### Key / value errors

Property-key-like keys (`string`, numbers) get **path-prefixed** issues. Other key types wrap failures in `invalid_key` / `invalid_element`.

```go
schema:= z.Map(z.String, z.String)

res:= schema.SafeParse(map[any]any{1: "foo"})
// int key → path-prefixed invalid_type on the key schema
// Path starts with 1

res = schema.SafeParse(map[any]any{"ok": 12})
// value type error, path ["ok"]
```

## Set

`z.Set(valueSchema)` accepts:

- `[]any` and other slices — **uniqueness enforced** (first occurrence wins; duplicates dropped)
- `map[any]struct{}` / `map[string]struct{}` — keys are the elements

Output is always `[]any`.

```go
schema:= z.Set(z.String)

out:= schema.MustParse([]any{"first", "second"})
// []any{"first", "second"}

// Uniqueness
out = schema.MustParse([]any{"a", "a", "b"})
// []any{"a", "b"}

// From a Go “set” map
out = schema.MustParse(map[string]struct{}{"a": {}, "b": {}})
```

:::info Uniqueness & comparability
Only comparable values (strings, numbers, bools, nil, …) participate in the seen-set. Non-comparable values are always kept (cannot be map keys in Go).
:::

### Size checks

Size is measured on the **deduplicated** output (`Origin: "set"`).

```go
minTwo:= z.Set(z.String).Min(2)
minTwo.MustParse([]any{"a", "b"})
res:= minTwo.SafeParse([]any{"just_one"})
// too_small, Origin: "set"

maxTwo:= z.Set(z.String).Max(2)
_ = maxTwo.SafeParse([]any{"one", "two", "three"})

justTwo:= z.Set(z.String).Size(2)
justTwo.MustParse([]any{"a", "b"})

_ = z.Set(z.String).NonEmpty.SafeParse([]any{})
```

### Element errors (no path)

Set element issues are **not** path-prefixed (matches `handleSetResult`):

```go
schema:= z.Set(z.String)
res:= schema.SafeParse([]any{"ok", 12})
// invalid_type, Path: [] (empty)
```

### Rejected inputs

```go
// map[string]any is not a set stand-in
_ = z.Set(z.String).SafeParse(map[string]any{"a": 1})

_ = z.Set(z.String).SafeParse("nope")
```

## Map vs Record vs Set

| | Map | Record | Set |
|--|-----|--------|-----|
| Keys | any (validated) | strings only | — |
| Input | Go maps | string-key maps | slices / struct{} maps |
| Output | `map[any]any` | `map[string]any` | `[]any` |
| Exhaustive enum keys | no | yes | — |
| Uniqueness | n/a | n/a | yes |

Prefer **Record** for JSON objects with dynamic string keys. Prefer **Map** when keys are non-strings or you already have `map[any]any`. Prefer **Set** for unique collections.

## API surface

```go
func Map(keySchema, valueSchema AnySchemaLike, params...any) *MapSchema
func (m *MapSchema) Min / Max / Size / NonEmpty(...)

func Set(valueSchema AnySchemaLike, params...any) *SetSchema
func (s *SetSchema) Min / Max / Size / NonEmpty(...)
```
