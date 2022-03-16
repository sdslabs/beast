// Package struct2csv creates slices of strings out of struct fields.
// Struct2csv can work with either a struct or a slice of structs,  The data
// can be returned as []string, in the case of a single struct, or [][]string,
// in the case of a slice of structs, by using struct2csv's encoder and
// `Marshal`.
//
// For writing struct data directly to a CSV file, a writer can be used.
// `NewWriter` accepts an `io.Writer` and uses it to create a `csv.Writer`,
// which is used for writing the generated data.  The `csv.Writer`'s exported
// fields are exposed as methods.
//
// Fields of Kind Uintptr, Unsafepointer, Chan, Func, and Interface are not
// supported.
//
// Unexported fields will be ignored.
//
// Field tags are supported.  Struct2csv will look for field tags matching
// `csv`, unless it's configured to either ignore field tags or use different
// field tag.
package struct2csv

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// A StructRequiredError is returned when a non-struct type is received.
type StructRequiredError struct {
	kind reflect.Kind
}

func (e StructRequiredError) Error() string {
	return fmt.Sprintf("struct2csv: a value of type struct is required: type was %s", e.kind)
}

// A StructSliceError is returned when an interface that isn't a slice of
// type struct is received.
type StructSliceError struct {
	kind      reflect.Kind
	sliceKind reflect.Kind
}

func (e StructSliceError) Error() string {
	if e.kind != reflect.Slice {
		return fmt.Sprintf("struct2csv: a type of slice is required: type was %s", e.kind)
	}
	return fmt.Sprintf("struct2csv: a slice of type struct is required: slice type was %s", e.sliceKind)
}

var (
	// ErrNilSlice occurs when the slice of structs to encode is nil.
	ErrNilSlice = errors.New("struct2csv: the slice of structs was nil")
	// ErrEmptySlice occurs when the slice of structs to encode is empty.
	ErrEmptySlice = errors.New("struct2csv: the slice of structs was empty")
)

// Below is implemented from
// https://golang.org/src/encoding/json/encode.go#L773 through L780
// This is the copyright of the original code:
// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// stringValues is a slice of reflect.Value holding *reflect.StringValue.
// It implements the methods to sort by string.
type stringValues []reflect.Value

func (sv stringValues) Len() int           { return len(sv) }
func (sv stringValues) Swap(i, j int)      { sv[i], sv[j] = sv[j], sv[i] }
func (sv stringValues) Less(i, j int) bool { return sv.get(i) < sv.get(j) }
func (sv stringValues) get(i int) string   { return sv[i].String() }

// Encoder handles encoding of a CSV from a struct.
type Encoder struct {
	// Whether or not tags should be use for header (column) names; by default this is csv,
	useTags  bool
	base     int
	tag      string // The tag to use when tags are being used for headers; defaults to csv.
	sepBeg   string
	sepEnd   string
	colNames []string
}

// New returns an initialized Encoder.
func New() *Encoder {
	return &Encoder{
		useTags: true, base: 10, tag: "csv",
		sepBeg: "(", sepEnd: ")",
	}
}

// SetTag sets the tag that the Encoder should use for header (column)
// names.  By default, this is set to 'csv'.  If the received value is an
// empty string, nothing will be done
func (e *Encoder) SetTag(s string) {
	if s == "" {
		return
	}
	e.tag = s
}

// SetUseTags sets whether or not tags should be used for header (column)
// names.
func (e *Encoder) SetUseTags(b bool) {
	e.useTags = b
}

// SetSeparators sets the begin and end separator values for lists.   Setting
// the separators to "", empty strings, results in no separators being added.
// By default, "(" and ")" are used as the begin and end separators,
func (e *Encoder) SetSeparators(beg, end string) {
	e.sepBeg = beg
	e.sepEnd = end
}

// SetBase sets the base for strings.FormatUint. By default, this is 10. Set
// the base if another base should be used for formatting uint values.
//
// Base 2 is the minimum value; anything less will be set to two.
func (e *Encoder) SetBase(i int) {
	if i < 2 {
		i = 2
	}
	e.base = i
}

