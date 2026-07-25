# go-z Benchmarks

Comparative benchmarks for **go-z** vs **go-playground/validator/v10** vs **Oudwins/zog**, plus a handwritten FlatUser baseline.

Run from `bench/`:

```bash
cd bench && go test -bench=. -benchmem -count=9 -benchtime=500ms
```

## Machine

| | |
|---|---|
| OS / kernel | `Linux 6.12.94+ x86_64` |
| Go | `go1.26.5 linux/amd64` |
| `GOMAXPROCS` | unset → defaults to **4** (`nproc` = 4) |
| CPU | Intel(R) Xeon(R) Processor · 4 cores |
| Memory | 15 GiB |
| Date | 2026-07-25 |

Values are the **median of 9 runs** (`-benchtime=500ms`) for the per-value
benchmarks and the **median of 5 runs** (`-benchtime=300ms`) for the ArrayN
family, which is slow enough that more samples buy little.

This is a shared cloud VM, so treat the numbers as indicative of ratios rather
than absolute throughput. Within a single invocation the spread was ≤1.1× for
the sequential benchmarks; the `RunParallel` variants swing up to 1.4× and their
absolute values are the least trustworthy figures on this page. Two invocations
an hour apart differed by up to 30% on the same benchmark, which is why the
sampling protocol is stated rather than a single run reported.

## Notes

- **go-z** validates `map[string]any` (the JSON model), matching the untyped core it is ported from.
- **validator** validates typed Go structs with tags — no parse or coercion from maps, which is a real advantage in these benchmarks and a real limitation in a JSON pipeline.
- **zog** parses maps into structs (includes coercion / reflection), `github.com/Oudwins/zog@v0.22.2`.
- **Handwritten** is a minimal manual FlatUser check — the lower bound.
- Parallel array mode uses `z.ParseParallelSlice` with default `ParallelOpts` (`Workers=GOMAXPROCS`, `MinChunk=64`).

---

## FlatUser

`name` string min 5 · `email` · `age` int 0..150

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-z | sequential | 528 | 536 | 12 |
| go-z + `ToStruct` | sequential | 689 | 680 | 15 |
| go-z | `RunParallel` | 306 | 536 | 12 |
| validator | sequential | 637 | 0 | 0 |
| validator | `RunParallel` | 179 | 0 | 0 |
| zog | sequential | 1258 | 111 | 7 |
| zog | `RunParallel` | 431 | 111 | 7 |
| handwritten | sequential | 184 | 96 | 5 |
| handwritten | `RunParallel` | 67 | 96 | 5 |

**Headline:** go-z is **~1.2× faster than validator** and **~2.4× faster than zog** on flat objects, at 2.9× the cost of a handwritten check.

The `ToStruct` row is the fair comparison against the other two: go-z's plain
`Object` produces a `map[string]any`, while validator is handed a struct and zog
decodes into one. Ending in a struct costs 689 ns, which is **~1.8× faster than
zog** and **~1.2× slower than validator** — the honest numbers if a struct is what
your handler needs.

---

## Nested

user + `address{city,zip}` + `[]tags` max 10

| Library | Mode | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-z | sequential | 1184 | 1008 | 20 |
| go-z | `RunParallel` | 605 | 1009 | 20 |
| validator | sequential | 1112 | 88 | 4 |
| validator | `RunParallel` | 380 | 88 | 4 |
| zog | sequential | 2646 | 362 | 22 |
| zog | `RunParallel` | 928 | 361 | 22 |

**Headline:** on nested objects go-z is **~6% slower than validator** and **~2.2× faster than zog**. Nesting is where the payload-per-child model costs the most: a child payload and a path segment per level, against validator walking one struct with reflection.

---

## ArrayN (`[]FlatUser`)

| N | Library | ns/op | B/op | allocs/op |
|---:|---|---:|---:|---:|
| 100 | go-z sequential | 52884 | 53627 | 1200 |
| 100 | go-z `ParseParallelSlice` | 39772 | 58217 | 1217 |
| 100 | validator | 62654 | 6 | 0 |
| 100 | zog | 125411 | 11203 | 700 |
| 1000 | go-z sequential | 542204 | 536342 | 12001 |
| 1000 | go-z `ParseParallelSlice` | 323172 | 570798 | 12020 |
| 1000 | validator | 627579 | 67 | 0 |
| 1000 | zog | 1248999 | 111682 | 7000 |
| 10000 | go-z sequential | 5282721 | 5362096 | 120009 |
| 10000 | go-z `ParseParallelSlice` | 2447755 | 5691449 | 120027 |
| 10000 | validator | 6277177 | 688 | 0 |
| 10000 | zog | 12495714 | 1116529 | 70005 |

**Headline — parallel speedup (10k):** sequential **5.28 ms** → parallel **2.45 ms** ≈ **2.2×** on 4 workers, for ~6% more allocated bytes.

At N=10 000 even sequential go-z beats validator's per-element loop (**5.28 ms** vs **6.28 ms**); parallel widens that to **2.6×**.

---

## StringFormats

`email` + `uuid` + `url`

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| go-z | 798 | 688 | 8 |
| validator | 1088 | 145 | 1 |
| zog | 1713 | 274 | 9 |

**Headline:** go-z is **~1.4× faster than validator** and **~2.1× faster than zog**. Email, UUID and the ISO date/time formats use hand-written matchers instead of backtracking regexes (see `z/matchers.go`).

---

## FailurePath

Invalid FlatUser, including error construction and rendering to a string.

| Library | Rendering | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| go-z | `err.Error()` | 6845 | 5689 | 37 |
| validator | `err.Error()` | 1914 | 1923 | 29 |
| zog | `Issues.Prettify(errs)` | 2645 | 1072 | 27 |

The three are not doing identical work: go-z and validator render their default
error string, while zog returns an issue map with no `Error()` method, so
`Prettify` stands in as the closest analogue. Read this table as an
order-of-magnitude comparison, not a precise ranking.

**Headline:** failure-path rendering is heavier in go-z — **~3.6× validator**, **~2.6× zog** — because building structured issues means finalizing every message through the error-map chain and copying a large `Issue` per problem.

---

## Summary

| Scenario | go-z vs validator | go-z vs zog |
|---|---|---|
| FlatUser | **~1.2× faster** | **~2.4× faster** |
| Nested | ~6% slower | **~2.2× faster** |
| StringFormats | **~1.4× faster** | **~2.1× faster** |
| Array 10k parallel | **~2.6× faster** than validator's loop | **~5.1× faster** than zog's loop |
| FailurePath | ~3.6× slower (structured errors) | ~2.6× slower (structured errors) |

Parallel validation pays off once the element count clears `MinChunk` (default
64): **~2.2×** wall-time improvement at 10 000 elements on this 4-core host.

The failure path is where go-z loses, and it is the next optimization target:
finalizing messages through the error-map chain and copying an `Issue` per
problem dominates. Note that both comparisons validate different inputs — go-z
parses an untyped map, validator inspects a typed struct — so the happy-path
ratios above are the meaningful ones.
