package z

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// decodePlans caches decode plans. A plan depends only on the target type, so
// the type is the whole key: keying it by schema as well would add an entry per
// schema instance and never release one.
var decodePlans sync.Map // reflect.Type → *structPlan

// structPlan is a build-once reflection plan for decoding map[string]any → T.
// Hot-path decode walks this plan only — no per-parse type walks.
type structPlan struct {
	typ    reflect.Type
	fields []fieldPlan
}

type fieldPlan struct {
	jsonName string
	index    []int
	typ      reflect.Type
	kind     reflect.Kind
	// nested is non-nil when typ (or its element, for pointers/slices) is a struct.
	nested *structPlan
	// elem is the element type plan for slices/arrays of structs.
	elem      *structPlan
	isPtr     bool
	isSlice   bool
	omitEmpty bool // json:",omitempty" — unused on decode, kept for tag fidelity
}

// ToStruct wraps schema (expecting map[string]any output) and decodes into T
// using a reflection plan cached per target type. Honors `json` struct tags,
// including promotion of embedded struct fields.
func ToStruct[T any](schema AnySchemaLike) Schema[T] {
	if schema == nil {
		panic("go-z: ToStruct: schema is nil")
	}
	var zero T
	typ := reflect.TypeOf(zero)
	if typ == nil {
		// T is an interface with nil type — use pointer elem of *T
		typ = reflect.TypeOf((*T)(nil)).Elem()
	}
	if typ.Kind() == reflect.Ptr {
		panic("go-z: ToStruct: T must be a non-pointer struct type")
	}
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("go-z: ToStruct: T must be a struct, got %s", typ.Kind()))
	}

	plan := planFor(typ)
	innerIn := schema.Internals()
	s := &toStructSchema[T]{inner: schema, plan: plan}
	parse := func(p *Payload, ctx *ParseCtx) {
		RunSelf(innerIn, p, ctx)
		if len(p.Issues) > 0 {
			return
		}
		m, ok := p.Value.(map[string]any)
		if !ok {
			if v, isT := p.Value.(T); isT {
				p.Value = v
				return
			}
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "object",
				Input:    p.Value,
			})
			return
		}
		out, err := decodeWithPlan[T](plan, m)
		if err != nil {
			p.AddIssue(Issue{
				Code:    IssueCustom,
				Message: err.Error(),
				Input:   p.Value,
			})
			return
		}
		p.Value = out
	}
	s.schemaBase = newBase[T](buildInternals(&Def{Type: "tostruct"}, parse))
	return s
}

type toStructSchema[T any] struct {
	schemaBase[T]
	inner AnySchemaLike
	plan  *structPlan
}

// Unwrap returns the validated schema underneath the struct decoding, so callers
// can introspect it the way they can through any other wrapper.
func (s *toStructSchema[T]) Unwrap() AnySchemaLike { return s.inner }

// DecodeStruct decodes data into T without schema validation, using a
// reflection plan cached by typeof(T).
func DecodeStruct[T any](data map[string]any) (T, error) {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ == nil {
		typ = reflect.TypeOf((*T)(nil)).Elem()
	}
	if typ.Kind() != reflect.Struct {
		return zero, fmt.Errorf("go-z: DecodeStruct: T must be a struct, got %s", typ.Kind())
	}
	planAny, ok := decodePlans.Load(typ)
	var plan *structPlan
	if ok {
		plan = planAny.(*structPlan)
	} else {
		plan = buildStructPlan(typ)
		decodePlans.Store(typ, plan)
	}
	return decodeWithPlan[T](plan, data)
}

// planFor returns the cached decode plan for typ, building it once.
func planFor(typ reflect.Type) *structPlan {
	if plan, ok := decodePlans.Load(typ); ok {
		return plan.(*structPlan)
	}
	plan := buildStructPlan(typ)
	actual, _ := decodePlans.LoadOrStore(typ, plan)
	return actual.(*structPlan)
}

