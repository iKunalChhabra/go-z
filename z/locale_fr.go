package z

import "fmt"

// French locale dictionaries — port of locales/fr.ts.

var frSizable = map[string]localeSizing{
	"string": {"caractères", "avoir"},
	"file":   {"octets", "avoir"},
	"array":  {"éléments", "avoir"},
	"set":    {"éléments", "avoir"},
	"map":    {"entrées", "avoir"},
}

var frFormatDictionary = map[string]string{
	"regex":            "entrée",
	"email":            "adresse e-mail",
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
	"datetime":         "date et heure ISO",
	"date":             "date ISO",
	"time":             "heure ISO",
	"duration":         "durée ISO",
	"ipv4":             "adresse IPv4",
	"mac":              "adresse MAC",
	"ipv6":             "adresse IPv6",
	"cidrv4":           "plage IPv4",
	"cidrv6":           "plage IPv6",
	"base64":           "chaîne encodée en base64",
	"base64url":        "chaîne encodée en base64url",
	"json_string":      "chaîne JSON",
	"e164":             "numéro E.164",
	"jwt":              "JWT",
	"template_literal": "entrée",
}

var frTypeDictionary = map[string]string{
	"string":      "chaîne",
	"number":      "nombre",
	"int":         "entier",
	"boolean":     "booléen",
	"bigint":      "grand entier",
	"symbol":      "symbole",
	"undefined":   "indéfini",
	"null":        "null",
	"never":       "jamais",
	"void":        "vide",
	"date":        "date",
	"array":       "tableau",
	"object":      "objet",
	"tuple":       "tuple",
	"record":      "enregistrement",
	"map":         "carte",
	"set":         "ensemble",
	"file":        "fichier",
	"nonoptional": "non-optionnel",
	"nan":         "NaN",
	"function":    "fonction",
}

// FrLocale is the French error map (locales/fr.ts).
func FrLocale(issue *Issue) string {
	switch issue.Code {
	case IssueInvalidType:
		expected := issue.Expected
		if d, ok := frTypeDictionary[expected]; ok {
			expected = d
		}
		received := ParsedType(issue.Input)
		if d, ok := frTypeDictionary[received]; ok {
			received = d
		}
		if startsWithUpper(issue.Expected) {
			return fmt.Sprintf("Entrée invalide : instanceof %s attendu, %s reçu", issue.Expected, received)
		}
		return fmt.Sprintf("Entrée invalide : %s attendu, %s reçu", expected, received)

	case IssueInvalidValue:
		if len(issue.Values) == 1 {
			return "Entrée invalide : " + StringifyPrimitive(issue.Values[0]) + " attendu"
		}
		return "Option invalide : une valeur parmi " + JoinValues(issue.Values, "|") + " attendue"

	case IssueTooBig:
		adj := "<"
		if issue.Inclusive {
			adj = "<="
		}
		origin := "valeur"
		if d, ok := frTypeDictionary[issue.Origin]; ok {
			origin = d
		} else if issue.Origin != "" {
			origin = issue.Origin
		}
		if s, ok := frSizable[issue.Origin]; ok {
			unit := s.Unit
			if unit == "" {
				unit = "élément(s)"
			}
			return fmt.Sprintf("Trop grand : %s doit %s %s%s %s", origin, s.Verb, adj, FormatNumeric(issue.Maximum), unit)
		}
		return fmt.Sprintf("Trop grand : %s doit être %s%s", origin, adj, FormatNumeric(issue.Maximum))

	case IssueTooSmall:
		adj := ">"
		if issue.Inclusive {
			adj = ">="
		}
		origin := "valeur"
		if d, ok := frTypeDictionary[issue.Origin]; ok {
			origin = d
		} else if issue.Origin != "" {
			origin = issue.Origin
		}
		if s, ok := frSizable[issue.Origin]; ok {
			return fmt.Sprintf("Trop petit : %s doit %s %s%s %s", origin, s.Verb, adj, FormatNumeric(issue.Minimum), s.Unit)
		}
		return fmt.Sprintf("Trop petit : %s doit être %s%s", origin, adj, FormatNumeric(issue.Minimum))

	case IssueInvalidFormat:
		switch issue.Format {
		case "starts_with":
			return fmt.Sprintf("Chaîne invalide : doit commencer par %q", issue.Prefix)
		case "ends_with":
			return fmt.Sprintf("Chaîne invalide : doit se terminer par %q", issue.Suffix)
		case "includes":
			return fmt.Sprintf("Chaîne invalide : doit inclure %q", issue.Includes)
		case "regex":
			return "Chaîne invalide : doit correspondre au modèle " + issue.Pattern
		}
		if d, ok := frFormatDictionary[issue.Format]; ok {
			return d + " invalide"
		}
		return issue.Format + " invalide"

	case IssueNotMultipleOf:
		return "Nombre invalide : doit être un multiple de " + formatFloat(issue.Divisor)

	case IssueUnrecognizedKeys:
		plural := ""
		if len(issue.Keys) > 1 {
			plural = "s"
		}
		vals := make([]any, len(issue.Keys))
		for i, k := range issue.Keys {
			vals[i] = k
		}
		return fmt.Sprintf("Clé%s non reconnue%s : %s", plural, plural, JoinValues(vals, ", "))

	case IssueInvalidKey:
		return "Clé invalide dans " + issue.Origin

	case IssueInvalidUnion:
		if len(issue.Values) > 0 {
			return "Valeur de discriminant invalide. Attendu " + JoinValues(issue.Values, " | ")
		}
		return "Entrée invalide"

	case IssueInvalidElement:
		return "Valeur invalide dans " + issue.Origin
	}
	return "Entrée invalide"
}
