package z

import (
	"regexp"
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
