# Intersection

Parse with both schemas, then deep-merge their outputs — Zod’s `z.intersection`.

## Intersection

```go
a := zod.Object(zod.Shape{"a": zod.String()})
b := zod.Object(zod.Shape{"b": zod.String()})
ab := zod.Intersection(a, b)

ab.Parse(map[string]any{"a": "foo", "b": "bar"})
// map[string]any{"a": "foo", "b": "bar"}
```

Both sides always run. Non-`unrecognized_keys` issues from either side are collected onto the result.

## Merge rules

`mergeValues` ports Zod’s merge:

| Left / Right | Result |
|---|---|
| Same reference / equal primitives (`===`) | Keep left |
| Equal `time.Time` (`Equal`) | Keep left |
| Both `map[string]any` | Merge keys; shared keys recurse |
| Both `[]any` of equal length | Element-wise merge |
| Both `[]any` of unequal length | **Unmergeable** |
| Conflicting primitives / types | **Unmergeable** |

```go
// Shared key merges recursively
left  := map[string]any{"meta": map[string]any{"x": 1}}
right := map[string]any{"meta": map[string]any{"y": 2}}
// → {"meta": {"x": 1, "y": 2}}
```

### Unrecognized keys

Strict / strip interplay follows Zod: only keys unrecognized by **both** sides are re-emitted as `unrecognized_keys`. A key known to either side is allowed.

```go
strictA := zod.Object(zod.Shape{"a": zod.String()}).Strict()
strictB := zod.Object(zod.Shape{"b": zod.String()}).Strict()
cat := zod.Intersection(strictA, strictB)

// {"a","b"} ok; {"a","b","c"} → unrecognized_keys ["c"]
```

## Unmergeable results

When merge fails, go-zod does **not** panic. It emits a `custom` issue so `SafeParse` stays safe:

```go
Issue{
    Code:    zod.IssueCustom,
    Message: "Unmergeable intersection results",
    Path:    mergePath, // e.g. ["field"] or [0]
}
```

:::warn Prefer Object.Extend / Merge for shapes
For object composition, `Object.Extend` / `Object.Merge` is usually clearer and avoids runtime merge failures. Use `Intersection` when you truly need both parsers to run (e.g. overlapping validations).
:::

## Optionality

If either side has OptIn / OptOut, the intersection inherits them (`OR`).

## Signatures

```go
func Intersection(left, right AnySchemaLike, params ...any) *IntersectionSchema

type IntersectionSchema struct {
    Left  AnySchemaLike
    Right AnySchemaLike
}
func (s *IntersectionSchema) Check(checks ...*Check) *IntersectionSchema
```
