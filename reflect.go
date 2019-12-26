package log

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

// retirm s;oce as a stromg
func sliceAsString(data []interface{}) string {
	if nil == data {
		return ""
	}
	parts := make([]string, len(data))
	for i, obj := range data {
		parts[i] = interfaceAsString(obj)
	}
	return strings.Join(parts, ", ")
}

// returns interface as a string
func interfaceAsString(obj interface{}) string {
	if false == isInterfacesNil(obj) {
		switch val := obj.(type) {
		case error:
			return val.Error()
		case *time.Duration:
			return _f("%.02fs", val.Seconds())
		case time.Duration:
			return _f("%.02fs", val.Seconds())
		case time.Time:
			return val.Format(time.RFC1123)
		case string:
			return val
		case bool:
			if true == val {
				return "true"
			}
			return "false"
		case int:
			return strconv.FormatInt(int64(val), 10)
		case int64:
			return strconv.FormatInt(val, 10)
		case uint64:
			return strconv.FormatUint(val, 10)
		case uint:
			return strconv.FormatUint(uint64(val), 10)
		case *Tag:
			return _f("[%s: %s]", val.name, interfaceAsString(val.value))
		default:
			t, v := reflectDeref(obj)
			name := strings.TrimPrefix(t.String(), "main.")
			if reflect.Struct == t.Kind() {
				return _f("[%s: %s]", name, structFieldsAsString(t, v))
			} else if reflect.Slice == t.Kind() {
				str := make([]string, v.Len())
				for i := 0; i < v.Len(); i++ {
					str[i] = _f("%d: %s", i, interfaceAsString(v.Index(i).Interface()))
				}
				return _f("[%s: %s]", name, strings.Join(str, ", "))
			} else {
				return _f("[%s: %v]", name, obj)
			}
		}
	}
	return "nil"
}

// defers a point type value in reflection to get to pointed to item
func reflectDeref(obj interface{}) (reflect.Type, reflect.Value) {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	if reflect.Ptr == t.Kind() {
		t = t.Elem()
		v = v.Elem()
	}
	return t, v
}

// decides if an interface is nil
func isInterfacesNil(o interface{}) bool {
	if nil == o || (reflect.Ptr == reflect.TypeOf(o).Kind() && reflect.ValueOf(o).IsNil()) {
		return true
	}
	return false
}

// decides if object is a ptr and is nil
func isNil(v reflect.Value) bool {
	if reflect.Ptr == reflect.TypeOf(v).Kind() && reflect.ValueOf(v).IsNil() {
		return true
	}
	return false
}

// returns struct field as a string
func structFieldsAsString(t reflect.Type, v reflect.Value) string {
	var str []string
	if false == isNil(v) {
		for i := 0; i < t.NumField(); i++ {
			name, found := t.Field(i).Tag.Lookup("log")
			if true == found {
				f := v.Field(i)
				if isNil(f) {
					str = append(str, _f("%s: (nil)", name))
				} else {
					str = append(str, _f("%s: %s", name, interfaceAsString(f.Interface())))
				}
			}
		}
	}
	if nil == str {
		return _f("%v", v)
	}
	return strings.Join(str, ", ")
}
