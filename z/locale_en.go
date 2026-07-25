package z

import "fmt"

// sizing describes how a sized origin is phrased ("string to have N
// characters"). Ports the Sizable table in locales/en.ts.
type localeSizing struct{ Unit, Verb string }

var enSizable = map[string]localeSizing{
	"string": {"characters", "to have"},
	"file":   {"bytes", "to have"},
	"array":  {"items", "to have"},
	"set":    {"items", "to have"},
	"map":    {"entries", "to have"},
}

// enFormatDictionary ports the FormatDictionary in locales/en.ts.
var enFormatDictionary = map[string]string{
	"regex":            "input",
	"email":            "email address",
	"url":              "URL",
	"emoji":            "emoji",
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
	"datetime":         "ISO datetime",
	"date":             "ISO date",
	"time":             "ISO time",
	"duration":         "ISO duration",
	"ipv4":             "IPv4 address",
	"ipv6":             "IPv6 address",
	"mac":              "MAC address",
	"cidrv4":           "IPv4 range",
	"cidrv6":           "IPv6 range",
	"base64":           "base64-encoded string",
	"base64url":        "base64url-encoded string",
	"json_string":      "JSON string",
	"e164":             "E.164 number",
	"jwt":              "JWT",
	"template_literal": "input",
}

// enTypeDictionary ports TypeDictionary (display names for parsed types).
var enTypeDictionary = map[string]string{
	"nan": "NaN",
}

// EnLocale is the English error map, a direct port of locales/en.ts.
func EnLocale(issue *Issue) string {
	switch issue.Code {
	case IssueInvalidType:
		expected := issue.Expected
		if d, ok := enTypeDictionary[expected]; ok {
			expected = d
		}
		received := ParsedType(issue.Input)
		if d, ok := enTypeDictionary[received]; ok {
			received = d
		}
		return fmt.Sprintf("Invalid input: expected %s, received %s", expected, received)

	case IssueInvalidValue:
		if len(issue.Values) == 1 {
			return "Invalid input: expected " + StringifyPrimitive(issue.Values[0])
		}
		return "Invalid option: expected one of " + JoinValues(issue.Values, "|")

	case IssueTooBig:
		adj := "<"
		if issue.Inclusive {
			adj = "<="
		}
		origin := issue.Origin
		if origin == "" {
			origin = "value"
		}
		if s, ok := enSizable[issue.Origin]; ok {
			return fmt.Sprintf("Too big: expected %s %s %s%s %s",
				origin, s.Verb, adj, FormatNumeric(issue.Maximum), s.Unit)
		}
		return fmt.Sprintf("Too big: expected %s to be %s%s", origin, adj, FormatNumeric(issue.Maximum))

	case IssueTooSmall:
		adj := ">"
		if issue.Inclusive {
			adj = ">="
		}
		origin := issue.Origin
		if origin == "" {
			origin = "value"
		}
		if s, ok := enSizable[issue.Origin]; ok {
			return fmt.Sprintf("Too small: expected %s %s %s%s %s",
				origin, s.Verb, adj, FormatNumeric(issue.Minimum), s.Unit)
		}
		return fmt.Sprintf("Too small: expected %s to be %s%s", origin, adj, FormatNumeric(issue.Minimum))

	case IssueInvalidFormat:
		switch issue.Format {
		case "starts_with":
			return fmt.Sprintf("Invalid string: must start with %q", issue.Prefix)
		case "ends_with":
			return fmt.Sprintf("Invalid string: must end with %q", issue.Suffix)
		case "includes":
			return fmt.Sprintf("Invalid string: must include %q", issue.Includes)
		case "regex":
			return "Invalid string: must match pattern " + issue.Pattern
		}
		if d, ok := enFormatDictionary[issue.Format]; ok {
			return "Invalid " + d
		}
		return "Invalid " + issue.Format

	case IssueNotMultipleOf:
		return "Invalid number: must be a multiple of " + formatFloat(issue.Divisor)

	case IssueUnrecognizedKeys:
		plural := ""
		if len(issue.Keys) > 1 {
			plural = "s"
		}
		vals := make([]any, len(issue.Keys))
		for i, k := range issue.Keys {
			vals[i] = k
		}
		return fmt.Sprintf("Unrecognized key%s: %s", plural, JoinValues(vals, ", "))

	case IssueInvalidKey:
		return "Invalid key in " + issue.Origin

	case IssueInvalidUnion:
		if len(issue.Values) > 0 {
			parts := ""
			for i, v := range issue.Values {
				if i > 0 {
					parts += " | "
				}
				parts += fmt.Sprintf("'%v'", v)
			}
			return "Invalid discriminator value. Expected " + parts
		}
		return "Invalid input"

	case IssueInvalidElement:
		return "Invalid value in " + issue.Origin
	}
	return "Invalid input"
}
