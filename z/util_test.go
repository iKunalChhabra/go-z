package z

import (
	"testing"
	"time"
)

func TestParsedTypeTypedNilTimePointer(t *testing.T) {
	// Regression: *time.Time matched the date case before the typed-nil
	// pointer guard, so a nil *time.Time reported "date" instead of "null".
	var nilTime *time.Time
	if got := ParsedType(nilTime); got != "null" {
		t.Fatalf("ParsedType(nil *time.Time) = %q, want %q", got, "null")
	}
	now := time.Now()
	if got := ParsedType(&now); got != "date" {
		t.Fatalf("ParsedType(&time.Time) = %q, want %q", got, "date")
	}
	if got := ParsedType(now); got != "date" {
		t.Fatalf("ParsedType(time.Time) = %q, want %q", got, "date")
	}
}
