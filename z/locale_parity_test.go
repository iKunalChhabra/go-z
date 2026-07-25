package z

import (
	"strings"
	"testing"
)

// localeFns is every shipped locale, keyed by the tag Locale() accepts.
var localeFns = map[string]func(*Issue) string{
	"en": EnLocale,
	"es": EsLocale,
	"fr": FrLocale,
	"de": DeLocale,
	"ja": JaLocale,
	"pt": PtLocale,
	"zh": ZhLocale,
}

// Every locale must know every format name and every sizable origin English
// knows. A missing entry is not a compile error — the renderer falls back to the
// raw key — so "mac" printed as the literal string "mac" in six languages until
// this test existed.
func TestLocaleDictionariesCoverEnglish(t *testing.T) {
	formats := map[string]map[string]string{
		"es": esFormatDictionary, "fr": frFormatDictionary, "de": deFormatDictionary,
		"ja": jaFormatDictionary, "pt": ptFormatDictionary, "zh": zhFormatDictionary,
	}
	for lang, dict := range formats {
		for key := range enFormatDictionary {
			if _, ok := dict[key]; !ok {
				t.Errorf("locale %s is missing the %q format name", lang, key)
			}
		}
	}

	sizable := map[string]map[string]localeSizing{
		"es": esSizable, "fr": frSizable, "de": deSizable,
		"ja": jaSizable, "pt": ptSizable, "zh": zhSizable,
	}
	for lang, dict := range sizable {
		for origin := range enSizable {
			if _, ok := dict[origin]; !ok {
				t.Errorf("locale %s is missing the %q size unit", lang, origin)
			}
		}
	}
}

// Every issue code has to render in every locale. Two of the eleven used to be
// smoke-tested, so a code with no case in a locale's switch would have shipped
// producing an empty message.
func TestLocalesRenderEveryIssueCode(t *testing.T) {
	samples := map[IssueCode]Issue{
		IssueInvalidType:      {Code: IssueInvalidType, Expected: "string", Input: 1},
		IssueInvalidValue:     {Code: IssueInvalidValue, Values: []any{"a"}},
		IssueTooBig:           {Code: IssueTooBig, Origin: "string", Maximum: float64(5), Inclusive: true},
		IssueTooSmall:         {Code: IssueTooSmall, Origin: "array", Minimum: float64(2)},
		IssueInvalidFormat:    {Code: IssueInvalidFormat, Format: "email", Input: "x"},
		IssueNotMultipleOf:    {Code: IssueNotMultipleOf, Divisor: 3},
		IssueUnrecognizedKeys: {Code: IssueUnrecognizedKeys, Keys: []string{"a", "b"}},
		IssueInvalidKey:       {Code: IssueInvalidKey, Origin: "record"},
		IssueInvalidUnion:     {Code: IssueInvalidUnion},
		IssueInvalidElement:   {Code: IssueInvalidElement, Origin: "set"},
		IssueCustom:           {Code: IssueCustom},
	}
	if len(samples) != len(AllIssueCodes) {
		t.Fatalf("this test covers %d codes but the package defines %d", len(samples), len(AllIssueCodes))
	}

	for _, code := range AllIssueCodes {
		sample, ok := samples[code]
		if !ok {
			t.Fatalf("no sample issue for %s", code)
		}
		for lang, render := range localeFns {
			issue := sample
			msg := render(&issue)
			if strings.TrimSpace(msg) == "" {
				t.Errorf("locale %s renders %s as an empty message", lang, code)
				continue
			}
			// A raw issue code or Go format verb in the output means a missing
			// case or a broken template.
			if strings.Contains(msg, "%!") || strings.Contains(msg, string(code)) {
				t.Errorf("locale %s renders %s as %q", lang, code, msg)
			}
		}
	}
}

// Every locale actually translates: a locale that fell through to English for a
// code would silently look fine in the test above.
func TestLocalesDifferFromEnglish(t *testing.T) {
	sample := Issue{Code: IssueTooSmall, Origin: "string", Minimum: float64(3), Inclusive: true}
	english := EnLocale(&sample)
	for lang, render := range localeFns {
		if lang == "en" {
			continue
		}
		issue := sample
		if got := render(&issue); got == english {
			t.Errorf("locale %s returned the English message %q", lang, got)
		}
	}
}
