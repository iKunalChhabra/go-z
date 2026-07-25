# Issue codes

go-z uses **11** issue codes. JSON field names are stable across releases.

```go
const (
    IssueInvalidType      IssueCode = "invalid_type"
    IssueTooBig           IssueCode = "too_big"
    IssueTooSmall         IssueCode = "too_small"
    IssueInvalidFormat    IssueCode = "invalid_format"
    IssueNotMultipleOf    IssueCode = "not_multiple_of"
    IssueUnrecognizedKeys IssueCode = "unrecognized_keys"
    IssueInvalidUnion     IssueCode = "invalid_union"
    IssueInvalidKey       IssueCode = "invalid_key"
    IssueInvalidElement   IssueCode = "invalid_element"
    IssueInvalidValue     IssueCode = "invalid_value"
    IssueCustom           IssueCode = "custom"
)
```

Every issue has: `code`, `path` (`[]any`), `message` (after finalize). `input` is omitted unless `ParseCtx.ReportInput` is true.

---

## 1. `invalid_type`

Wrong runtime type (or `NonOptional` after Missing).

| Field | Type | When |
|---|---|---|
| `expected` | string | `"string"`, `"number"`, `"object"`, `"nonoptional"`, … |

**EnLocale:** `Invalid input: expected {expected}, received {received}`

```text
Invalid input: expected string, received number
```

---

## 2. `too_big`

Upper bound failed (string length, number max, array size, …).

| Field | Type | When |
|---|---|---|
| `origin` | string | `"string"` `"number"` `"array"` `"set"` `"map"` `"date"` … |
| `maximum` | any | bound |
| `inclusive` | bool | `<=` vs `<` |
| `exact` | bool | exact length/size mode |

**EnLocale (sizable):** `Too big: expected {origin} to have <={n} {unit}`

```text
Too big: expected string to have <=100 characters
Too big: expected number to be <=150
```

---

## 3. `too_small`

Lower bound failed.

| Field | Type | When |
|---|---|---|
| `origin` | string | same as too_big |
| `minimum` | any | bound |
| `inclusive` | bool | `>=` vs `>` |
| `exact` | bool | exact mode |

**EnLocale:** `Too small: expected {origin} to have >={n} {unit}`

```text
Too small: expected string to have >=2 characters
Too small: expected array to have >=1 items
```

---

## 4. `invalid_format`

String format / pattern check failed.

| Field | Type | When |
|---|---|---|
| `format` | string | `"email"`, `"uuid"`, `"regex"`, `"starts_with"`, … |
| `pattern` | string | regex formats |
| `prefix` / `suffix` / `includes` | string | string affinity checks |
| `algorithm` | string | JWT etc. when set |

**EnLocale examples:**

```text
Invalid email address
Invalid UUID
Invalid string: must start with "https://"
Invalid string: must match pattern ^[a-z]+$
```

---

## 5. `not_multiple_of`

Number not divisible by divisor.

| Field | Type |
|---|---|
| `divisor` | float64 |

**EnLocale:** `Invalid number: must be a multiple of {divisor}`

```text
Invalid number: must be a multiple of 5
```

---

## 6. `unrecognized_keys`

Object strict / catchall-never rejected extra keys.

| Field | Type |
|---|---|
| `keys` | `[]string` | sorted unrecognized keys |

**EnLocale:** `Unrecognized key(s): {keys}`

```text
Unrecognized key: "extra"
Unrecognized keys: "a", "b"
```

---

## 7. `invalid_union`

Union / discriminated-union failure.

| Field | Type | When |
|---|---|---|
| `errors` | `[][]Issue` | per-option issues (plain union) |
| `discriminator` | string | disc-union unknown tag |
| `values` | `[]any` | allowed discriminator literals |

**EnLocale:**

- With `values`: `Invalid discriminator value. Expected 'a' | 'b'`
- Otherwise: `Invalid input`

```text
Invalid discriminator value. Expected 'admin' | 'guest'
```

---

## 8. `invalid_key`

Record/map key validation failed.

| Field | Type |
|---|---|
| `origin` | string | `"record"` / `"map"` |
| `issues` | `[]Issue` | nested key issues |
| `key` | any | offending key when set |

**EnLocale:** `Invalid key in {origin}`

```text
Invalid key in record
```

---

## 9. `invalid_element`

Map/set element validation failed.

| Field | Type |
|---|---|
| `origin` | string | `"map"` / `"set"` … |
| `issues` | `[]Issue` | nested |
| `key` | any | optional |

**EnLocale:** `Invalid value in {origin}`

```text
Invalid value in map
```

---

## 10. `invalid_value`

Literal / enum mismatch.

| Field | Type |
|---|---|
| `values` | `[]any` | allowed values |

**EnLocale:**

- One value: `Invalid input: expected {value}`
- Many: `Invalid option: expected one of {a}|{b}`

```text
Invalid input: expected "admin"
Invalid option: expected one of "user"|"admin"
```

---

## 11. `custom`

Refinements, catch merge failures, decode errors, user issues.

| Field | Type |
|---|---|
| `params` | `map[string]any` | optional metadata |
| `message` | string | often set explicitly |

**EnLocale:** falls through to `Invalid input` when message empty.

```text
passwords must match
Unmergeable intersection results
```

---

## Finalize chain

Message resolution order (schema-compatible):

1. Explicit `Issue.Message`
2. Producing schema/check `ErrorMap`
3. Per-parse `ParseCtx.Error`
4. Global `Config.CustomError`
5. Locale (`Config.LocaleError`, default `EnLocale`)
6. `"Invalid input"`

```go
cfg:= z.GetConfig
// cfg.LocaleError = z.EsLocale  // etc.
```
