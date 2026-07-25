# Benchmarks

Summary of comparative numbers from the repo’s [`BENCHMARKS.md`](https://github.com/iKunalChhabra/go-zod/blob/main/BENCHMARKS.md). Re-run locally:

```bash
cd bench && go test -bench=. -benchmem -count=3
```

## Machine (recorded)

| | |
|---|---|
| OS | Linux 6.12 · x86_64 |
| Go | 1.22.2 |
| CPU | 4-core Xeon · `GOMAXPROCS=4` |
| Date | 2026-07-25 |

Medians of 3 runs. go-zod validates `map[string]any`; validator uses tagged structs; zog parses maps into structs.

## FlatUser

`name` min 5 · `email` · `age` 0..150

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-zod | sequential | 739.2 | 394 | 6 |
| go-zod | `RunParallel` | 285.0 | 397 | 6 |
| validator | sequential | 609.5 | 0 | 0 |
| validator | `RunParallel` | 205.5 | 0 | 0 |
| zog | sequential | 1314 | 169 | 13 |
| zog | `RunParallel` | 528.7 | 170 | 13 |
| handwritten | sequential | 230.5 | 96 | 5 |

**Headline:** go-zod ≈ **1.2×** validator sequential; **~1.8× faster than zog**.

## Nested

User + `address{city,zip}` + `[]tags` max 10

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-zod | sequential | 1281 | 870 | 14 |
| go-zod | `RunParallel` | 459.3 | 875 | 14 |
| validator | sequential | 1090 | 88 | 4 |
| zog | sequential | 2829 | 491 | 36 |

**Headline:** go-zod ≈ **1.2×** validator; **~2.2× faster than zog**.

## Array 10k — parallel ~2.5×

`ParseParallelSlice` vs sequential on `[]FlatUser`:

| N | Mode | ns/op |
|---:|---|---:|
| 100 | sequential | 71 527 |
| 100 | parallel | 43 728 |
| 1 000 | sequential | 735 206 |
| 1 000 | parallel | 328 842 |
| 10 000 | sequential | 7 230 341 |
| 10 000 | parallel | **2 936 189** |

At **N=10 000**: sequential **7.23 ms** → parallel **2.94 ms** ≈ **2.5×** on 4 workers.

At the same N, parallel go-zod also beats validator’s per-element loop (**2.94 ms** vs **6.17 ms**) and zog (**13.1 ms**).

:::tip When parallel pays off
Default `MinChunk` is 64. Below that, `ParseParallelSlice` stays sequential. See [Parallel validation](#/guide/parallel).
:::

## StringFormats & FailurePath

| Scenario | go-zod | validator | zog |
|---|---:|---:|---:|
| email+uuid+url | 1157 ns | 1090 ns | 1817 ns |
| FailurePath (invalid FlatUser) | 7192 ns | 1987 ns | 2788 ns |

Format checks are within ~6% of validator. Failure-path rendering is heavier in go-zod (Zod-shaped finalize + JSON issues) — expected.

## Summary

| Scenario | vs validator | vs zog |
|---|---|---|
| FlatUser | ~1.2× slower | ~1.8× faster |
| Nested | ~1.2× slower | ~2.2× faster |
| StringFormats | ~1.06× slower | ~1.6× faster |
| Array 10k parallel | **~2.1× faster** | **~4.4× faster** |
| FailurePath | slower (richer errors) | slower |

Full tables, handwritten baseline, and methodology: **[BENCHMARKS.md](https://github.com/iKunalChhabra/go-zod/blob/main/BENCHMARKS.md)** in the repository root.
