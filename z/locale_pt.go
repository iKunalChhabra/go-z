package z

import "fmt"

// Portuguese locale dictionaries — port of locales/pt.ts.

var ptSizable = map[string]localeSizing{
	"string": {"caracteres", "ter"},
	"file":   {"bytes", "ter"},
	"array":  {"itens", "ter"},
	"set":    {"itens", "ter"},
	"map":    {"entradas", "ter"},
}

var ptFormatDictionary = map[string]string{
	"regex":            "padrão",
	"email":            "endereço de e-mail",
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
	"datetime":         "data e hora ISO",
	"date":             "data ISO",
	"time":             "hora ISO",
	"duration":         "duração ISO",
	"ipv4":             "endereço IPv4",
	"mac":              "endereço MAC",
	"ipv6":             "endereço IPv6",
	"cidrv4":           "faixa de IPv4",
	"cidrv6":           "faixa de IPv6",
	"base64":           "texto codificado em base64",
	"base64url":        "URL codificada em base64",
	"json_string":      "texto JSON",
	"e164":             "número E.164",
	"jwt":              "JWT",
	"template_literal": "entrada",
}

var ptTypeDictionary = map[string]string{
	"nan":    "NaN",
	"number": "número",
	"null":   "nulo",
}

// PtLocale is the Portuguese error map (locales/pt.ts).
func PtLocale(issue *Issue) string {
	switch issue.Code {
	case IssueInvalidType:
		expected := issue.Expected
		if d, ok := ptTypeDictionary[expected]; ok {
			expected = d
		}
		received := ParsedType(issue.Input)
		if d, ok := ptTypeDictionary[received]; ok {
			received = d
		}
		if startsWithUpper(issue.Expected) {
			return fmt.Sprintf("Tipo inválido: esperado instanceof %s, recebido %s", issue.Expected, received)
		}
		return fmt.Sprintf("Tipo inválido: esperado %s, recebido %s", expected, received)

	case IssueInvalidValue:
		if len(issue.Values) == 1 {
			return "Entrada inválida: esperado " + StringifyPrimitive(issue.Values[0])
		}
		return "Opção inválida: esperada uma das " + JoinValues(issue.Values, "|")

	case IssueTooBig:
		adj := "<"
		if issue.Inclusive {
			adj = "<="
		}
		origin := issue.Origin
		if origin == "" {
			origin = "valor"
		}
		if s, ok := ptSizable[issue.Origin]; ok {
			unit := s.Unit
			if unit == "" {
				unit = "elementos"
			}
			return fmt.Sprintf("Muito grande: esperado que %s tivesse %s%s %s", origin, adj, FormatNumeric(issue.Maximum), unit)
		}
		return fmt.Sprintf("Muito grande: esperado que %s fosse %s%s", origin, adj, FormatNumeric(issue.Maximum))

	case IssueTooSmall:
		adj := ">"
		if issue.Inclusive {
			adj = ">="
		}
		if s, ok := ptSizable[issue.Origin]; ok {
			return fmt.Sprintf("Muito pequeno: esperado que %s tivesse %s%s %s",
				issue.Origin, adj, FormatNumeric(issue.Minimum), s.Unit)
		}
		return fmt.Sprintf("Muito pequeno: esperado que %s fosse %s%s", issue.Origin, adj, FormatNumeric(issue.Minimum))

	case IssueInvalidFormat:
		switch issue.Format {
		case "starts_with":
			return fmt.Sprintf("Texto inválido: deve começar com %q", issue.Prefix)
		case "ends_with":
			return fmt.Sprintf("Texto inválido: deve terminar com %q", issue.Suffix)
		case "includes":
			return fmt.Sprintf("Texto inválido: deve incluir %q", issue.Includes)
		case "regex":
			return "Texto inválido: deve corresponder ao padrão " + issue.Pattern
		}
		if d, ok := ptFormatDictionary[issue.Format]; ok {
			return d + " inválido"
		}
		return issue.Format + " inválido"

	case IssueNotMultipleOf:
		return "Número inválido: deve ser múltiplo de " + formatFloat(issue.Divisor)

	case IssueUnrecognizedKeys:
		plural := ""
		if len(issue.Keys) > 1 {
			plural = "s"
		}
		vals := make([]any, len(issue.Keys))
		for i, k := range issue.Keys {
			vals[i] = k
		}
		return fmt.Sprintf("Chave%s desconhecida%s: %s", plural, plural, JoinValues(vals, ", "))

	case IssueInvalidKey:
		return "Chave inválida em " + issue.Origin

	case IssueInvalidUnion:
		if len(issue.Values) > 0 {
			return "Valor de discriminador inválido. Esperado " + JoinValues(issue.Values, " | ")
		}
		return "Entrada inválida"

	case IssueInvalidElement:
		return "Valor inválido em " + issue.Origin
	}
	return "Campo inválido"
}