func buildStructPlan(typ reflect.Type) *structPlan {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	plan := &structPlan{typ: typ}
	n := typ.NumField()
	plan.fields = make([]fieldPlan, 0, n)
	seen := make(map[string]bool, n)

	// Direct fields first: a field at this level wins over one promoted from an
	// embedded struct, the way encoding/json resolves the same conflict.
	var embedded []reflect.StructField
	for i := range n {
		sf := typ.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue // unexported
		}
		if isPromotableEmbed(sf) {
			embedded = append(embedded, sf)
			continue
		}
		name, omit, skip := parseJSONTag(sf.Tag.Get("json"), sf.Name)
		if skip || seen[name] {
			continue
		}
		seen[name] = true
		plan.fields = append(plan.fields, buildFieldPlan(sf, name, omit))
	}

	// Then the fields of embedded structs, flattened into this level.
	for _, sf := range embedded {
		et := sf.Type
		if et.Kind() == reflect.Ptr {
			et = et.Elem()
		}
		for _, ef := range buildStructPlan(et).fields {
			if seen[ef.jsonName] {
				continue
			}
			seen[ef.jsonName] = true
			promoted := ef
			promoted.index = append(append(make([]int, 0, len(sf.Index)+len(ef.index)), sf.Index...), ef.index...)
			plan.fields = append(plan.fields, promoted)
		}
	}
	return plan
}

// isPromotableEmbed reports whether sf is an embedded struct whose fields belong
// at the parent level. An embedded field with a json tag is a named field like
// any other, and time.Time is a value, not a shape to flatten.
func isPromotableEmbed(sf reflect.StructField) bool {
	if !sf.Anonymous || sf.Tag.Get("json") != "" {
		return false
	}
	t := sf.Type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct && t != reflect.TypeOf(time.Time{})
}

func buildFieldPlan(sf reflect.StructField, name string, omitEmpty bool) fieldPlan {
	fp := fieldPlan{
		jsonName:  name,
		index:     sf.Index,
		typ:       sf.Type,
		omitEmpty: omitEmpty,
	}
	t := sf.Type
	if t.Kind() == reflect.Ptr {
		fp.isPtr = true
		t = t.Elem()
	}
	fp.kind = t.Kind()
	switch t.Kind() {
	case reflect.Struct:
		if t != reflect.TypeOf(time.Time{}) {
			fp.nested = buildStructPlan(t)
		}
	case reflect.Slice, reflect.Array:
		fp.isSlice = true
		et := t.Elem()
		if et.Kind() == reflect.Ptr {
			et = et.Elem()
		}
		if et.Kind() == reflect.Struct && et != reflect.TypeOf(time.Time{}) {
			fp.elem = buildStructPlan(et)
		}
	}
	return fp
}

