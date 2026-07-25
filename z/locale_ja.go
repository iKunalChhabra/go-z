package z

import "fmt"

// Japanese locale dictionaries — port of locales/ja.ts.

var jaSizable = map[string]localeSizing{
	"string": {"文字", "である"},
	"file":   {"バイト", "である"},
	"array":  {"要素", "である"},
	"set":    {"要素", "である"},
	"map":    {"要素", "である"},
}

var jaFormatDictionary = map[string]string{
	"regex":            "入力値",
	"email":            "メールアドレス",
	"url":              "URL",
	"emoji":            "絵文字",
	"uuid":             "UUID",
	"uuidv4":           "UUIDv4",
	"uuidv6":           "UUIDv6",
	"nanoid":           "nanoid",
	"guid":             "GUID",
	"cuid":             "cuid",
	"cuid2":            "cuid2",
	"ulid":             "ULID",
	"xid":              "XID",
	"ksuid":            "KSUID",
	"datetime":         "ISO日時",
	"date":             "ISO日付",
	"time":             "ISO時刻",
	"duration":         "ISO期間",
	"ipv4":             "IPv4アドレス",
	"mac":              "MACアドレス",
	"ipv6":             "IPv6アドレス",
	"cidrv4":           "IPv4範囲",
	"cidrv6":           "IPv6範囲",
	"base64":           "base64エンコード文字列",
	"base64url":        "base64urlエンコード文字列",
	"json_string":      "JSON文字列",
	"e164":             "E.164番号",
	"jwt":              "JWT",
	"template_literal": "入力値",
}

var jaTypeDictionary = map[string]string{
	"nan":    "NaN",
	"number": "数値",
	"array":  "配列",
}

// JaLocale is the Japanese error map (locales/ja.ts).
func JaLocale(issue *Issue) string {
	switch issue.Code {
	case IssueInvalidType:
		expected := issue.Expected
		if d, ok := jaTypeDictionary[expected]; ok {
			expected = d
		}
		received := ParsedType(issue.Input)
		if d, ok := jaTypeDictionary[received]; ok {
			received = d
		}
		if startsWithUpper(issue.Expected) {
			return fmt.Sprintf("無効な入力: instanceof %sが期待されましたが、%sが入力されました", issue.Expected, received)
		}
		return fmt.Sprintf("無効な入力: %sが期待されましたが、%sが入力されました", expected, received)

	case IssueInvalidValue:
		if len(issue.Values) == 1 {
			return "無効な入力: " + StringifyPrimitive(issue.Values[0]) + "が期待されました"
		}
		return "無効な選択: " + JoinValues(issue.Values, "、") + "のいずれかである必要があります"

	case IssueTooBig:
		adj := "より小さい"
		if issue.Inclusive {
			adj = "以下である"
		}
		origin := issue.Origin
		if origin == "" {
			origin = "値"
		}
		if s, ok := jaSizable[issue.Origin]; ok {
			unit := s.Unit
			if unit == "" {
				unit = "要素"
			}
			return fmt.Sprintf("大きすぎる値: %sは%s%s%s必要があります", origin, FormatNumeric(issue.Maximum), unit, adj)
		}
		return fmt.Sprintf("大きすぎる値: %sは%s%s必要があります", origin, FormatNumeric(issue.Maximum), adj)

	case IssueTooSmall:
		adj := "より大きい"
		if issue.Inclusive {
			adj = "以上である"
		}
		if s, ok := jaSizable[issue.Origin]; ok {
			return fmt.Sprintf("小さすぎる値: %sは%s%s%s必要があります",
				issue.Origin, FormatNumeric(issue.Minimum), s.Unit, adj)
		}
		return fmt.Sprintf("小さすぎる値: %sは%s%s必要があります", issue.Origin, FormatNumeric(issue.Minimum), adj)

	case IssueInvalidFormat:
		switch issue.Format {
		case "starts_with":
			return fmt.Sprintf("無効な文字列: %qで始まる必要があります", issue.Prefix)
		case "ends_with":
			return fmt.Sprintf("無効な文字列: %qで終わる必要があります", issue.Suffix)
		case "includes":
			return fmt.Sprintf("無効な文字列: %qを含む必要があります", issue.Includes)
		case "regex":
			return "無効な文字列: パターン" + issue.Pattern + "に一致する必要があります"
		}
		if d, ok := jaFormatDictionary[issue.Format]; ok {
			return "無効な" + d
		}
		return "無効な" + issue.Format

	case IssueNotMultipleOf:
		return "無効な数値: " + formatFloat(issue.Divisor) + "の倍数である必要があります"

	case IssueUnrecognizedKeys:
		suffix := ""
		if len(issue.Keys) > 1 {
			suffix = "群"
		}
		vals := make([]any, len(issue.Keys))
		for i, k := range issue.Keys {
			vals[i] = k
		}
		return "認識されていないキー" + suffix + ": " + JoinValues(vals, "、")

	case IssueInvalidKey:
		return issue.Origin + "内の無効なキー"

	case IssueInvalidUnion:
		if len(issue.Values) > 0 {
			return "無効な識別子の値です。期待される値: " + JoinValues(issue.Values, " | ")
		}
		return "無効な入力"

	case IssueInvalidElement:
		return issue.Origin + "内の無効な値"
	}
	return "無効な入力"
}
