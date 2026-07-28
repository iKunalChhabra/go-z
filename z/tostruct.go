package z

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
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
	// nested is non-nil when typ (or its element, for pointers) is a struct.
	nested *structPlan
	// elem is the element plan for slices and arrays.
	elem      *fieldPlan
	isPtr     bool
	isSlice   bool
	isArray   bool
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
		plan = buildStructPlan(typ, make(map[reflect.Type]*structPlan))
		decodePlans.Store(typ, plan)
	}
	return decodeWithPlan[T](plan, data)
}

// planFor returns the cached decode plan for typ, building it once.
func planFor(typ reflect.Type) *structPlan {
	if plan, ok := decodePlans.Load(typ); ok {
		return plan.(*structPlan)
	}
	plan := buildStructPlan(typ, make(map[reflect.Type]*structPlan))
	actual, _ := decodePlans.LoadOrStore(typ, plan)
	return actual.(*structPlan)
}

// buildStructPlan builds the decode plan for typ. pending carries plans that
// are mid-build within this call tree, so a recursive type (type Node struct {
// Next *Node }) reuses its own in-progress plan instead of recursing forever.
func buildStructPlan(typ reflect.Type, pending map[reflect.Type]*structPlan) *structPlan {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if plan, ok := pending[typ]; ok {
		return plan
	}
	plan := &structPlan{typ: typ}
	pending[typ] = plan
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
		plan.fields = append(plan.fields, buildFieldPlan(sf, name, omit, pending))
	}

	// Then the fields of embedded structs, flattened into this level.
	for _, sf := range embedded {
		et := sf.Type
		if et.Kind() == reflect.Ptr {
			et = et.Elem()
		}
		for _, ef := range buildStructPlan(et, pending).fields {
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

func buildFieldPlan(sf reflect.StructField, name string, omitEmpty bool, pending map[reflect.Type]*structPlan) fieldPlan {
	fp := fieldPlan{
		jsonName:  name,
		index:     sf.Index,
		omitEmpty: omitEmpty,
	}
	buildTypePlan(sf.Type, &fp, pending)
	return fp
}

// buildTypePlan fills fp with the decode shape of t: pointer unwrapping, nested
// struct plans, and element plans for slices and arrays (scalar, struct, or
// nested-slice elements alike).
func buildTypePlan(t reflect.Type, fp *fieldPlan, pending map[reflect.Type]*structPlan) {
	fp.typ = t
	if t.Kind() == reflect.Ptr {
		fp.isPtr = true
		t = t.Elem()
	}
	fp.kind = t.Kind()
	switch t.Kind() {
	case reflect.Struct:
		if t != reflect.TypeOf(time.Time{}) {
			fp.nested = buildStructPlan(t, pending)
		}
	case reflect.Slice:
		fp.isSlice = true
		fp.elem = &fieldPlan{}
		buildTypePlan(t.Elem(), fp.elem, pending)
	case reflect.Array:
		fp.isArray = true
		fp.elem = &fieldPlan{}
		buildTypePlan(t.Elem(), fp.elem, pending)
	}
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
		elem := reflect.New(fv.Type().Elem())
		fv.Set(elem)
		target = elem.Elem()
	}

	if fp.nested != nil {
		m, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object, got %T", raw)
		}
		return applyPlan(fp.nested, target, m)
	}

	if fp.isSlice || fp.isArray {
		return setSlice(target, fp, raw)
	}

	return assignScalar(target, raw)
}

