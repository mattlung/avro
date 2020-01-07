package avro

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/actgardner/gogen-avro/schema"

	"github.com/heetch/avro/avrotypegen"
)

// avroTypes is effectively a map[reflect.Type]*Type
// that holds Avro types for Go types that specify the schema
// entirely. Go types that don't fully specify a schema must be resolved
// with respect to a given writer schema and so cannot live in
// here.
//
// If there's an error translating a type, it's stored here as
// an errorSchema.
var goTypeToAvroType sync.Map

// errorSchema is a hack - it pretends to be an AvroType
// so that it can be held as a schema map value.
//
// In fact it just holds an error so that we can cache errors.
type errorSchema struct {
	schema.AvroType
	err error
}

// TypeOf returns the Avro type for the Go type of x when decoding
// a value of Avro type wType. The wType argument may be nil - in
// that case interface types are disallowed inside x (see below).
//
// If the type was generated by avro-generate-go, the returned schema
// will be the same as the schema it was generated from.
//
// Otherwise TypeOf(T, W) is derived according to
// the following rules:
//
//	- int, int64 and uint32 encode as "long"
//	- int32, int16, uint16, int8 and uint8 encode as "int"
//	- float32 encodes as "float"
//	- float64 encodes as "double"
//	- string encodes as "string"
//	- [N]byte encodes as {"type": "fixed", "name": "go.FixedN", "size": N}
//	- a named type with underlying type [N]byte encodes as [N]byte but typeName(T) for the name.
//	- []T encodes as {"type": "array", "items": TypeOf(T)}
//	- map[string]T encodes as {"type": "map", "values": TypeOf(T)}
//	- *T encodes as ["null", TypeOf(T)]
//	- a named struct type encodes as {"type": "record", "name": typeName(T), "fields": ...}
//		where the fields are encoded as described below.
//	- an interface type is allowed only when wType is non-nil, in which case
//		the interface acts as a placeholder and is filled in with the relevant part of wType.
//
// Struct fields are encoded as follows:
//
//	- unexported struct fields are ignored
//	- the field name is taken from the Go field name, or from a "json" tag for the field if present.
//	- the default value for the field is the zero value for the type.
//	- anonymous struct fields are currently disallowed
func TypeOf(x interface{}, wType *Type) (*Type, error) {
	return avroTypeOf(reflect.TypeOf(x), wType)
}

func avroTypeOf(t reflect.Type, wType *Type) (*Type, error) {
	// TODO cache Avro type for a given Go type
	if wType != nil {
		return nil, fmt.Errorf("TypeOf with writer type not yet supported")
	}
	rType0, ok := goTypeToAvroType.Load(t)
	if ok {
		rType := rType0.(*Type)
		if es, ok := rType.avroType.(errorSchema); ok {
			return nil, es.err
		}
		return rType, nil
	}
	rType, err := schemaForGoTypeUncached(t, wType)
	if err != nil {
		// TODO if the error was because it needs the writer schema,
		// invoke schemaForGoType1(t, wSchema).
		// Perhaps the caller should pass in a cache so we can
		// store the result without using the global cache.
		goTypeToAvroType.LoadOrStore(t, &Type{
			avroType: errorSchema{err: err},
		})
		return nil, err
	}
	goTypeToAvroType.LoadOrStore(t, rType)
	return rType, nil
}

func schemaForGoTypeUncached(t reflect.Type, wType *Type) (*Type, error) {
	gts := &goTypeSchema{
		defs: make(map[reflect.Type]goTypeDef),
	}
	// TODO pass in wType so that we can determine a schema
	// even for partially specified Go types (e.g. interface{} values)
	schemaVal, err := gts.schemaForGoType(t)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(schemaVal)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal generated schema: %v", err)
	}
	return ParseType(string(data))
}

type goTypeDef struct {
	name   string
	schema interface{}
}

type goTypeSchema struct {
	defs map[reflect.Type]goTypeDef
}

