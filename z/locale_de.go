package z

import "fmt"

// German locale dictionaries — port of locales/de.ts.

var deSizable = map[string]localeSizing{
	"string": {"Zeichen", "zu haben"},
	"file":   {"Bytes", "zu haben"},
	"array":  {"Elemente", "zu haben"},
	"set":    {"Elemente", "zu haben"},
	"map":    {"Einträge", "zu haben"},
}

var deFormatDictionary = map[string]string{
	"regex":            "Eingabe",
	"email":            "E-Mail-Adresse",
	"url":              "URL",
	"emoji":            "Emoji",
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
	"datetime":         "ISO-Datum und -Uhrzeit",
	"date":             "ISO-Datum",
	"time":             "ISO-Uhrzeit",
	"duration":         "ISO-Dauer",
	"ipv4":             "IPv4-Adresse",
	"mac":              "MAC-Adresse",
	"ipv6":             "IPv6-Adresse",
	"cidrv4":           "IPv4-Bereich",
	"cidrv6":           "IPv6-Bereich",
	"base64":           "Base64-codierter String",
	"base64url":        "Base64-URL-codierter String",
	"json_string":      "JSON-String",
	"e164":             "E.164-Nummer",
	"jwt":              "JWT",
	"template_literal": "Eingabe",
}

var deTypeDictionary = map[string]string{
	"nan":    "NaN",
	"number": "Zahl",
	"array":  "Array",
}

// DeLocale is the German error map (locales/de.ts).
func DeLocale(issue *Issue) string {
	switch issue.Code {
	case IssueInvalidType:
		expected := issue.Expected
		if d, ok := deTypeDictionary[expected]; ok {
			expected = d
		}
		received := ParsedType(issue.Input)
		if d, ok := deTypeDictionary[received]; ok {
			received = d
		}
		if startsWithUpper(issue.Expected) {
			return fmt.Sprintf("Ungültige Eingabe: erwartet instanceof %s, erhalten %s", issue.Expected, received)
		}
		return fmt.Sprintf("Ungültige Eingabe: erwartet %s, erhalten %s", expected, received)

	case IssueInvalidValue:
		if len(issue.Values) == 1 {
			return "Ungültige Eingabe: erwartet " + StringifyPrimitive(issue.Values[0])
		}
		return "Ungültige Option: erwartet eine von " + JoinValues(issue.Values, "|")

	case IssueTooBig:
		adj := "<"
		if issue.Inclusive {
			adj = "<="
		}
		origin := issue.Origin
		if origin == "" {
			origin = "Wert"
		}
		if s, ok := deSizable[issue.Origin]; ok {
			unit := s.Unit
			if unit == "" {
				unit = "Elemente"
			}
			return fmt.Sprintf("Zu groß: erwartet, dass %s %s%s %s hat", origin, adj, FormatNumeric(issue.Maximum), unit)
		}
		return fmt.Sprintf("Zu groß: erwartet, dass %s %s%s ist", origin, adj, FormatNumeric(issue.Maximum))

	case IssueTooSmall:
		adj := ">"
		if issue.Inclusive {
			adj = ">="
		}
		if s, ok := deSizable[issue.Origin]; ok {
			return fmt.Sprintf("Zu klein: erwartet, dass %s %s%s %s hat",
				issue.Origin, adj, FormatNumeric(issue.Minimum), s.Unit)
		}
		return fmt.Sprintf("Zu klein: erwartet, dass %s %s%s ist", issue.Origin, adj, FormatNumeric(issue.Minimum))

	case IssueInvalidFormat:
		switch issue.Format {
		case "starts_with":
			return fmt.Sprintf("Ungültiger String: muss mit %q beginnen", issue.Prefix)
		case "ends_with":
			return fmt.Sprintf("Ungültiger String: muss mit %q enden", issue.Suffix)
		case "includes":
			return fmt.Sprintf("Ungültiger String: muss %q enthalten", issue.Includes)
		case "regex":
			return "Ungültiger String: muss dem Muster " + issue.Pattern + " entsprechen"
		}
		if d, ok := deFormatDictionary[issue.Format]; ok {
			return "Ungültig: " + d
		}
		return "Ungültig: " + issue.Format

	case IssueNotMultipleOf:
		return "Ungültige Zahl: muss ein Vielfaches von " + formatFloat(issue.Divisor) + " sein"

	case IssueUnrecognizedKeys:
		label := "Unbekannter Schlüssel"
		if len(issue.Keys) > 1 {
			label = "Unbekannte Schlüssel"
		}
		vals := make([]any, len(issue.Keys))
		for i, k := range issue.Keys {
			vals[i] = k
		}
		return label + ": " + JoinValues(vals, ", ")

	case IssueInvalidKey:
		return "Ungültiger Schlüssel in " + issue.Origin

	case IssueInvalidUnion:
		if len(issue.Values) > 0 {
			return "Ungültiger Diskriminatorwert. Erwartet " + JoinValues(issue.Values, " | ")
		}
		return "Ungültige Eingabe"

	case IssueInvalidElement:
		return "Ungültiger Wert in " + issue.Origin
	}
	return "Ungültige Eingabe"
}
