# Intersection

Parse with both schemas, then deep-merge their outputs — `z.intersection`.

## Intersection

```go
a := z.Object(z.Shape{"a": z.String()})
b := z.Object(z.Shape{"b": z.String()})
ab := z.Intersection(a, b)

ab.Parse(map[string]any{"a": "foo", "b": "bar"})
// map[string]any{"a": "foo", "b": "bar"}
```

Both sides always run. Non-`unrecognized_keys` issues from either side are collected onto the result.

## Merge rules

`mergeValues` ports merge:

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
left := map[string]any{"meta": map[string]any{"x": 1}}
right := map[string]any{"meta": map[string]any{"y": 2}}
// → {"meta": {"x": 1, "y": 2}}
```

### Unrecognized keys

Each side only knows its own shape, so a strict object inside an intersection
flags the *other* side's keys. When both sides are objects, the intersection's
recognized set is the **union of their shapes**, and only a key outside that union
is reported:

```go
strictA := z.Object(z.Shape{"a": z.String()}).Strict()
strictB := z.Object(z.Shape{"b": z.String()}).Strict()
cat := z.Intersection(strictA, strictB)

// {"a","b"} ok; {"a","b","c"} → unrecognized_keys ["c"]
```

A strict side keeps its strictness even when the other side is loose:

```go
mixed := z.Intersection(
    z.Object(z.Shape{"a": z.String()}).Strict(),
    z.Object(z.Shape{"b": z.String()}).Loose(),
)
// {"a","b"} ok; {"a","b","c"} → unrecognized_keys ["c"] — the strict side rejects it
```

When a side is not a plain object (a union, a pipe) there is no shape to union, and
the fallback is narrower: only keys **both** sides flagged are reported.

:::info Difference from the original library
TypeScript Zod runs both sides independently and surfaces each side's
`unrecognized_keys`, so intersecting two strict objects rejects every input. go-z
unions the shapes instead, which makes the combination usable.
:::

## Unmergeable results

When merge fails, go-z does **not** panic. It emits a `custom` issue so `SafeParse` stays safe:

```go
Issue{
    Code:    z.IssueCustom,
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
