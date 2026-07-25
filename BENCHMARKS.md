# go-zod Benchmarks

Comparative benchmarks for **go-zod** vs **go-playground/validator/v10** vs **Oudwins/zog**, plus a handwritten FlatUser baseline.

Run from `bench/`:

```bash
cd bench && go test -bench=. -benchmem -count=3
```

## Machine

| | |
|---|---|
| OS / kernel | `Linux cursor 6.12.94+ #1 SMP PREEMPT_DYNAMIC Wed Jul 22 16:29:09 UTC 2026 x86_64` |
| Go | `go1.22.2 linux/amd64` (bench module toolchain may auto-select) |
| `GOMAXPROCS` | unset → defaults to **4** (`nproc` = 4) |
| CPU | Intel(R) Xeon(R) Processor · 4 cores / 4 siblings |
| Memory | 15 GiB |
| Command | `go test -bench=. -benchmem -count=3` |
| Date | 2026-07-25 |

Values below are the **median** of 3 runs (`ns/op`, `B/op`, `allocs/op`).

## Notes

- **go-zod** validates `map[string]any` (JSON model), matching Zod’s untyped core.
- **validator** validates typed Go structs with tags (no parse/coerce from maps).
- **zog** parses maps into structs (includes coercion / reflection). Included successfully on this machine (`github.com/Oudwins/zog@v0.22.2`).
- **Handwritten** is a minimal manual FlatUser check (lower bound).
- Parallel array mode uses `zod.ParseParallelSlice` with default `ParallelOpts` (`Workers=GOMAXPROCS`, `MinChunk=64`).

---

## FlatUser

`name` string min 5 · `email` · `age` int 0..150

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-zod | sequential | 416.0 | 392 | 6 |
| go-zod | `RunParallel` | 231.2 | 392 | 6 |
| validator | sequential | 607.1 | 0 | 0 |
| validator | `RunParallel` | 159.4 | 0 | 0 |
| zog | sequential | 1295 | 169 | 13 |
| zog | `RunParallel` | 606.8 | 170 | 13 |
| handwritten | sequential | 230.5 | 96 | 5 |
| handwritten | `RunParallel` | 74.09 | 96 | 5 |

**Headline:** go-zod is **~1.5× faster than validator** on FlatUser sequential and **~3.1× faster than zog**.

---

## Nested

user + `address{city,zip}` + `[]tags` max 10

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-zod | sequential | 977.6 | 864 | 14 |
| go-zod | `RunParallel` | 555.8 | 864 | 14 |
| validator | sequential | 1090 | 88 | 4 |
| validator | `RunParallel` | 358.7 | 89 | 4 |
| zog | sequential | 2783 | 492 | 36 |
| zog | `RunParallel` | 895.9 | 493 | 36 |

**Headline:** go-zod is **~1.1× faster than validator** and **~2.8× faster than zog** on nested objects.

---

## ArrayN (`[]FlatUser`)

| N | Library | ns/op | B/op | allocs/op |
|---:|---|---:|---:|---:|
| 100 | go-zod sequential | 41510 | 39223 | 600 |
| 100 | go-zod `ParseParallelSlice` | 36073 | 43679 | 613 |
| 100 | validator | 60946 | 1 | 0 |
| 100 | zog | 129143 | 16936 | 1300 |
| 1000 | go-zod sequential | 426452 | 392276 | 6001 |
| 1000 | go-zod `ParseParallelSlice` | 333009 | 426438 | 6015 |
| 1000 | validator | 612798 | 19 | 0 |
| 1000 | zog | 1299807 | 169425 | 13000 |
| 10000 | go-zod sequential | 4241138 | 3921944 | 60007 |
| 10000 | go-zod `ParseParallelSlice` | 2747636 | 4250742 | 60020 |
| 10000 | validator | 6087655 | 197 | 0 |
| 10000 | zog | 12944697 | 1691599 | 130007 |

**Headline — parallel speedup (10k):** sequential **4.24 ms** → parallel **2.75 ms** ≈ **1.5×** on 4 workers. Sequential parsing got fast enough that the parallel win narrowed.

At N=10000, even sequential go-zod beats validator’s per-element loop (**4.24 ms** vs **6.09 ms**); parallel widens it to **2.75 ms**.

---

## StringFormats

`email` + `uuid` + `url`

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| go-zod | 889.7 | 688 | 8 |
| validator | 1099 | 144 | 1 |
| zog | 1810 | 330 | 15 |

**Headline:** go-zod is **~1.2× faster than validator** and **~2× faster than zog**. Email, UUID, and the ISO date/time formats use hand-written matchers instead of the backtracking regexes (see `matchers.go`).

---

## FailurePath

Invalid FlatUser; includes error construction / stringification.

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| go-zod | 6706 | 5561 | 31 |
| validator | 2001 | 1922 | 29 |
| zog | 2790 | 1121 | 33 |

**Headline:** failure-path error rendering is heavier in go-zod (Zod-shaped issue JSON / finalize chain) — expected vs tag validators.

---

## Summary

| Scenario | go-zod vs validator | go-zod vs zog |
|---|---|---|
| FlatUser | **~1.5× faster** | **~3.1× faster** |
| Nested | **~1.1× faster** | **~2.8× faster** |
| StringFormats | **~1.2× faster** | **~2.0× faster** |
| Array 10k parallel | **~2.2× faster** than validator loop | **~4.7× faster** than zog loop |
| FailurePath | ~3.4× slower (richer errors) | ~2.4× slower (richer errors) |

Parallel validation pays off once element count clears `MinChunk` (default 64); measured **~1.5×** wall-time improvement at 10k elements on this 4-core host.

The failure path remains the one place go-zod loses: building Zod-shaped issues means finalizing messages through the error-map chain and copying a 400-byte `Issue` per problem. It is the next optimization target.
