package z

import "fmt"

// Simplified Chinese locale dictionaries — port of locales/zh-CN.ts.

var zhSizable = map[string]localeSizing{
	"string": {"字符", "包含"},
	"file":   {"字节", "包含"},
	"array":  {"项", "包含"},
	"set":    {"项", "包含"},
}

var zhFormatDictionary = map[string]string{
	"regex":            "输入",
	"email":            "电子邮件",
	"url":              "URL",
	"emoji":            "表情符号",
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
	"datetime":         "ISO日期时间",
	"date":             "ISO日期",
	"time":             "ISO时间",
	"duration":         "ISO时长",
	"ipv4":             "IPv4地址",
	"ipv6":             "IPv6地址",
	"cidrv4":           "IPv4网段",
	"cidrv6":           "IPv6网段",
	"base64":           "base64编码字符串",
	"base64url":        "base64url编码字符串",
	"json_string":      "JSON字符串",
	"e164":             "E.164号码",
	"jwt":              "JWT",
	"template_literal": "输入",
}

var zhTypeDictionary = map[string]string{
	"nan":    "NaN",
	"number": "数字",
	"array":  "数组",
	"null":   "空值(null)",
}

// ZhLocale is the Simplified Chinese error map (locales/zh-CN.ts).
func ZhLocale(issue *Issue) string {
	switch issue.Code {
	case IssueInvalidType:
		expected := issue.Expected
		if d, ok := zhTypeDictionary[expected]; ok {
			expected = d
		}
		received := ParsedType(issue.Input)
		if d, ok := zhTypeDictionary[received]; ok {
			received = d
		}
		if startsWithUpper(issue.Expected) {
			return fmt.Sprintf("无效输入：期望 instanceof %s，实际接收 %s", issue.Expected, received)
		}
		return fmt.Sprintf("无效输入：期望 %s，实际接收 %s", expected, received)

	case IssueInvalidValue:
		if len(issue.Values) == 1 {
			return "无效输入：期望 " + StringifyPrimitive(issue.Values[0])
		}
		return "无效选项：期望以下之一 " + JoinValues(issue.Values, "|")

	case IssueTooBig:
		adj := "<"
		if issue.Inclusive {
			adj = "<="
		}
		origin := issue.Origin
		if origin == "" {
			origin = "值"
		}
		if s, ok := zhSizable[issue.Origin]; ok {
			unit := s.Unit
			if unit == "" {
				unit = "个元素"
			}
			return fmt.Sprintf("数值过大：期望 %s %s%s %s", origin, adj, FormatNumeric(issue.Maximum), unit)
		}
		return fmt.Sprintf("数值过大：期望 %s %s%s", origin, adj, FormatNumeric(issue.Maximum))

	case IssueTooSmall:
		adj := ">"
		if issue.Inclusive {
			adj = ">="
		}
		if s, ok := zhSizable[issue.Origin]; ok {
			return fmt.Sprintf("数值过小：期望 %s %s%s %s",
				issue.Origin, adj, FormatNumeric(issue.Minimum), s.Unit)
		}
		return fmt.Sprintf("数值过小：期望 %s %s%s", issue.Origin, adj, FormatNumeric(issue.Minimum))

	case IssueInvalidFormat:
		switch issue.Format {
		case "starts_with":
			return fmt.Sprintf("无效字符串：必须以 %q 开头", issue.Prefix)
		case "ends_with":
			return fmt.Sprintf("无效字符串：必须以 %q 结尾", issue.Suffix)
		case "includes":
			return fmt.Sprintf("无效字符串：必须包含 %q", issue.Includes)
		case "regex":
			return "无效字符串：必须满足正则表达式 " + issue.Pattern
		}
		if d, ok := zhFormatDictionary[issue.Format]; ok {
			return "无效" + d
		}
		return "无效" + issue.Format

	case IssueNotMultipleOf:
		return "无效数字：必须是 " + formatFloat(issue.Divisor) + " 的倍数"

	case IssueUnrecognizedKeys:
		vals := make([]any, len(issue.Keys))
		for i, k := range issue.Keys {
			vals[i] = k
		}
		return "出现未知的键(key): " + JoinValues(vals, ", ")

	case IssueInvalidKey:
		return issue.Origin + " 中的键(key)无效"

	case IssueInvalidUnion:
		return "无效输入"

	case IssueInvalidElement:
		return issue.Origin + " 中包含无效值(value)"
	}
	return "无效输入"
}
