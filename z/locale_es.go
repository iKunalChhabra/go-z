package z

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Spanish locale dictionaries — port of locales/es.ts.

var esSizable = map[string]localeSizing{
	"string": {"caracteres", "tener"},
	"file":   {"bytes", "tener"},
	"array":  {"elementos", "tener"},
	"set":    {"elementos", "tener"},
	"map":    {"entradas", "tener"},
}

var esFormatDictionary = map[string]string{
	"regex":            "entrada",
	"email":            "dirección de correo electrónico",
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
	"datetime":         "fecha y hora ISO",
	"date":             "fecha ISO",
	"time":             "hora ISO",
	"duration":         "duración ISO",
	"ipv4":             "dirección IPv4",
	"mac":              "dirección MAC",
	"ipv6":             "dirección IPv6",
	"cidrv4":           "rango IPv4",
	"cidrv6":           "rango IPv6",
	"base64":           "cadena codificada en base64",
	"base64url":        "URL codificada en base64",
	"json_string":      "cadena JSON",
	"e164":             "número E.164",
	"jwt":              "JWT",
	"template_literal": "entrada",
}

var esTypeDictionary = map[string]string{
	"nan":       "NaN",
	"string":    "texto",
	"number":    "número",
	"boolean":   "booleano",
	"array":     "arreglo",
	"object":    "objeto",
	"set":       "conjunto",
	"file":      "archivo",
	"date":      "fecha",
	"bigint":    "número grande",
	"symbol":    "símbolo",
	"undefined": "indefinido",
	"null":      "nulo",
	"function":  "función",
	"map":       "mapa",
	"record":    "registro",
	"tuple":     "tupla",
	"enum":      "enumeración",
	"union":     "unión",
	"literal":   "literal",
	"promise":   "promesa",
	"void":      "vacío",
	"never":     "nunca",
	"unknown":   "desconocido",
	"any":       "cualquiera",
}

// EsLocale is the Spanish error map (locales/es.ts).
func EsLocale(issue *Issue) string {
	switch issue.Code {
	case IssueInvalidType:
		expected := issue.Expected
		if d, ok := esTypeDictionary[expected]; ok {
			expected = d
		}
		received := ParsedType(issue.Input)
		if d, ok := esTypeDictionary[received]; ok {
			received = d
		}
		if startsWithUpper(issue.Expected) {
			return fmt.Sprintf("Entrada inválida: se esperaba instanceof %s, recibido %s", issue.Expected, received)
		}
		return fmt.Sprintf("Entrada inválida: se esperaba %s, recibido %s", expected, received)

	case IssueInvalidValue:
		if len(issue.Values) == 1 {
			return "Entrada inválida: se esperaba " + StringifyPrimitive(issue.Values[0])
		}
		return "Opción inválida: se esperaba una de " + JoinValues(issue.Values, "|")

	case IssueTooBig:
		adj := "<"
		if issue.Inclusive {
			adj = "<="
		}
		origin := issue.Origin
		if d, ok := esTypeDictionary[origin]; ok {
			origin = d
		}
		if origin == "" {
			origin = "valor"
		}
		if s, ok := esSizable[issue.Origin]; ok {
			unit := s.Unit
			if unit == "" {
				unit = "elementos"
			}
			return fmt.Sprintf("Demasiado grande: se esperaba que %s tuviera %s%s %s",
				origin, adj, FormatNumeric(issue.Maximum), unit)
		}
		return fmt.Sprintf("Demasiado grande: se esperaba que %s fuera %s%s", origin, adj, FormatNumeric(issue.Maximum))

	case IssueTooSmall:
		adj := ">"
		if issue.Inclusive {
			adj = ">="
		}
		origin := issue.Origin
		if d, ok := esTypeDictionary[origin]; ok {
			origin = d
		}
		if s, ok := esSizable[issue.Origin]; ok {
			return fmt.Sprintf("Demasiado pequeño: se esperaba que %s tuviera %s%s %s",
				origin, adj, FormatNumeric(issue.Minimum), s.Unit)
		}
		return fmt.Sprintf("Demasiado pequeño: se esperaba que %s fuera %s%s", origin, adj, FormatNumeric(issue.Minimum))

	case IssueInvalidFormat:
		switch issue.Format {
		case "starts_with":
			return fmt.Sprintf("Cadena inválida: debe comenzar con %q", issue.Prefix)
		case "ends_with":
			return fmt.Sprintf("Cadena inválida: debe terminar en %q", issue.Suffix)
		case "includes":
			return fmt.Sprintf("Cadena inválida: debe incluir %q", issue.Includes)
		case "regex":
			return "Cadena inválida: debe coincidir con el patrón " + issue.Pattern
		}
		if d, ok := esFormatDictionary[issue.Format]; ok {
			return "Inválido " + d
		}
		return "Inválido " + issue.Format

	case IssueNotMultipleOf:
		return "Número inválido: debe ser múltiplo de " + formatFloat(issue.Divisor)

	case IssueUnrecognizedKeys:
		plural := ""
		if len(issue.Keys) > 1 {
			plural = "s"
		}
		vals := make([]any, len(issue.Keys))
		for i, k := range issue.Keys {
			vals[i] = k
		}
		return fmt.Sprintf("Llave%s desconocida%s: %s", plural, plural, JoinValues(vals, ", "))

	case IssueInvalidKey:
		origin := issue.Origin
		if d, ok := esTypeDictionary[origin]; ok {
			origin = d
		}
		return "Llave inválida en " + origin

	case IssueInvalidUnion:
		return "Entrada inválida"

	case IssueInvalidElement:
		origin := issue.Origin
		if d, ok := esTypeDictionary[origin]; ok {
			origin = d
		}
		return "Valor inválido en " + origin
	}
	return "Entrada inválida"
}

// Locale returns the ErrorMap for a language name ("en","es","fr","de","ja","pt","zh").
// Unknown names fall back to EnLocale.
func Locale(name string) ErrorMap {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "en", "eng", "english":
		return EnLocale
	case "es", "spa", "spanish", "español", "espanol":
		return EsLocale
	case "fr", "fra", "french", "français", "francais":
		return FrLocale
	case "de", "deu", "ger", "german", "deutsch":
		return DeLocale
	case "ja", "jpn", "japanese":
		return JaLocale
	case "pt", "por", "portuguese", "português", "portugues":
		return PtLocale
	case "zh", "zh-cn", "zh_cn", "zh-hans", "chinese", "cn":
		return ZhLocale
	default:
		return EnLocale
	}
}

func startsWithUpper(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}