// ColNames returns the encoder's saved column names as a copy.  The
// colNames field must be populated before using this.
func (e *Encoder) ColNames() []string {
	ret := make([]string, len(e.colNames))
	_ = copy(ret, e.colNames)
	return ret
}

// GetColNames get's the column names from the received struct.  If the
// interface is not a struct, an error will occur.
//
// Field tags are supported. By default, the column names will be the value
// of the `csv` tag, if any.  This can be changed with the SetTag(newTag)
// func; e.g. `json` to use JSON tags.  Use of field tags can be toggled with
// the the SetUseTag(bool) func.  If use of field tags is set to FALSE, the
// field's name will be used.
func (e *Encoder) GetColNames(v interface{}) ([]string, error) {
	if reflect.TypeOf(v).Kind() != reflect.Struct {
		return nil, StructRequiredError{reflect.TypeOf(v).Kind()}
	}
	// the returned bool is ignored because it's only used for recursive calls.
	names := e.getColNames(v)
	// keep a copy
	e.colNames = make([]string, len(names))
	_ = copy(e.colNames, names)
	return names, nil
}

// The private func where the work is done.  This also copies the headers
// to the Encoder.colNames field.
func (e *Encoder) getColNames(v interface{}) []string {
	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v)
	var cols []string
	for i := 0; i < typ.NumField(); i++ {
		// skip unexported
		tF := typ.Field(i)
		if len(tF.PkgPath) > 0 {
			continue
		}
		name := e.getFieldName(tF)
		if name == "" {
			continue
		}
		vF := val.Field(i)
		switch vF.Kind() {
		case reflect.Struct:
			tmp := e.getColNames(vF.Interface())
			cols = append(cols, tmp...)
			continue
		default:
			ok := supportedBaseKind(vF)
			if !ok {
				continue
			}
		}
		cols = append(cols, name)
	}
	return cols
}

// GetRow get's the data from the passed struct. This only operates on
// single structs.  If you wish to transmogrify everything at once, use
// Encoder.Marshal([]T).
func (e *Encoder) GetRow(v interface{}) ([]string, error) {
	if reflect.TypeOf(v).Kind() != reflect.Struct {
		return nil, StructRequiredError{reflect.TypeOf(v).Kind()}
	}
	// 2nd parm is only used for recursive calls.
	cols, _ := e.marshalStruct(v, false)
	return cols, nil
}

// Marshal takes a slice of structs and returns a [][]byte representing CSV
// data. Each field in the struct results in a column.  Fields that are slices
// are stored in a single column as a comma separated list of values.  Fields
// that are maps are stored in a single column as a comma separted list of
// key:value pairs.
//
// If the passed data isn't a slice of structs an error will be returned.
func (e *Encoder) Marshal(v interface{}) ([][]string, error) {
	// must be a slice
	if reflect.TypeOf(v).Kind() != reflect.Slice {
		return nil, StructSliceError{kind: reflect.TypeOf(v).Kind()}
	}
	val := reflect.ValueOf(v)
	// must be a slice of struct
	if val.IsNil() {
		return nil, ErrNilSlice
	}
	if val.Len() == 0 {
		return nil, ErrEmptySlice
	}
	var rows [][]string
	// get the first value in the slice to get the struct's field names
	s := val.Index(0)
	switch s.Kind() {
	case reflect.Struct:
		cols := e.getColNames(s.Interface())
		// keep a copy
		e.colNames = make([]string, len(cols))
		_ = copy(e.colNames, cols)
		// add as a row
		rows = append(rows, cols)
	default:
		return nil, StructSliceError{kind: reflect.Slice, sliceKind: s.Kind()}
	}
	// go through each element in the slice and marshal the element'd data.
	for i := 0; i < val.Len(); i++ {
		s := val.Index(i)
		row, ok := e.marshalStruct(s.Interface(), false)
		if !ok {
			continue
		}
		rows = append(rows, row)

	}
	return rows, nil
}