func (gts *goTypeSchema) schemaForGoType(t reflect.Type) (interface{}, error) {
	d, ok := gts.defs[t]
	if ok {
		// We've already defined a name for this type, so use it.
		return d.name, nil
	}

	if r, ok := reflect.Zero(t).Interface().(avrotypegen.AvroRecord); ok {
		// It's a generated type which comes with its own schema.
		// TODO the schema might refer to names that are used the
		// go type - we should de-duplicate those entries (probably
		// by name but also making sure that the names actually match).
		return gts.define(t, json.RawMessage(r.AvroRecord().Schema))
	}

	if syms := enumSymbols(t); len(syms) > 0 {
		// It looks like an enum.
		// TODO full names.
		def := map[string]interface{}{
			"name":    t.Name(),
			"symbols": syms,
			"default": syms[0],
		}
		gts.defs[t] = goTypeDef{
			name:   t.Name(),
			schema: def,
		}
		return def, nil
	}
	switch t.Kind() {
	case reflect.String:
		return "string", nil
	case reflect.Int, reflect.Int64, reflect.Uint32:
		return "long", nil
	case reflect.Int32, reflect.Int16, reflect.Uint16, reflect.Int8, reflect.Uint8:
		return "int", nil
	case reflect.Float32:
		return "float", nil
	case reflect.Float64:
		return "double", nil
	case reflect.Slice:
		items, err := gts.schemaForGoType(t.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"type":  "array",
			"items": items,
		}, nil
	case reflect.Map:
		// TODO support the same map keys types that JSON does.
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map must have string key")
		}
		values, err := gts.schemaForGoType(t.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"type":   "map",
			"values": values,
		}, nil
	case reflect.Struct:
		name := t.Name()
		if name == "" {
			return nil, fmt.Errorf("unnamed struct type")
		}
		for _, def := range gts.defs {
			if def.name == name {
				// TODO use package path to disambiguate.
				return nil, fmt.Errorf("duplicate struct type name %q", name)
			}
		}

		var fields []interface{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Anonymous {
				return nil, fmt.Errorf("anonymous fields not yet supported (in %s)", t)
			}
			// Technically in Go, every field is optional because
			// that's the way that the encoding/json package works,
			// so we'll make them all optional, but we could experiment by making optional
			// only the fields that specify omitempty.
			name, _ := jsonFieldName(f)
			ftype, err := gts.schemaForGoType(f.Type)
			if err != nil {
				return nil, err
			}
			fields = append(fields, map[string]interface{}{
				"name":    name,
				"default": defaultForType(f.Type),
				"type":    ftype,
			})
		}
		return gts.define(t, map[string]interface{}{
			"name":   name,
			"type":   "record",
			"fields": fields,
		})
	case reflect.Array:
		if t.Elem() != reflect.TypeOf(byte(0)) {
			return nil, fmt.Errorf("the only array type supported is [...]byte, not %s", t)
		}
		name := t.Name()
		if name == "" {
			name = fmt.Sprintf("go.Fixed%d", t.Len())
		}
		return gts.define(t, map[string]interface{}{
			"name": name,
			"type": "fixed",
			"size": t.Len(),
		})
	case reflect.Ptr:
		if t.Elem().Kind() == reflect.Ptr {
			return nil, fmt.Errorf("can only cope with a single level of pointer indirection")
		}
		elem, err := gts.schemaForGoType(t.Elem())
		if err != nil {
			return nil, err
		}
		return []interface{}{
			"null",
			elem,
		}, nil
	case reflect.Interface:
		// TODO fill in from the writer schema.
		return nil, fmt.Errorf("interface types (%s) not yet supported (use avro-generate-go instead)", t)
	default:
		return nil, fmt.Errorf("cannot make Avro schema for Go type %s", t)
	}
}

