package z

import (
	"regexp"
	"testing"
)

func TestNormalizeParamsSkipsNilEntriesAndPointers(t *testing.T) {
	var nilOpts *URLOpts
	var nilParams *Params
	got := normalizeParams([]any{nil, nilOpts, nilParams, "boom"})
	if got.Error == nil {
		t.Fatal("string message should survive nil entries")
	}
	if got.Abort {
		t.Fatal("abort should stay false")
	}
}

func TestNormalizeParamsAcceptsPointerOpts(t *testing.T) {
	got := normalizeParams([]any{&Params{Abort: true, Coerce: true}})
	if !got.Abort || !got.Coerce {
		t.Fatalf("pointer Params not merged: %+v", got)
	}
	withOpts := normalizeParams([]any{&JWTOpts{Alg: "HS256", Abort: true}})
	if !withOpts.Abort {
		t.Fatalf("pointer format opts not merged: %+v", withOpts)
	}
}

// Pointer format opts now reach the format helper itself, not just its
// Error/Abort fields.
func TestPointerFormatOptsReadTypedFields(t *testing.T) {
	dashed := String().Check(FormatMAC(&MACOpts{Delimiter: "-"}))
	if _, err := dashed.Parse("00-1A-2B-3C-4D-5E"); err != nil {
		t.Fatalf("dashed MAC: %v", err)
	}
	if dashed.SafeParse("00:1A:2B:3C:4D:5E").Success {
		t.Fatal("colon MAC should fail when delimiter is '-'")
	}

	httpsOnly := String().URL(&URLOpts{Protocol: regexp.MustCompile(`^https$`)})
	if _, err := httpsOnly.Parse("https://example.com"); err != nil {
		t.Fatalf("https URL: %v", err)
	}
	if httpsOnly.SafeParse("http://example.com").Success {
		t.Fatal("http should fail when protocol is https-only")
	}
}

// The map[string]string JWT shorthand is gone: a raw map would collide with
// any future map-shaped param, so JWTOpts is the only spelling.
func TestJWTMapShorthandRejected(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("map[string]string params should panic")
		}
	}()
	_ = String().JWT(map[string]string{"alg": "HS256"})
}

func TestJWTOptsStillWorks(t *testing.T) {
	schema := String().JWT(JWTOpts{Alg: "HS256"})
	if schema.SafeParse("not-a-jwt").Success {
		t.Fatal("invalid JWT should fail")
	}
}
