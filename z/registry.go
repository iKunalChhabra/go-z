package z

import (
	"reflect"
	"strings"
	"sync"
)

// GlobalMeta is the conventional metadata shape stored in GlobalRegistry
// (GlobalMeta / JSONSchemaMeta). Callers may also use map[string]any.
type GlobalMeta struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
}

// Registry maps schemas to metadata, Schemas are keyed
// by *Internals pointer identity (schemas are immutable after build).
type Registry[M any] struct {
	mu    sync.RWMutex
	meta  map[*Internals]M
	idmap map[string]AnySchemaLike
}

// NewRegistry creates an empty schema→meta registry.
func NewRegistry[M any]() *Registry[M] {
	return &Registry[M]{
		meta:  map[*Internals]M{},
		idmap: map[string]AnySchemaLike{},
	}
}

// GlobalRegistry is the process-wide metadata registry (globalRegistry).
var GlobalRegistry = NewRegistry[map[string]any]()

// Add associates meta with schema. When meta carries an "id" string, it is
// indexed for GetByID (later adds with the same id silently overwrite).
func (r *Registry[M]) Add(schema AnySchemaLike, meta M) *Registry[M] {
	if schema == nil {
		return r
	}
	in := schema.Internals()
	r.mu.Lock()
	defer r.mu.Unlock()
	if prev, ok := r.meta[in]; ok {
		if id, ok := extractMetaID(prev); ok {
			if existing, exists := r.idmap[id]; exists && existing.Internals() == in {
				delete(r.idmap, id)
			}
		}
	}
	r.meta[in] = meta
	if id, ok := extractMetaID(meta); ok {
		r.idmap[id] = schema
	}
	return r
}

// Get returns the metadata for schema, if registered.
func (r *Registry[M]) Get(schema AnySchemaLike) (M, bool) {
	var zero M
	if schema == nil {
		return zero, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.meta[schema.Internals()]
	return m, ok
}

// Has reports whether schema is registered.
func (r *Registry[M]) Has(schema AnySchemaLike) bool {
	if schema == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.meta[schema.Internals()]
	return ok
}

// Remove unregisters schema and clears any id index entry it owned.
func (r *Registry[M]) Remove(schema AnySchemaLike) {
	if schema == nil {
		return
	}
	in := schema.Internals()
	r.mu.Lock()
	defer r.mu.Unlock()
	if prev, ok := r.meta[in]; ok {
		if id, ok := extractMetaID(prev); ok {
			if existing, exists := r.idmap[id]; exists && existing.Internals() == in {
				delete(r.idmap, id)
			}
		}
		delete(r.meta, in)
	}
}

// Clear removes all entries.
func (r *Registry[M]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.meta = map[*Internals]M{}
	r.idmap = map[string]AnySchemaLike{}
}

// GetByID returns the schema registered under id, if any.
func (r *Registry[M]) GetByID(id string) (AnySchemaLike, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.idmap[id]
	return s, ok
}

// Describe stores {description} on schema in GlobalRegistry (merging with any
// existing meta) and returns schema. Package-level stand-in for .describe().
func Describe(schema AnySchemaLike, description string) AnySchemaLike {
	if schema == nil {
		return nil
	}
	meta := copyStringAnyMap(nil)
	if existing, ok := GlobalRegistry.Get(schema); ok {
		meta = copyStringAnyMap(existing)
	}
	meta["description"] = description
	GlobalRegistry.Add(schema, meta)
	return schema
}

// Meta merges meta into schema's GlobalRegistry entry and returns schema.
// Package-level stand-in for .meta(data).
func Meta(schema AnySchemaLike, meta map[string]any) AnySchemaLike {
	if schema == nil {
		return nil
	}
	merged := copyStringAnyMap(nil)
	if existing, ok := GlobalRegistry.Get(schema); ok {
		merged = copyStringAnyMap(existing)
	}
	for k, v := range meta {
		merged[k] = v
	}
	GlobalRegistry.Add(schema, merged)
	return schema
}

// GetDescription returns schema's description from GlobalRegistry, or "".
func GetDescription(schema AnySchemaLike) string {
	if schema == nil {
		return ""
	}
	meta, ok := GlobalRegistry.Get(schema)
	if !ok || meta == nil {
		return ""
	}
	if d, ok := meta["description"].(string); ok {
		return d
	}
	return ""
}

func copyStringAnyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func extractMetaID(meta any) (string, bool) {
	if meta == nil {
		return "", false
	}
	switch m := meta.(type) {
	case map[string]any:
		if id, ok := m["id"].(string); ok && id != "" {
			return id, true
		}
		return "", false
	case GlobalMeta:
		if m.ID != "" {
			return m.ID, true
		}
		return "", false
	case *GlobalMeta:
		if m != nil && m.ID != "" {
			return m.ID, true
		}
		return "", false
	}
	rv := reflect.ValueOf(meta)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "", false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", false
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		name := f.Name
		tag := f.Tag.Get("json")
		if tag != "" && tag != "-" {
			if before, _, ok := strings.Cut(tag, ","); ok {
				tag = before
			}
			if tag != "" {
				name = tag
			}
		}
		if name != "id" && f.Name != "ID" && f.Name != "Id" {
			continue
		}
		fv := rv.Field(i)
		if fv.Kind() == reflect.String {
			id := fv.String()
			if id != "" {
				return id, true
			}
		}
	}
	return "", false
}
