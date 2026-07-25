package zod

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
