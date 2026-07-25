package zod

import (
	"errors"
	"fmt"
	"testing"
)

func TestAsZodErrorUnwrapsWrapped(t *testing.T) {
	_, err := String().Min(5).Parse("hi")
	if err == nil {
		t.Fatal("want failure")
	}
	if _, ok := AsZodError(err); !ok {
		t.Fatal("direct error should be a ZodError")
	}

	wrapped := fmt.Errorf("decoding body: %w", err)
	zerr, ok := AsZodError(wrapped)
	if !ok {
		t.Fatal("wrapped error should still be a ZodError")
	}
	if len(zerr.Issues) != 1 || zerr.Issues[0].Code != IssueTooSmall {
		t.Fatalf("issues = %+v", zerr.Issues)
	}
	if !IsZodError(wrapped) {
		t.Fatal("IsZodError should follow the wrap chain")
	}

	// errors.As works directly too, since ZodError is a pointer type.
	var target *ZodError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should find *ZodError")
	}

	if _, ok := AsZodError(errors.New("unrelated")); ok {
		t.Fatal("unrelated error must not report as a ZodError")
	}
	if IsZodError(nil) {
		t.Fatal("nil is not a ZodError")
	}
}

// A schema whose declared edge does not match what it produces is a bug in the
// schema; Parse must say so rather than silently returning the zero value.
func TestParseReportsInternalTypeMismatch(t *testing.T) {
	// Overwrite replaces the value after the string parse, so the declared
	// string edge no longer holds.
	broken := String().Check(&Check{
		Name: "break",
		Fn:   func(p *Payload) { p.Value = 42 },
	})
	_, err := broken.Parse("hello")
	if err == nil {
		t.Fatal("want an error, not a silent zero value")
	}
	zerr, ok := AsZodError(err)
	if !ok {
		t.Fatalf("want *ZodError, got %T", err)
	}
	if zerr.Issues[0].Code != IssueCustom {
		t.Fatalf("issue = %+v", zerr.Issues[0])
	}
}

// Null and absent values still map to the zero value without an error, rather
// than tripping the internal-mismatch guard above.
func TestParseZeroValueForNullAndMissing(t *testing.T) {
	if got, err := Null().Parse(nil); err != nil || got != nil {
		t.Fatalf("null: %v %v", got, err)
	}
	// Optional's typed edge folds an absent value into a nil pointer...
	got, err := Optional(String()).Parse(Missing)
	if err != nil || got != nil {
		t.Fatalf("missing: %v %v", got, err)
	}
	// ...while the raw JSON model still carries the Missing sentinel.
	raw, err := Optional(String()).ParseAny(Missing)
	if err != nil || !IsMissing(raw) {
		t.Fatalf("missing raw: %v %v", raw, err)
	}
}
