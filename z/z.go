// Package z provides schema-first validation for Go: schemas are values you
// build, compose and share, and parsing returns a typed result or a structured
// error describing every failure.
//
// The design is ported from Zod v4 (https://zod.dev). go-z is an independent
// project, not affiliated with or endorsed by Zod or its authors.
//
// The architecture it inherits:
//
//   - a ParsePayload {value, issues} threaded through parsing — issues
//     accumulate instead of aborting, containers prefix child paths;
//   - schemas built from a Def (serializable definition), a bare type parser,
//     and a "run" function that layers checks on top (zero-check schemas fast
//     path run == parse);
//   - composable checks with when/abort/continue semantics and onattach hooks;
//   - exact issue taxonomy (invalid_type, too_small, too_big,
//     invalid_format, not_multiple_of, unrecognized_keys, invalid_union,
//     invalid_key, invalid_element, invalid_value, custom) and error-map
//     resolution chain (per-check → per-parse → global custom → locale);
//   - a fluent API: z.String().Min(5).Email(), plus wrappers
//     like Optional(schema) / Default(schema, v)
//
// # Concurrency
//
// Schemas are immutable after construction and safe for concurrent use from
// any number of goroutines without locking — share one schema and call
// Parse/SafeParse freely. See concurrent.go for the safety model and helpers
// (Share, ConcurrentBatch, ConcurrentParseAny, ParseParallelSlice).
package z