func setSlice(fv reflect.Value, fp *fieldPlan, raw any) error {
	// []byte mirrors encoding/json: a base64 string decodes into bytes.
	if fp.isSlice && fv.Type().Elem().Kind() == reflect.Uint8 {
		if s, ok := raw.(string); ok {
			b, err := decodeBase64(s)
			if err != nil {
				return fmt.Errorf("cannot decode base64 string into %s: %w", fv.Type(), err)
			}
			bv := reflect.ValueOf(b)
			if bv.Type().ConvertibleTo(fv.Type()) {
				fv.Set(bv.Convert(fv.Type()))
				return nil
			}
			// A named element type ([]Digit where Digit is a uint8) has the
			// right kind but is not slice-convertible; fill it element-wise
			// rather than letting Convert panic.
			out := reflect.MakeSlice(fv.Type(), len(b), len(b))
			for i, c := range b {
				out.Index(i).SetUint(uint64(c))
			}
			fv.Set(out)
			return nil
		}
	}

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

	// Fixed-size arrays decode in place and reject overflow input — there is no
	// grow operation the way a slice grows.
	if fp.isArray {
		if len(arr) > fv.Len() {
			return fmt.Errorf("expected at most %d elements for %s, got %d", fv.Len(), fv.Type(), len(arr))
		}
		for i, item := range arr {
			if err := setSliceElem(fv.Index(i), fp.elem, item); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
		return nil
	}

	slice := reflect.MakeSlice(fv.Type(), len(arr), len(arr))
	for i, item := range arr {
		if err := setSliceElem(slice.Index(i), fp.elem, item); err != nil {
			return fmt.Errorf("[%d]: %w", i, err)
		}
	}
	fv.Set(slice)
	return nil
}

// decodeBase64 accepts the standard and URL-safe alphabets, padded or raw.
// This is deliberately more permissive than encoding/json, which only takes
// padded StdEncoding; APIs commonly emit URL-safe or unpadded base64.
func decodeBase64(s string) ([]byte, error) {
	var err error
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		var b []byte
		if b, err = enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, err
}

// setSliceElem decodes one collection element: allocating pointer elements,
// descending into nested struct plans, recursing into nested slices, and
// assigning scalars.
func setSliceElem(ev reflect.Value, ep *fieldPlan, item any) error {
	target := ev
	if ep.isPtr {
		if item == nil {
			return nil
		}
		ptr := reflect.New(ev.Type().Elem())
		ev.Set(ptr)
		target = ptr.Elem()
	}
	if ep.nested != nil {
		m, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object, got %T", item)
		}
		return applyPlan(ep.nested, target, m)
	}
	if ep.isSlice || ep.isArray {
		return setSlice(target, ep, item)
	}
	return assignScalar(target, item)
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

	// The kind switch runs before the ConvertibleTo fallback: Go conversions
	// truncate (300 → int8 = 44) and stringify runes (65 → "A"), neither of
	// which is a decode anyone asked for.
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := scalarToInt64(raw)
		if err != nil {
			return fmt.Errorf("cannot assign %s to %s: %v", describeRaw(raw), fv.Type(), err)
		}
		if fv.OverflowInt(n) {
			return fmt.Errorf("value %d overflows %s", n, fv.Type())
		}
		fv.SetInt(n)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := scalarToUint64(raw)
		if err != nil {
			return fmt.Errorf("cannot assign %s to %s: %v", describeRaw(raw), fv.Type(), err)
		}
		if fv.OverflowUint(n) {
			return fmt.Errorf("value %d overflows %s", n, fv.Type())
		}
		fv.SetUint(n)
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
		// The AssignableTo fast path above already accepted anything that
		// satisfies the interface; reaching here means raw does not implement
		// it (e.g. a string into a fmt.Stringer field), so error rather than
		// let Set panic.
		return fmt.Errorf("cannot assign %T to %s", raw, fv.Type())
	}

	if rv.Type().ConvertibleTo(fv.Type()) {
		fv.Set(rv.Convert(fv.Type()))
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

// scalarToInt64 coerces a JSON scalar to int64 without loss: non-integral floats and
// values outside int64 range are errors, not truncations.
func scalarToInt64(v any) (int64, error) {
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int8:
		return int64(x), nil
	case int16:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case uint:
		return uintToInt64(uint64(x))
	case uint8:
		return int64(x), nil
	case uint16:
		return int64(x), nil
	case uint32:
		return int64(x), nil
	case uint64:
		return uintToInt64(x)
	case float32:
		return floatToInt64(float64(x))
	case float64:
		return floatToInt64(x)
	case json.Number:
		// Parse the literal exactly: integers above 2^53 lose precision
		// through float64, which is where database identifiers live.
		if n, err := strconv.ParseInt(x.String(), 10, 64); err == nil {
			return n, nil
		}
		if f, err := x.Float64(); err == nil {
			return floatToInt64(f)
		}
		return 0, fmt.Errorf("not a number")
	default:
		return 0, fmt.Errorf("not a number")
	}
}

func uintToInt64(u uint64) (int64, error) {
	if u > math.MaxInt64 {
		return 0, fmt.Errorf("value %d overflows int64", u)
	}
	return int64(u), nil
}

func floatToInt64(f float64) (int64, error) {
	if f != math.Trunc(f) {
		return 0, fmt.Errorf("non-integral value %v", f)
	}
	if f >= 1<<63 || f < -(1<<63) {
		return 0, fmt.Errorf("value %v overflows int64", f)
	}
	return int64(f), nil
}

// scalarToUint64 coerces a JSON scalar to uint64 without loss: negative and
// non-integral values are errors.
func scalarToUint64(v any) (uint64, error) {
	switch x := v.(type) {
	case int:
		return intToUint64(int64(x))
	case int8:
		return intToUint64(int64(x))
	case int16:
		return intToUint64(int64(x))
	case int32:
		return intToUint64(int64(x))
	case int64:
		return intToUint64(x)
	case uint:
		return uint64(x), nil
	case uint8:
		return uint64(x), nil
	case uint16:
		return uint64(x), nil
	case uint32:
		return uint64(x), nil
	case uint64:
		return x, nil
	case float32:
		return floatToUint64(float64(x))
	case float64:
		return floatToUint64(x)
	case json.Number:
		if u, err := strconv.ParseUint(x.String(), 10, 64); err == nil {
			return u, nil
		}
		if f, err := x.Float64(); err == nil {
			return floatToUint64(f)
		}
		return 0, fmt.Errorf("not a number")
	default:
		return 0, fmt.Errorf("not a number")
	}
}

func intToUint64(n int64) (uint64, error) {
	if n < 0 {
		return 0, fmt.Errorf("negative value %d", n)
	}
	return uint64(n), nil
}

func floatToUint64(f float64) (uint64, error) {
	if f != math.Trunc(f) {
		return 0, fmt.Errorf("non-integral value %v", f)
	}
	if f < 0 || f >= 1<<64 {
		return 0, fmt.Errorf("value %v out of uint64 range", f)
	}
	return uint64(f), nil
}

// describeRaw renders a rejected scalar for an error message, quoting strings
// the way %v renders numbers.
func describeRaw(raw any) string {
	if s, ok := raw.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%v", raw)
}
