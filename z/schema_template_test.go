package z

import (
	"regexp"
	"strings"
	"testing"
)

func TestTemplateLiteralBasic(t *testing.T) {
	hello := TemplateLiteral([]any{"hello"})
	if got, err := hello.Parse("hello"); err != nil || got != "hello" {
		t.Fatalf("%v %v", got, err)
	}
	if hello.SafeParse("hell").Success {
		t.Fatal("expected fail")
	}
}

func TestTemplateLiteralURL(t *testing.T) {
	url := TemplateLiteral([]any{
		"https://",
		String().Regex(regexp.MustCompile(`\w+`)),
		".",
		Enum("com", "net"),
	})
	if _, err := url.Parse("https://example.com"); err != nil {
		t.Fatal(err)
	}
	if url.SafeParse("http://example.com").Success {
		t.Fatal("expected fail")
	}
	if url.SafeParse("https://example.org").Success {
		t.Fatal("expected enum fail")
	}
}

func TestTemplateLiteralOptionalPart(t *testing.T) {
	sch := TemplateLiteral([]any{
		"v",
		Number(),
		Literal("a").Optional(),
	})
	if _, err := sch.Parse("v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := sch.Parse("v1a"); err != nil {
		t.Fatal(err)
	}
}

func TestTemplateLiteralPrimitiveParts(t *testing.T) {
	sch := TemplateLiteral([]any{"x", 1, true, nil})
	if _, err := sch.Parse("x1truenull"); err != nil {
		t.Fatal(err)
	}
}

// A part's regex and format checks have to reach the composed pattern. They used
// to set only Bag["format"], so the part compiled to [\s\S]* and the template
// accepted anything — including the example in this package's own doc comment.
func TestTemplateLiteralEnforcesPartConstraints(t *testing.T) {
	ids := TemplateLiteral([]any{"id-", String().Regex(regexp.MustCompile(`^\d+$`))})
	if _, err := ids.Parse("id-123"); err != nil {
		t.Fatalf("id-123 must match: %v", err)
	}
	if ids.SafeParse("id-abc").Success {
		t.Fatal("id-abc must not match a digits-only part")
	}

	uuids := TemplateLiteral([]any{"u_", String().UUID()})
	if _, err := uuids.Parse("u_550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatalf("valid uuid must match: %v", err)
	}
	if uuids.SafeParse("u_not-a-uuid").Success {
		t.Fatal("a non-uuid must not match a uuid part")
	}

	// An integer schema contributes an integer shape, not the general number one.
	ints := TemplateLiteral([]any{"v", Int()})
	if _, err := ints.Parse("v42"); err != nil {
		t.Fatalf("v42 must match: %v", err)
	}
	if ints.SafeParse("v4.2").Success {
		t.Fatal("v4.2 must not match an integer part")
	}
	if TemplateLiteral([]any{"u", Uint32()}).SafeParse("u-1").Success {
		t.Fatal("a negative number must not match an unsigned part")
	}
}

// A check with no pattern equivalent cannot be honoured by a single regexp, so
// the constructor says so instead of quietly dropping it.
func TestTemplateLiteralRejectsInexpressibleParts(t *testing.T) {
	cases := map[string]func(){
		"min length":  func() { TemplateLiteral([]any{"n", String().Min(5)}) },
		"with format": func() { TemplateLiteral([]any{"x", String().Email().Min(3)}) },
		"refinement":  func() { TemplateLiteral([]any{"r", String().Refine(func(string) bool { return true })}) },
		"two patterns": func() {
			TemplateLiteral([]any{"x", String().
				Regex(regexp.MustCompile(`^a+$`)).
				Regex(regexp.MustCompile(`^b+$`))})
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected a panic naming the unrepresentable check")
				}
				if msg, ok := r.(string); !ok || !strings.Contains(msg, "TemplateLiteral") {
					t.Fatalf("unhelpful panic: %v", r)
				}
			}()
			build()
		})
	}
}

// An unconstrained string is still a wildcard: that is the pattern it means.
func TestTemplateLiteralPlainStringStaysWildcard(t *testing.T) {
	s := TemplateLiteral([]any{"pre-", String()})
	if _, err := s.Parse("pre-anything at all"); err != nil {
		t.Fatal(err)
	}
	if s.SafeParse("nope").Success {
		t.Fatal("the literal prefix is still required")
	}
}
