# Registries & metadata

Registries attach metadata to schemas **without changing the schema**. The metadata travels alongside the schema and is consumed by exporters like [JSON Schema](#/api/json-schema), documentation generators, or your own tooling.

## Quick start

```go
email := z.Describe(z.String().Email(), "Primary contact address")

js, _ := z.ToJSONSchema(email)
// js["description"] == "Primary contact address"
```

`Describe` is sugar over the global registry — everything below is the machinery underneath it.

## The global registry

`z.GlobalRegistry` is a process-wide `*Registry[map[string]any]`. Three helpers read and write it:

```go
z.Meta(schema, map[string]any{
    "id":          "user-email",
    "title":       "Email",
    "description": "Primary contact address",
    "deprecated":  false,
})

z.Describe(schema, "Primary contact address") // sets "description"

z.GetDescription(schema) // "Primary contact address"
```

`Meta` and `Describe` **merge** into any existing entry for the schema (under a lock, so they are safe to call from concurrent setup code).

### GlobalMeta

For typed metadata, use the conventional shape `z.GlobalMeta`:

```go
type GlobalMeta struct {
    ID          string `json:"id,omitempty"`
    Title       string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
    Deprecated  bool   `json:"deprecated,omitempty"`
}
```

JSON Schema export reads exactly these four keys from a registry entry: `id`, `title`, `description`, `deprecated`.

## Custom registries

`z.NewRegistry[M]()` creates an isolated registry with your own metadata type — useful when several tools need different views of the same schemas:

```go
type DocsMeta struct {
    ID      string
    Section string
    Since   string
}

reg := z.NewRegistry[DocsMeta]()
reg.Add(userSchema, DocsMeta{ID: "user", Section: "models", Since: "v1.2"})

meta, ok := reg.Get(userSchema)
reg.Has(userSchema)      // true
reg.Remove(userSchema)   // delete one entry
reg.Clear()              // delete everything
```

Schemas are keyed by their internals pointer, so **clones are distinct entries** — fluent methods like `.Min(5)` return a new schema and do not inherit registrations made on the original. Register the final schema you hand out.

### Lookup by ID

If the metadata you `Add` carries an ID — a `"id"` string key in a map, or any struct field named `ID` / `Id` / tagged `json:"id"` — the registry indexes it for reverse lookup:

```go
z.Meta(z.String(), map[string]any{"id": "str-1"})

schema, ok := z.GlobalRegistry.GetByID("str-1")
// schema is the registered schema
```

```go
func (r *Registry[M]) GetByID(id string) (AnySchemaLike, bool)
```

## Feeding JSON Schema export

Pass a registry explicitly to keep export-time metadata separate from the global one:

```go
reg := z.NewRegistry[map[string]any]()
reg.Add(userSchema, map[string]any{
    "id":    "user",
    "title": "User",
})

js, err := z.ToJSONSchema(userSchema, z.ToJSONSchemaOpts{Metadata: reg})
```

Omitting `Metadata` reads from `z.GlobalRegistry`. See [JSON Schema export](#/api/json-schema) for what each key becomes in the document.

## Signatures

```go
func NewRegistry[M any]() *Registry[M]
func (r *Registry[M]) Add(schema AnySchemaLike, meta M) *Registry[M]
func (r *Registry[M]) Get(schema AnySchemaLike) (M, bool)
func (r *Registry[M]) Has(schema AnySchemaLike) bool
func (r *Registry[M]) Remove(schema AnySchemaLike)
func (r *Registry[M]) Clear()
func (r *Registry[M]) GetByID(id string) (AnySchemaLike, bool)

var GlobalRegistry = NewRegistry[map[string]any]()

func Meta(schema AnySchemaLike, meta map[string]any) AnySchemaLike
func Describe(schema AnySchemaLike, description string) AnySchemaLike
func GetDescription(schema AnySchemaLike) string
```

:::tip Not validation
Registries never affect parsing. A schema with metadata parses identically to one without — metadata is purely for tooling.
:::
