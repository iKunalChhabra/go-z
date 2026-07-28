package z

import "slices"

// RecordSchema is the schema.
// Validates map[string]any with a key schema and a value schema.
type RecordSchema struct {
	schemaBase[map[string]any]
	def         *Def
	keySchema   AnySchemaLike
	valueSchema AnySchemaLike
	loose       bool // when true, invalid keys pass through (mode:"loose")
}

// Record returns a record schema (z.record(key, value)).
func Record(keySchema, valueSchema AnySchemaLike, params ...any) *RecordSchema {
	p := normalizeParams(params)
	def := &Def{Type: "record", Error: p.Error}
	return newRecord(def, keySchema, valueSchema, false)
}

func newRecord(def *Def, keySchema, valueSchema AnySchemaLike, loose bool) *RecordSchema {
	s := &RecordSchema{
		def:         def,
		keySchema:   keySchema,
		valueSchema: valueSchema,
		loose:       loose,
	}
	parse := makeRecordParse(def, keySchema, valueSchema, loose)
	s.schemaBase = newBase[map[string]any](buildInternals(def, parse))
	return s
}

func makeRecordParse(def *Def, keySchema, valueSchema AnySchemaLike, loose bool) ParseFn {
	var keyIn, valIn *Internals
	if keySchema != nil {
		keyIn = keySchema.Internals()
	}
	if valueSchema != nil {
		valIn = valueSchema.Internals()
	}
	return func(p *Payload, ctx *ParseCtx) {
		input, ok := asStringMap(p.Value)
		if !ok {
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "record",
				Input:    p.Value,
				errMap:   def.Error,
			})
			return
		}

		out := make(map[string]any)

		// Exhaustive key schema (enum/literal Values set): require every key,
		// reject unrecognized keys (record enum path).
		if keyIn != nil && len(keyIn.Values) > 0 {
			expectedKeys := slices.Sorted(stringValues(keyIn.Values))
			seen := make(map[string]struct{}, len(expectedKeys))
			for _, k := range expectedKeys {
				seen[k] = struct{}{}
				// Validate key itself.
				kp := AcquirePayload(k)
				kp.parseCtx = ctx
				keyIn.Run(kp, ctx)
				if len(kp.Issues) > 0 {
					p.AddIssue(Issue{
						Code:   IssueInvalidKey,
						Origin: "record",
						Issues: finalizeNestedIssues(kp.Issues, ctx),
						Input:  k,
						Path:   []any{k},
						errMap: def.Error,
					})
					ReleasePayload(kp)
					continue
				}
				outKey, _ := kp.Value.(string)
				ReleasePayload(kp)
				if outKey == "" {
					outKey = k
				}

				raw, present := input[k]
				if !present {
					raw = missingSentinel
				}
				if valIn != nil {
					childOut, _ := RunChild(valIn, p, raw, ctx, k)
					if IsMissing(childOut) {
						// Optional value on an absent key: omit from output.
						continue
					}
					out[outKey] = childOut
				} else if !IsMissing(raw) {
					out[outKey] = raw
				}
			}
			if unrecognized := slices.Sorted(unknownKeys(input, seen)); len(unrecognized) > 0 {
				p.AddIssue(Issue{
					Code:   IssueUnrecognizedKeys,
					Keys:   unrecognized,
					Input:  p.Value,
					errMap: def.Error,
				})
			}
			p.Value = out
			return
		}

		// Open key schema: validate each enumerable key.
		keys := slices.Sorted(unknownKeys(input, nil))

		for _, k := range keys {
			if keyIn != nil {
				kp := AcquirePayload(k)
				kp.parseCtx = ctx
				keyIn.Run(kp, ctx)
				if len(kp.Issues) > 0 {
					if loose {
						out[k] = input[k]
						ReleasePayload(kp)
						continue
					}
					p.AddIssue(Issue{
						Code:   IssueInvalidKey,
						Origin: "record",
						Issues: finalizeNestedIssues(kp.Issues, ctx),
						Input:  k,
						Path:   []any{k},
						errMap: def.Error,
					})
					ReleasePayload(kp)
					continue
				}
				outKey := k
				if s, ok := kp.Value.(string); ok {
					outKey = s
				}
				ReleasePayload(kp)

				if valIn != nil {
					childOut, _ := RunChild(valIn, p, input[k], ctx, k)
					out[outKey] = childOut
				} else {
					out[outKey] = input[k]
				}
			} else if valIn != nil {
				childOut, _ := RunChild(valIn, p, input[k], ctx, k)
				out[k] = childOut
			} else {
				out[k] = input[k]
			}
		}
		p.Value = out
	}
}

// finalizeNestedIssues finalizes child issues for embedding in invalid_key /
// invalid_element (maps iss → finalizeIssue before nesting).
func finalizeNestedIssues(raw []Issue, ctx *ParseCtx) []Issue {
	cfg := GetConfig()
	out := make([]Issue, len(raw))
	for i, iss := range raw {
		out[i] = FinalizeIssue(iss, ctx, cfg)
	}
	return out
}

// Loose returns a record that passes through keys failing the key schema.
func (r *RecordSchema) Loose() *RecordSchema {
	return newRecord(r.def, r.keySchema, r.valueSchema, true)
}

// Check attaches raw checks.
func (r *RecordSchema) Check(checks ...*Check) *RecordSchema {
	return newRecord(r.def.withChecks(checks...), r.keySchema, r.valueSchema, r.loose)
}
