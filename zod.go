// Package zod is a native Go port of Zod v4 (https://zod.dev).
//
// It mirrors Zod's architecture exactly:
//
//   - a ParsePayload {value, issues} threaded through parsing — issues
//     accumulate instead of aborting, containers prefix child paths;
//   - schemas built from a Def (serializable definition), a bare type parser,
//     and a "run" function that layers checks on top (zero-check schemas fast
//     path run == parse);
//   - composable checks with when/abort/continue semantics and onattach hooks;
//   - Zod's exact issue taxonomy (invalid_type, too_small, too_big,
//     invalid_format, not_multiple_of, unrecognized_keys, invalid_union,
//     invalid_key, invalid_element, invalid_value, custom) and error-map
//     resolution chain (per-check → per-parse → global custom → locale);
//   - the classic fluent API: zod.String().Min(5).Email().Optional().
//
// Schemas are immutable after construction and safe for concurrent use from
// any number of goroutines without locking.
package zod
