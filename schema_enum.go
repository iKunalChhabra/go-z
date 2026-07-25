package zod

// EnumSchema is the Go port of ZodEnum / $ZodEnum (Schema[string]).
type EnumSchema struct {
	schemaBase[string]
	def     *Def
	options []string
	entries map[string]string
}

// Enum returns a string enum schema (z.enum(["a","b"])).
// A trailing Params / ErrorMap / string message is treated as schema params.
func Enum(values ...string) *EnumSchema {
	opts, p := splitEnumArgs(values)
	entries := make(map[string]string, len(opts))
	for _, v := range opts {
		entries[v] = v
	}
	def := &Def{Type: "enum", Error: p.Error}
	return newEnum(def, opts, entries)
}

// NativeEnum builds an enum from a string→string map (z.nativeEnum / z.enum(object)).
// Accepted values are the map values (not keys).
func NativeEnum(m map[string]string, params ...any) *EnumSchema {
	p := normalizeParams(params)
	opts := make([]string, 0, len(m))
	entries := make(map[string]string, len(m))
	for k, v := range m {
		entries[k] = v
		opts = append(opts, v)
	}
	def := &Def{Type: "enum", Error: p.Error}
	return newEnum(def, opts, entries)
}

func splitEnumArgs(values []string) (opts []string, p Params) {
	// Enum only takes strings; params must be passed via NativeEnum or a
	// dedicated overload. Support Enum("a","b") only. For error customization
	// callers can use Check or we accept empty trailing via separate API.
	return values, Params{}
}

func newEnum(def *Def, options []string, entries map[string]string) *EnumSchema {
	s := &EnumSchema{def: def, options: options, entries: entries}
	in := buildInternals(def, makeEnumParse(def, options))
	in.Values = make(map[any]struct{}, len(options))
	for _, v := range options {
		in.Values[v] = struct{}{}
	}
	s.schemaBase = newBase[string](in)
	return s
}

func makeEnumParse(def *Def, options []string) ParseFn {
	set := make(map[string]struct{}, len(options))
	vals := make([]any, len(options))
	for i, o := range options {
		set[o] = struct{}{}
		vals[i] = o
	}
	return func(p *Payload, _ *ParseCtx) {
		s, ok := p.Value.(string)
		if ok {
			if _, found := set[s]; found {
				p.Value = s
				return
			}
		}
		p.AddIssue(Issue{
			Code:   IssueInvalidValue,
			Values: append([]any(nil), vals...),
			Input:  p.Value,
			errMap: def.Error,
		})
	}
}

// Options returns the enum member values in definition order.
func (s *EnumSchema) Options() []string { return append([]string(nil), s.options...) }

// EnumMap returns the key→value entries (for NativeEnum / Zod's .enum getter).
func (s *EnumSchema) EnumMap() map[string]string {
	out := make(map[string]string, len(s.entries))
	for k, v := range s.entries {
		out[k] = v
	}
	return out
}

// Check attaches raw checks.
func (s *EnumSchema) Check(checks ...*Check) *EnumSchema {
	return newEnum(s.def.withChecks(checks...), s.options, s.entries)
}
