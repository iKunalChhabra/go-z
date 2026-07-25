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
| go-zod | sequential | 416.0 | 392 | 6 |
| go-zod | `RunParallel` | 231.2 | 392 | 6 |
| validator | sequential | 607.1 | 0 | 0 |
| validator | `RunParallel` | 159.4 | 0 | 0 |
| zog | sequential | 1295 | 169 | 13 |
| zog | `RunParallel` | 606.8 | 170 | 13 |
| handwritten | sequential | 230.5 | 96 | 5 |

**Headline:** go-zod is **~1.5× faster than validator** sequential and **~3.1× faster than zog**.

## Nested

User + `address{city,zip}` + `[]tags` max 10

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-zod | sequential | 977.6 | 864 | 14 |
| go-zod | `RunParallel` | 555.8 | 864 | 14 |
| validator | sequential | 1090 | 88 | 4 |
| zog | sequential | 2783 | 492 | 36 |

**Headline:** go-zod is **~1.1× faster than validator** and **~2.8× faster than zog**.

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

At the same N, even sequential go-zod beats validator’s per-element loop (**4.24 ms** vs **6.09 ms**); parallel widens it to **2.75 ms**. zog takes **12.9 ms**.

:::tip When parallel pays off
Default `MinChunk` is 64. Below that, `ParseParallelSlice` stays sequential. See [Parallel validation](#/guide/parallel).
:::

## StringFormats & FailurePath

| Scenario | go-zod | validator | zog |
|---|---:|---:|---:|
| email+uuid+url | 890 ns | 1099 ns | 1810 ns |
| FailurePath (invalid FlatUser) | 6706 ns | 2001 ns | 2790 ns |

Format checks beat validator by ~20% since email, UUID, and the ISO date/time formats moved from backtracking regexes to hand-written matchers. Failure-path rendering is still heavier in go-zod (Zod-shaped finalize + JSON issues).

## Summary

| Scenario | vs validator | vs zog |
|---|---|---|
| FlatUser | **~1.5× faster** | **~3.1× faster** |
| Nested | **~1.1× faster** | **~2.8× faster** |
| StringFormats | **~1.2× faster** | **~2.0× faster** |
| Array 10k parallel | **~2.2× faster** | **~4.7× faster** |
| FailurePath | ~3.4× slower (richer errors) | ~2.4× slower |

Full tables, handwritten baseline, and methodology: **[BENCHMARKS.md](https://github.com/iKunalChhabra/go-zod/blob/main/BENCHMARKS.md)** in the repository root.