func (gts *goTypeSchema) define(t reflect.Type, def0 interface{}) (interface{}, error) {
	def, ok := def0.(map[string]interface{})
	if !ok {
		if err := json.Unmarshal(def0.(json.RawMessage), &def); err != nil {
			return nil, err
		}
	}
	name, _ := def["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("definition with empty name")
	}
	for _, def := range gts.defs {
		if def.name == name {
			// TODO use package path to disambiguate.
			return nil, fmt.Errorf("duplicate struct type name %q", name)
		}
	}
	gts.defs[t] = goTypeDef{
		name:   name,
		schema: def,
	}
	return def, nil
}

const maxEnum = 250

// enumSymbols returns the enum symbols represented by the given
// type. If the type doesn't represent an enum it returns no symbols.
func enumSymbols(t reflect.Type) []string {
	k := t.Kind()
	isSignedInt := reflect.Int <= k && k <= reflect.Int64
	isUnsignedInt := reflect.Uint <= k && k <= reflect.Uint64
	if !isSignedInt && !isUnsignedInt {
		return nil
	}
	if _, ok := reflect.Zero(t).Interface().(fmt.Stringer); !ok {
		return nil
	}
	v := reflect.New(t)
	vs := v.Interface().(fmt.Stringer) // Note: pointer type will also include String method.
	setInt := v.SetInt
	getIntVal := v.Int
	if isUnsignedInt {
		setInt = func(i int64) {
			v.SetUint(uint64(i))
		}
		getIntVal = func() int64 {
			return int64(v.Int())
		}
	}
	symOf := func(i int64) (sym string, actual int64, ok bool) {
		defer func() {
			// It panics when calling String, which is a decent indication
			// that it's out of bounds.
			if recover() != nil {
				ok = false
			}
		}()
		setInt(i)
		return vs.String(), getIntVal(), true
	}
	sym, _, ok := symOf(-1)
	// Note: the String implementation created by the stringer tool
	// returns "T(x)" for an out-of-bounds number x of type T
	// so we use a bracket as an indicator of "out of bounds".
	// TODO we could look for the numeric value of the enum too
	// to cover more formats.
	if ok && !strings.Contains(sym, "(") {
		// If -1 is OK, then our heuristic isn't going to work.
		return nil
	}
	prev := ""
	var syms []string
	for i := 0; i < maxEnum; i++ {
		sym, _, ok := symOf(int64(i))
		if !ok || strings.Contains(sym, "(") || sym == "" {
			return syms
		}
		if sym == prev {
			// If it's the same as the previous value, it might be "unknown"
			// or something, so treat both it and the previous value as
			// out-of-bounds.
			return syms[0 : len(syms)-1]
		}
		// TODO cope with non-Avro-compatible symbols. Avro symbols must match [A-Za-z_][A-Za-z0-9_]*
		syms = append(syms, sym)
		prev = sym
	}
	// Too many values.
	return nil
}

func defaultForType(t reflect.Type) interface{} {
	// TODO perhaps a Go slice/map should accept a union
	// of null and array/map?
	switch t.Kind() {
	case reflect.Slice:
		return reflect.MakeSlice(t, 0, 0).Interface()
	case reflect.Map:
		return reflect.MakeMap(t).Interface()
	case reflect.Array:
		return strings.Repeat("\u0000", t.Len())
	default:
		return reflect.Zero(t).Interface()
	}
}

// jsonFieldName returns the name that the field will be given
// when marshaled to JSON, or the empty string if
// the field is ignored.
// It also reports whether the field has been qualified with
// the "omitempty" qualifier.
func jsonFieldName(f reflect.StructField) (name string, omitEmpty bool) {
	if f.PkgPath != "" {
		// It's unexported.
		return "", false
	}
	tag := f.Tag.Get("json")
	parts := strings.Split(tag, ",")
	for _, part := range parts[1:] {
		if part == "omitempty" {
			omitEmpty = true
		}
	}
	switch {
	case parts[0] == "":
		return f.Name, omitEmpty
	case parts[1] == "-":
		return "", omitEmpty
	}
	return parts[0], omitEmpty
}

var recordField struct {
	Name    string
	Type    interface{}
	Default interface{}
}

type arrayType struct {
	Type  string `json:"type"` // always "array"
	Items interface{}
}
