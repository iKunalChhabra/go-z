package z

import (
	"strings"
	"testing"
)

// Smoke tests: non-English locales return localized text for invalid_type and too_small string.

func TestLocalesInvalidTypeAndTooSmall(t *testing.T) {
	typeCase := &Issue{
		Code:     IssueInvalidType,
		Expected: "string",
		Input:    123,
	}
	smallCase := &Issue{
		Code:      IssueTooSmall,
		Origin:    "string",
		Minimum:   5,
		Inclusive: true,
	}

	cases := []struct {
		name     string
		locale   ErrorMap
		typeSub  string
		smallSub string
		notEN    string // English fragment that must NOT appear
	}{
		{"es", EsLocale, "se esperaba", "Demasiado pequeño", "Invalid input"},
		{"fr", FrLocale, "attendu", "Trop petit", "Invalid input"},
		{"de", DeLocale, "erwartet", "Zu klein", "Invalid input"},
		{"ja", JaLocale, "期待されました", "小さすぎる値", "Invalid input"},
		{"pt", PtLocale, "esperado", "Muito pequeno", "Invalid input"},
		{"zh", ZhLocale, "期望", "数值过小", "Invalid input"},
	}

	enType := EnLocale(typeCase)
	enSmall := EnLocale(smallCase)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotType := tc.locale(typeCase)
			if gotType == "" || gotType == enType {
				t.Fatalf("invalid_type still English/empty: %q", gotType)
			}
			if !strings.Contains(gotType, tc.typeSub) {
				t.Fatalf("invalid_type missing %q: %q", tc.typeSub, gotType)
			}
			if strings.Contains(gotType, tc.notEN) {
				t.Fatalf("invalid_type contains English: %q", gotType)
			}

			gotSmall := tc.locale(smallCase)
			if gotSmall == "" || gotSmall == enSmall {
				t.Fatalf("too_small still English/empty: %q", gotSmall)
			}
			if !strings.Contains(gotSmall, tc.smallSub) {
				t.Fatalf("too_small missing %q: %q", tc.smallSub, gotSmall)
			}
		})
	}
}

// Regression: es/pt translated the base64url format as a base64-encoded URL,
// which describes a different thing. The message must name base64url.
func TestLocaleBase64URLNamesBase64URL(t *testing.T) {
	issue := &Issue{Code: IssueInvalidFormat, Format: "base64url", Input: "x"}
	cases := []struct {
		name   string
		locale ErrorMap
		not    string
	}{
		{"es", EsLocale, "URL codificada en base64"},
		{"pt", PtLocale, "URL codificada em base64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := tc.locale(issue)
			if !strings.Contains(msg, "base64url") {
				t.Fatalf("base64url message = %q, want it to mention base64url", msg)
			}
			if strings.Contains(msg, tc.not) {
				t.Fatalf("base64url message = %q, still says %q", msg, tc.not)
			}
		})
	}
}

// Regression: an empty Origin in IssueTooSmall produced a double space
// ("erwartet, dass  >5 ist") where IssueTooBig already fell back to a generic
// noun. TooSmall now uses the same fallback.
func TestLocaleTooSmallEmptyOriginFallback(t *testing.T) {
	issue := &Issue{Code: IssueTooSmall, Origin: "", Minimum: float64(5)}
	locales := map[string]ErrorMap{
		"es": EsLocale, "de": DeLocale, "pt": PtLocale, "zh": ZhLocale,
	}
	for name, locale := range locales {
		msg := locale(issue)
		if strings.Contains(msg, "  ") {
			t.Errorf("locale %s renders an empty origin as a double space: %q", name, msg)
		}
	}
}

// Regression: the pt default message said "Campo inválido" while every other
// pt fallback says "Entrada inválida".
func TestPtDefaultFallbackIsEntradaInvalida(t *testing.T) {
	if got := PtLocale(&Issue{Code: IssueCustom}); got != "Entrada inválida" {
		t.Fatalf("pt default = %q, want %q", got, "Entrada inválida")
	}
}

func TestLocaleLookup(t *testing.T) {
	iss := &Issue{Code: IssueInvalidType, Expected: "string", Input: nil}
	if Locale("es")(iss) != EsLocale(iss) {
		t.Fatal("Locale(es)")
	}
	if Locale("zh-CN")(iss) != ZhLocale(iss) {
		t.Fatal("Locale(zh-CN)")
	}
	if Locale("unknown")(iss) != EnLocale(iss) {
		t.Fatal("unknown should fall back to en")
	}
}

func TestLocaleViaConfigure(t *testing.T) {
	prev := Configure(Config{LocaleError: EsLocale})
	t.Cleanup(func() { Configure(prev) })

	iss := Issue{Code: IssueInvalidType, Expected: "string", Input: 1}
	final := FinalizeIssue(iss, nil, GetConfig())
	if !strings.Contains(final.Message, "se esperaba") {
		t.Fatalf("Configure locale: %q", final.Message)
	}
}