// marshal returns the marshaled value. If the received value is not of a
// supported Kind, a nil is returned along with false. For supported kinds, a
// slice of values is returned along with true.
func (e *Encoder) marshal(val reflect.Value, child bool) (cols []string, ok bool) {
	var s string
	switch val.Kind() {
	case reflect.Ptr:
		// for maps and slices, check that they are of supported types
		ok = supportedBaseKind(val)
		if !ok {
			return nil, false
		}
		vv := reflect.Indirect(val)
		switch vv.Kind() {
		case reflect.Invalid:
			// do nothing
		default:
			return e.marshal(vv, child)
		}
	case reflect.Struct:
		return e.marshalStruct(val.Interface(), true)
	case reflect.Map:
		s, ok = e.marshalMap(val, child)
		if !ok {
			return nil, false
		}
	case reflect.Array, reflect.Slice:
		s, ok = e.marshalSlice(val, child)
		if !ok {
			return nil, false
		}
	default:
		var ok bool
		s, ok = e.stringify(val, child)
		if !ok {
			return nil, false
		}
	}
	return append(cols, s), true
}

// marshal struct field data into a slice.
func (e *Encoder) marshalStruct(str interface{}, child bool) ([]string, bool) {
	var cols []string
	val := reflect.ValueOf(str)
	typ := reflect.TypeOf(str)
	for i := 0; i < typ.NumField(); i++ {
		tF := typ.Field(i)
		if len(tF.PkgPath) > 0 {
			continue
		}
		name := e.getFieldName(tF)
		if name == "" {
			continue
		}
		vF := val.Field(i)
		tmp, ok := e.marshal(vF, child)
		if !ok {
			// wasn't a supported kind, skip
			continue
		}
		cols = append(cols, tmp...)
	}
	return cols, true
}

// marshal map handles marshalling of maps.  Both the key and value types must
// be supported Kinds.
func (e *Encoder) marshalMap(m reflect.Value, child bool) (string, bool) {
	var ok bool
	if ok = supportedBaseKind(m); !ok {
		return "", false
	}
	// get the kind of the map value
	var row string
	var sv stringValues = m.MapKeys()
	// sort the map keys first
	// TODO have this work on things other than string?
	sort.Sort(sv)
	for i, key := range sv {
		val := m.MapIndex(key)
		kk, ok := e.marshal(key, true)
		if !ok {
			return "", false
		}
		var kval, vval string
		for j, tmp := range kk {
			if j > 0 && j < len(kk) {
				kval += ","
			}
			kval += tmp
		}
		if len(kk) > 1 {
			kval = fmt.Sprintf("%s%s%s", e.sepBeg, kval, e.sepEnd)
		}
		vv, ok := e.marshal(val, true)
		if !ok {
			return "", false
		}
		for j, tmp := range vv {
			if j > 0 && j < len(vv) {
				vval += ","
			}
			vval += tmp
		}
		if len(vv) > 1 {
			vval = fmt.Sprintf("%s%s%s", e.sepBeg, vval, e.sepEnd)
		}
		vval = fmt.Sprintf("%s:%s", kval, vval)
		if i == 0 {
			row = vval
		} else {
			row = fmt.Sprintf("%s,%s", row, vval)
		}
	}
	if child {
		row = fmt.Sprintf("%s%s%s", e.sepBeg, row, e.sepEnd)
	}
	return row, true
}

// marshalSlice handles marshaling of slices. This should not receive a
// pointer. Is is assumed that any pointers to the slice have already been
// dereferenced.
func (e *Encoder) marshalSlice(val reflect.Value, child bool) (string, bool) {
	var ok bool
	if ok = supportedBaseKind(val); !ok {
		return "", false
	}
	var sl, str string
	// check the type of slice and handle
	for j := 0; j < val.Len(); j++ {
		str = ""
		str, ok = e.stringify(val.Index(j), child)
		if !ok {
			return "", false
		}
		if j == 0 {
			sl = str
			continue
		}
		sl = fmt.Sprintf("%s,%s", sl, str)
	}
	if child {
		sl = fmt.Sprintf("%s%s%s", e.sepBeg, sl, e.sepEnd)
	}
	return sl, true
}