func parseJSONTag(tag, fieldName string) (name string, omitEmpty bool, skip bool) {
	if tag == "-" {
		return "", false, true
	}
	if tag == "" {
		return fieldName, false, false
	}
	// name,omitempty
	name = tag
	if i := indexByte(tag, ','); i >= 0 {
		name = tag[:i]
		rest := tag[i+1:]
		omitEmpty = containsToken(rest, "omitempty")
	}
	if name == "" {
		name = fieldName
	}
	return name, omitEmpty, false
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func containsToken(s, tok string) bool {
	for len(s) > 0 {
		var part string
		if i := indexByte(s, ','); i >= 0 {
			part = s[:i]
			s = s[i+1:]
		} else {
			part = s
			s = ""
		}
		if part == tok {
			return true
		}
	}
	return false
}

func decodeWithPlan[T any](plan *structPlan, data map[string]any) (T, error) {
	var zero T
	v := reflect.New(plan.typ).Elem()
	if err := applyPlan(plan, v, data); err != nil {
		return zero, err
	}
	out, ok := v.Interface().(T)
	if !ok {
		return zero, fmt.Errorf("go-z: DecodeStruct: type assertion to %T failed", zero)
	}
	return out, nil
}

func applyPlan(plan *structPlan, dest reflect.Value, data map[string]any) error {
	if data == nil {
		return nil
	}
	for i := range plan.fields {
		fp := &plan.fields[i]
		raw, exists := data[fp.jsonName]
		if !exists || raw == nil {
			continue
		}
		fv, err := fieldByIndexAlloc(dest, fp.index)
		if err != nil {
			return fmt.Errorf("%s: %w", fp.jsonName, err)
		}
		if err := setField(fv, fp, raw); err != nil {
			return fmt.Errorf("%s: %w", fp.jsonName, err)
		}
	}
	return nil
}

// fieldByIndexAlloc walks an index path to a field, allocating any nil pointer
// it passes through. reflect.Value.FieldByIndex panics on such a pointer, and the
// path crosses one whenever a struct embeds *Base — which encoding/json handles by
// allocating, so a request body that only mentions a promoted field cannot be
// turned into a crash.
func fieldByIndexAlloc(v reflect.Value, index []int) (reflect.Value, error) {
	for i, x := range index {
		if i > 0 && v.Kind() == reflect.Ptr {
			if v.Type().Elem().Kind() != reflect.Struct {
				return reflect.Value{}, fmt.Errorf("cannot reach field through %s", v.Type())
			}
			if v.IsNil() {
				if !v.CanSet() {
					// An embedded pointer to an unexported type cannot be
					// allocated, the same limitation encoding/json reports.
					return reflect.Value{}, fmt.Errorf("cannot set embedded pointer to unexported type %s", v.Type().Elem())
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(x)
	}
	return v, nil
}

func setField(fv reflect.Value, fp *fieldPlan, raw any) error {
	if !fv.CanSet() {
		return fmt.Errorf("cannot set field")
	}
	target := fv
	if fp.isPtr {
		if raw == nil {
			fv.Set(reflect.Zero(fv.Type()))
			return nil
		}
		elem := reflect.New(fv.Type().Elem())
		fv.Set(elem)
		target = elem.Elem()
	}

	if fp.nested != nil && !fp.isSlice {
		m, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object, got %T", raw)
		}
		return applyPlan(fp.nested, target, m)
	}

	if fp.isSlice {
		return setSlice(target, fp, raw)
	}

	return assignScalar(target, raw)
}

func setSlice(fv reflect.Value, fp *fieldPlan, raw any) error {
	var arr []any
	switch v := raw.(type) {
	case []any:
		arr = v
	case []string:
		arr = make([]any, len(v))
		for i := range v {
			arr[i] = v[i]
		}
	default:
		rv := reflect.ValueOf(raw)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return fmt.Errorf("expected array, got %T", raw)
		}
		arr = make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			arr[i] = rv.Index(i).Interface()
		}
	}

	slice := reflect.MakeSlice(fv.Type(), len(arr), len(arr))
	elemType := fv.Type().Elem()
	for i, item := range arr {
		ev := slice.Index(i)
		if elemType.Kind() == reflect.Ptr {
			if item == nil {
				continue
			}
			ptr := reflect.New(elemType.Elem())
			ev.Set(ptr)
			ev = ptr.Elem()
			if fp.elem != nil {
				m, ok := item.(map[string]any)
				if !ok {
					return fmt.Errorf("[%d]: expected object, got %T", i, item)
				}
				if err := applyPlan(fp.elem, ev, m); err != nil {
					return err
				}
				continue
			}
			if err := assignScalar(ev, item); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
			continue
		}
		if fp.elem != nil {
			m, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("[%d]: expected object, got %T", i, item)
			}
			if err := applyPlan(fp.elem, ev, m); err != nil {
				return err
			}
			continue
		}
		if err := assignScalar(ev, item); err != nil {
			return fmt.Errorf("[%d]: %w", i, err)
		}
	}
	fv.Set(slice)
	return nil
}

