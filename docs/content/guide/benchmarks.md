# Benchmarks

Summary of comparative numbers from the repo's [`BENCHMARKS.md`](https://github.com/iKunalChhabra/go-z/blob/main/BENCHMARKS.md). Re-run locally:

```bash
cd bench && go test -bench=. -benchmem -count=9 -benchtime=500ms
```

## Machine (recorded)

| | |
|---|---|
| OS | Linux 6.12 · x86_64 |
| Go | 1.26.5 |
| CPU | 4-core Xeon · `GOMAXPROCS=4` |
| Date | 2026-07-25 |

Medians of 9 runs (5 for the array family). go-z validates `map[string]any`; validator uses tagged structs; zog parses maps into structs.

:::warn Shared host
These were measured on a shared cloud VM. Ratios between libraries held steady across invocations; absolute numbers moved by up to 30%, and the `RunParallel` figures are the noisiest on the page. Re-run on your own hardware before making a decision on the margins.
:::

## FlatUser

`name` min 5 · `email` · `age` 0..150

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-z | sequential | 528 | 536 | 12 |
| go-z | `RunParallel` | 306 | 536 | 12 |
| validator | sequential | 637 | 0 | 0 |
| validator | `RunParallel` | 179 | 0 | 0 |
| zog | sequential | 1258 | 111 | 7 |
| zog | `RunParallel` | 431 | 111 | 7 |
| handwritten | sequential | 184 | 96 | 5 |

**Headline:** go-z is **~1.2× faster than validator** sequential and **~2.4× faster than zog**.

## Nested

User + `address{city,zip}` + `[]tags` max 10

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-z | sequential | 1184 | 1008 | 20 |
| go-z | `RunParallel` | 605 | 1009 | 20 |
| validator | sequential | 1112 | 88 | 4 |
| zog | sequential | 2646 | 362 | 22 |

**Headline:** on nested objects go-z is **~6% slower than validator** and **~2.2× faster than zog**. Each level of nesting costs a child payload and a path segment, which is the price of per-issue paths.

## Array 10k — parallel ~2.2×

`ParseParallelSlice` vs sequential on `[]FlatUser`:

| N | Mode | ns/op |
|---:|---|---:|
| 100 | sequential | 52 884 |
| 100 | parallel | 39 772 |
| 1 000 | sequential | 542 204 |
| 1 000 | parallel | 323 172 |
| 10 000 | sequential | 5 282 721 |
| 10 000 | parallel | **2 447 755** |

At **N=10 000**: sequential **5.28 ms** → parallel **2.45 ms** ≈ **2.2×** on 4 workers.

At the same N, even sequential go-z beats validator's per-element loop (**5.28 ms** vs **6.28 ms**); parallel widens it to 2.6×. zog takes **12.5 ms**.

:::tip When parallel pays off
Default `MinChunk` is 64. Below that, `ParseParallelSlice` stays sequential. See [Parallel validation](#/guide/parallel).
:::

## StringFormats & FailurePath

| Scenario | go-z | validator | zog |
|---|---:|---:|---:|
| email+uuid+url | 798 ns | 1088 ns | 1713 ns |
| FailurePath (invalid FlatUser) | 6845 ns | 1914 ns | 2645 ns |

Format checks beat validator by ~35% since email, UUID and the ISO date/time formats moved from backtracking regexes to hand-written matchers. Failure-path rendering is heavier in go-z: every message is finalized through the error-map chain and an `Issue` is copied per problem. The three failure-path benchmarks also render differently — `err.Error()` for go-z and validator, `Prettify` for zog, which has no `Error()` on its issue map — so read that row as an order of magnitude.

## Summary

| Scenario | vs validator | vs zog |
|---|---|---|
| FlatUser | **~1.2× faster** | **~2.4× faster** |
| Nested | ~6% slower | **~2.2× faster** |
| StringFormats | **~1.4× faster** | **~2.1× faster** |
| Array 10k parallel | **~2.6× faster** | **~5.1× faster** |
| FailurePath | ~3.6× slower (structured errors) | ~2.6× slower |

Full tables, handwritten baseline, and methodology: **[BENCHMARKS.md](https://github.com/iKunalChhabra/go-z/blob/main/BENCHMARKS.md)** in the repository root.
