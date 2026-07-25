# go-zod Benchmarks

Comparative benchmarks for **go-zod** vs **go-playground/validator/v10** vs **Oudwins/zog**, plus a handwritten FlatUser baseline.

Run from `bench/`:

```bash
cd bench && go test -bench=. -benchmem -count=3
```

## Machine

| | |
|---|---|
| OS / kernel | `Linux 6.12.94+ #1 SMP PREEMPT_DYNAMIC Wed Jul 22 16:29:09 UTC 2026 x86_64` |
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
| go-zod | sequential | 739.2 | 394 | 6 |
| go-zod | `RunParallel` | 285.0 | 397 | 6 |
| validator | sequential | 609.5 | 0 | 0 |
| validator | `RunParallel` | 205.5 | 0 | 0 |
| zog | sequential | 1314 | 169 | 13 |
| zog | `RunParallel` | 528.7 | 170 | 13 |
| handwritten | sequential | 230.5 | 96 | 5 |
| handwritten | `RunParallel` | 74.09 | 96 | 5 |

**Headline:** go-zod ≈ **1.2×** validator on FlatUser sequential; **~1.8× faster than zog**.

---

## Nested

user + `address{city,zip}` + `[]tags` max 10

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-zod | sequential | 1281 | 870 | 14 |
| go-zod | `RunParallel` | 459.3 | 875 | 14 |
| validator | sequential | 1090 | 88 | 4 |
| validator | `RunParallel` | 345.4 | 89 | 4 |
| zog | sequential | 2829 | 491 | 36 |
| zog | `RunParallel` | 810.6 | 494 | 36 |

**Headline:** go-zod ≈ **1.2×** validator; **~2.2× faster than zog** on nested objects.

---

## ArrayN (`[]FlatUser`)

| N | Library | ns/op | B/op | allocs/op |
|---:|---|---:|---:|---:|
| 100 | go-zod sequential | 71527 | 39484 | 600 |
| 100 | go-zod `ParseParallelSlice` | 43728 | 44188 | 613 |
| 100 | validator | 61757 | 0 | 0 |
| 100 | zog | 130025 | 16956 | 1300 |
| 1000 | go-zod sequential | 735206 | 395480 | 6001 |
| 1000 | go-zod `ParseParallelSlice` | 328842 | 434026 | 6017 |
| 1000 | validator | 618746 | 0 | 0 |
| 1000 | zog | 1303859 | 169459 | 13001 |
| 10000 | go-zod sequential | 7230341 | 3945356 | 60011 |
| 10000 | go-zod `ParseParallelSlice` | 2936189 | 4272305 | 60025 |
| 10000 | validator | 6167454 | 198 | 0 |
| 10000 | zog | 13014799 | 1693240 | 130007 |

**Headline — parallel speedup (10k):** sequential **7.23 ms** → parallel **2.94 ms** ≈ **2.5×** on 4 workers.

At N=10000, `ParseParallelSlice` also beats validator’s per-element loop (**2.94 ms** vs **6.17 ms**).

---

## StringFormats

`email` + `uuid` + `url`

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| go-zod | 1157 | 548 | 7 |
| validator | 1090 | 144 | 1 |
| zog | 1817 | 330 | 15 |

**Headline:** go-zod within **~6%** of validator; **~1.6× faster than zog**.

---

## FailurePath

Invalid FlatUser; includes error construction / stringification.

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| go-zod | 7192 | 5611 | 31 |
| validator | 1987 | 1921 | 29 |
| zog | 2788 | 1121 | 33 |

**Headline:** failure-path error rendering is heavier in go-zod (Zod-shaped issue JSON / finalize chain) — expected vs tag validators.

---

## Summary

| Scenario | go-zod vs validator | go-zod vs zog |
|---|---|---|
| FlatUser | ~1.2× slower | ~1.8× faster |
| Nested | ~1.2× slower | ~2.2× faster |
| StringFormats | ~1.06× slower | ~1.6× faster |
| Array 10k parallel | **~2.1× faster** than validator loop | **~4.4× faster** than zog loop |
| FailurePath | slower (richer errors) | slower (richer errors) |

Parallel validation pays off once element count clears `MinChunk` (default 64); measured **~2.5×** wall-time improvement at 10k elements on this 4-core host.