// stringify takes a interface and returns the value it contains as a string
// and true.  Composite types will first be marshaled.  If the received Kind is
// not supported, and empty string and false will be returned.
func (e *Encoder) stringify(v reflect.Value, child bool) (string, bool) {
	if !isSupportedKind(v.Kind()) {
		return "", false
	}
	switch v.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.Itoa(int(v.Int())), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(uint64(v.Uint()), e.base), true
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'E', -1, 32), true
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'E', -1, 64), true
	case reflect.Complex64, reflect.Complex128:
		return fmt.Sprintf("%g", v.Complex()), true
	case reflect.String:
		return v.String(), true
	case reflect.Ptr:
		return e.stringify(v.Elem(), child)
	default:
		cols, ok := e.marshal(v, true)
		if !ok {
			return "", false
		}
		r := cols[0]
		for i := 1; i < len(cols); i++ {
			r = fmt.Sprintf("%s,%s", r, cols[i])
		}
		if strings.HasPrefix(r, "(") {
			return r, true
		}
		return fmt.Sprintf("%s%s%s", e.sepBeg, r, e.sepEnd), true
	}
}

// ptrKind returns the Kind that it points to; if it's another pointer, this
// will recurse until it encounters a non-pointer kind.
func ptrKind(typ reflect.Type) reflect.Kind {
	if typ.Kind() == reflect.Ptr {
		return ptrKind(typ.Elem())
	}
	return typ.Kind()
}

// baseKind returns the baseKind of the type.
//
// For pointers, it's the Kind of the first non-pointer type.
// For slices, it's the Kind of the slice type, e.g. []int's kind is int.
// For maps, it's the Kind of the key and value of the map.  If the map
// consists of other maps, the key of the lowest level map will be returned.
//
// For non-map Kinds, v will be reflect.Invalid; which should be ignored.  For
// maps, v will be the Kind
func baseKind(typ reflect.Type) (k, v reflect.Kind) {
	switch typ.Kind() {
	case reflect.Ptr:
		return baseKind(typ.Elem())
	case reflect.Array, reflect.Slice:
		return baseKind(typ.Elem())
	case reflect.Map:
		return baseMapKind(typ)
	}
	return typ.Kind(), v
}

func supportedBaseKind(val reflect.Value) bool {
	k, v := baseKind(val.Type())
	if !isSupportedKind(k) {
		return false
	}
	// v is only used for maps; when the baseKind is a map, its value
	// will be something other than reflect.Invalid
	if v != reflect.Invalid {
		if !isSupportedKind(v) {
			return false
		}
	}
	return true
}

// sliceKind returns the Kind of the slice; e.g. reflect.Slice will be
//returned for [][]*int.
func sliceKind(val reflect.Value) reflect.Kind {
	switch val.Type().Elem().Kind() {
	case reflect.Ptr:
		return ptrKind(val.Type().Elem())
	}
	return val.Type().Elem().Kind()
}

// baseMapKind returns the the Kind of the key and value types of a map.  If
// the map consists of other maps, the Kind of the key of the lowest level map
// will be returned; e.g. reflect.String and reflect.Int will be returned for
// map[string]map[int]string.
func baseMapKind(typ reflect.Type) (k, v reflect.Kind) {
	if typ.Kind() == reflect.Ptr {
		return baseMapKind(typ.Elem())
	}
	k = typ.Key().Kind()
	if k == reflect.Ptr {
		k, _ = baseKind(typ.Key())
	}
	v, _ = baseKind(typ.Elem())
	return k, v
}

// returns whether or not the Kind is supported for usage in CSV.
func isSupportedKind(k reflect.Kind) bool {
	switch k {
	case reflect.Chan:
		return false
	case reflect.Func:
		return false
	case reflect.Interface:
		return false
	case reflect.Uintptr:
		return false
	case reflect.UnsafePointer:
		return false
	case reflect.Invalid:
		return false
	}
	return true
}

// getFieldName gets the field name.  If field tags are being used and the field
// is tagged with -, or skip this field, an empty string will be returned;
// which is a signal to skip this field.
func (e *Encoder) getFieldName(field reflect.StructField) string {
	if e.useTags {
		name := field.Tag.Get(e.tag)
		// skip columns tagged with -
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}
	return field.Name

}
