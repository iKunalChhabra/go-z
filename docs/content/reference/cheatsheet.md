# API cheat sheet

Dense reference for `github.com/iKunalChhabra/go-z`. Import: `import "github.com/iKunalChhabra/go-z/z"`.

## Parse surface

| Call | Returns |
|---|---|
| `schema.Parse(data)` | `(T, error)` — error is `*z.Error` |
| `schema.MustParse(data)` | `T` or panic |
| `schema.SafeParse(data)` | `SafeParseResult[T]{Success, Data, Error}` |
| `schema.ParseCtx(data, ctx)` | `(T, error)` with per-parse `ParseCtx` |
| `schema.Internals` | `*Internals` untyped core |

## Primitives

| Constructor | Output `T` | Notes |
|---|---|---|
| `String` | `string` | |
| `Number` | `float64` | rejects NaN/Inf |
| `Int` | `float64` | + safeint check |
| `Int32` / `Uint32` / `Float32` / `Float64` | `float64` | format checks |
| `Bool` | `bool` | |
| `Time` | `time.Time` | |
| `BigInt` | `*big.Int` | |
| `Literal(values...)` | `any` | |
| `Enum(values...)` | `string` | string enums |
| `Any` / `Unknown` | `any` | |
| `Never` | — | always fails |
| `Nil` | `any` | JSON null only |

### String checks (fluent)

`Min` `Max` `Length` `NonEmpty` `Regex` `Includes` `StartsWith` `EndsWith` `Uppercase` `Lowercase` `Trim` `ToLowerCase` `ToUpperCase` `Normalize`

Formats: `Email` `URL` `UUID` `UUIDv4` `UUIDv6` `UUIDv7` `GUID` `NanoID` `CUID` `CUID2` `ULID` `KSUID` `XID` `Base64` `Base64URL` `Hex` `JWT` `E164` `Emoji` `IPv4` `IPv6` `CIDRv4` `CIDRv6` `MAC` `ISODate` `ISOTime` `ISODateTime` `ISODuration`

### Number checks (fluent)

`Gt` `Gte`/`Min` `Lt` `Lte`/`Max` `Positive` `Negative` `NonPositive` `NonNegative` `MultipleOf` `Step` `Int`/`Safe` `Finite`

## Coercion

```go
z.Coerce.String
z.Coerce.Number
z.Coerce.Bool
z.Coerce.Time
```

## Collections

| API | Output |
|---|---|
| `Object(Shape{...})` | `map[string]any` |
| `Array(elem)` | `[]any` |
| `Tuple([]AnySchemaLike{...})` | `[]any` |
| `Record(key, val)` | `map[string]any` |
| `Map(key, val)` | `map[any]any` |
| `Set(elem)` | `any` (set-like) |

### Object methods

`Strict` `Loose`/`Passthrough` `Strip` `Catchall` `Pick` `Omit` `Extend` `Merge` `Partial` `Required` `Keyof` `Shape` `Check`

### Array methods

`Min` `Max` `Length` `NonEmpty` `Check` (plus element schema)

## Composition

| API | Role |
|---|---|
| `Optional(s)` | Missing ok |
| `Nullable(s)` | nil ok |
| `Nullish(s)` | Optional(Nullable) |
| `NonOptional(s)` | reject Missing |
| `Readonly(s)` | no-op wrapper |
| `Default(s, v)` / `DefaultFunc` | Missing → value (no re-parse) |
| `Prefault(s, v)` / `PrefaultFunc` | Missing → value → parse |
| `Catch(s, v)` / `CatchFunc` | on failure → value |
| `Union(opts)` / `UnionOf(...)` | try in order |
| `DiscriminatedUnion(key, opts)` | disc map fast path |
| `Intersection(a, b)` | parse both + merge |
| `Lazy(fn)` | recursive |
| `Pipe(a, b)` | A then B |
| `Transform` / `TransformTo[T]` | map output |
| `Preprocess(fn, s)` | map input |
| `OverwriteSchema(s, fn)` | in-place after parse |
| `Refine` / `SuperRefine` | custom checks |
| `CheckSchema` / `Custom` | checks / any+pred |
| `ToStruct[T](s)` | map → struct |

## Parallel & errors

| API | Role |
|---|---|
| `ParseParallelSlice(ctx, elem, data, opts)` | worker pool |
| `Flatten` / `Treeify` / `Format` / `Prettify` | error shapes |
| `ToDotPath` | path string |
| `EnLocale` (+ `Es` `Fr` `De` `Ja` `Pt` `Zh`) | messages |

## Gin (`zgin`)

| API | Role |
|---|---|
| `BindJSON` / `BindJSONAny` | body |
| `BindQuery` / `BindURI` | query / params |
| `Validate` / `Get` | middleware |
| `AbortWithError` + `Options` | 400 shapes |

## Sentinels & config

| Symbol | Meaning |
|---|---|
| `Missing` / `IsMissing` | JS `undefined` |
| `ParseCtx{Error, ReportInput}` | per-parse options |
| `GetConfig` / locale / `CustomError` | global error maps |

:::info Package wrappers
`Optional`, `Default`, `Pipe`, `Transform`, … are **functions**, not methods — Go cannot add type params on methods.
:::