func assignScalar(fv reflect.Value, raw any) error {
	if raw == nil {
		fv.Set(reflect.Zero(fv.Type()))
		return nil
	}

	// time.Time from time.Time or RFC3339 string
	if fv.Type() == reflect.TypeOf(time.Time{}) {
		switch t := raw.(type) {
		case time.Time:
			fv.Set(reflect.ValueOf(t))
			return nil
		case string:
			parsed, err := time.Parse(time.RFC3339Nano, t)
			if err != nil {
				parsed, err = time.Parse(time.RFC3339, t)
			}
			if err != nil {
				return fmt.Errorf("cannot parse time: %w", err)
			}
			fv.Set(reflect.ValueOf(parsed))
			return nil
		default:
			return fmt.Errorf("cannot assign %T to time.Time", raw)
		}
	}

	rv := reflect.ValueOf(raw)
	if rv.Type().AssignableTo(fv.Type()) {
		fv.Set(rv)
		return nil
	}
	if rv.Type().ConvertibleTo(fv.Type()) {
		fv.Set(rv.Convert(fv.Type()))
		return nil
	}

	// JSON numbers decode as float64 — coerce into numeric fields.
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f, ok := toFloat64(raw)
		if !ok {
			return fmt.Errorf("cannot assign %T to %s", raw, fv.Type())
		}
		fv.SetInt(int64(f))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		f, ok := toFloat64(raw)
		if !ok {
			return fmt.Errorf("cannot assign %T to %s", raw, fv.Type())
		}
		fv.SetUint(uint64(f))
		return nil
	case reflect.Float32, reflect.Float64:
		f, ok := toFloat64(raw)
		if !ok {
			return fmt.Errorf("cannot assign %T to %s", raw, fv.Type())
		}
		fv.SetFloat(f)
		return nil
	case reflect.Bool:
		switch b := raw.(type) {
		case bool:
			fv.SetBool(b)
			return nil
		case string:
			switch b {
			case "true", "1":
				fv.SetBool(true)
				return nil
			case "false", "0":
				fv.SetBool(false)
				return nil
			}
		}
	case reflect.String:
		switch s := raw.(type) {
		case string:
			fv.SetString(s)
			return nil
		case bool, int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64, float32, float64:
			// A JSON scalar in a string field is a reasonable coercion.
			fv.SetString(fmt.Sprint(raw))
			return nil
		}
		// Anything else — an object, an array — would be stringified into
		// garbage like "map[]", so refuse it instead.
		return fmt.Errorf("cannot assign %T to %s", raw, fv.Type())
	case reflect.Map:
		return assignMap(fv, raw)
	case reflect.Interface:
		fv.Set(reflect.ValueOf(raw))
		return nil
	}

	return fmt.Errorf("cannot assign %T to %s", raw, fv.Type())
}

// assignMap decodes a JSON object into a typed map, converting each key and value
// the same way a struct field would be. map[string]any assigns directly and never
// reaches here.
func assignMap(fv reflect.Value, raw any) error {
	rv := reflect.ValueOf(raw)
	if rv.Kind() != reflect.Map {
		return fmt.Errorf("cannot assign %T to %s", raw, fv.Type())
	}
	mapType := fv.Type()
	out := reflect.MakeMapWithSize(mapType, rv.Len())
	for iter := rv.MapRange(); iter.Next(); {
		key := reflect.New(mapType.Key()).Elem()
		if err := assignScalar(key, iter.Key().Interface()); err != nil {
			return fmt.Errorf("key %v: %w", iter.Key().Interface(), err)
		}
		val := reflect.New(mapType.Elem()).Elem()
		if err := assignScalar(val, iter.Value().Interface()); err != nil {
			return fmt.Errorf("[%v]: %w", iter.Key().Interface(), err)
		}
		out.SetMapIndex(key, val)
	}
	fv.Set(out)
	return nil
}

func toFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	default:
		return 0, false
	}
}
