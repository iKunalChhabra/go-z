# Zod v4 test parity

go-z ports behavioral cases from Zod’s `packages/zod/src/v4/classic/tests/*.test.ts`.

## Suite layout

| Go file | Zod sources (primary) |
|---|---|
| `parity_string_test.go` | `string.test.ts`, `string-formats.test.ts`, `url.test.ts`, `datetime.test.ts` |
| `parity_number_test.go` | `number.test.ts`, `nan.test.ts`, validations |
| `parity_object_test.go` | `object.test.ts`, `pickomit.test.ts`, `partial.test.ts` |
| `parity_optional_test.go` | `optional.test.ts`, `nullable.test.ts`, `nonoptional.test.ts` |
| `parity_default_test.go` | `default.test.ts`, `prefault.test.ts`, `catch.test.ts` |
| `parity_array_tuple_test.go` | `array.test.ts`, `tuple.test.ts` |
| `parity_union_test.go` | `union.test.ts`, `discriminated-unions.test.ts` |
| `parity_refine_test.go` | `refine.test.ts`, `nested-refine.test.ts`, `custom.test.ts`, `continuability.test.ts` |
| `parity_pipe_transform_test.go` | `pipe.test.ts`, `transform.test.ts`, `preprocess.test.ts` |
| `parity_error_test.go` | `error.test.ts`, `error-utils.test.ts` |
| `parity_coerce_test.go` | `coerce.test.ts` |
| `parity_collections_test.go` | `record.test.ts`, `map.test.ts`, `set.test.ts` |
| `parity_misc_test.go` | `literal`, `enum`, `bigint`, `date`, `anyunknown`, `primitive`, `intersection`, `lazy`/`recursive-types`, `registries`, `global-config` |
| `concurrent_test.go` | go-z concurrency model (not in Zod) |

Run:

```bash
go test -count=1 -run 'Parity|Concurrent' .
go test -race -count=1 ./...
```

## Intentionally skipped (unsupported in v0)

Marked with `t.Skip` in the parity files:

- Async parse / async refinements / async codecs
- `fromJSONSchema` (import); `ToJSONSchema` export is supported
- `z.function`, `z.promise`, `z.file`, `z.symbol`, `z.instanceof`
- Brands / TypeScript-only assignability / Mini API
- ~~Codecs / `encode` / `decode`~~ (supported via `Codec`, `Decode`/`Encode`, `InvertCodec`)
- ~~JSON Schema convert~~ (`ToJSONSchema` supported; `fromJSONSchema` not)
- ~~Template literals~~ (supported via `TemplateLiteral`)
- ~~Some URL hostname/protocol constraint options~~ (supported via `URLOpts.Hostname` / `URLOpts.Protocol`, `HttpURL`)

## Concurrency

Schemas are immutable after build and **safe to share across goroutines**.
Helpers: `Share`, `ConcurrentBatch`, `ConcurrentParseAny`, `ParseParallelSlice`.
See comments in `concurrent.go`.
