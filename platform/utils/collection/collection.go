package collection

import (
	"fmt"
	"reflect"
	"unicode"
)

func Contains[T comparable](value T, array []T) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}

	return false
}

func ContainsWithMatcher[T comparable, B any](value T, array []B, matcher func(value T, arrayValue B) bool) bool {
	for i := 0; i < len(array); i++ {
		if matcher(value, array[i]) {
			return true
		}
	}

	return false
}

// StructEqual checks if the given matching exported columns of two struct types are equal
func StructEqual(a, b any, fields []string) error {
	if reflect.ValueOf(a).Kind() != reflect.Struct || reflect.ValueOf(b).Kind() != reflect.Struct {
		return fmt.Errorf("a and/or b is/are not struct type/s")
	}

	for _, v := range fields {
		if !unicode.IsUpper(rune(v[0])) {
			return fmt.Errorf("%s is unexported field", v)
		}

		fa := reflect.ValueOf(a).FieldByName(v)
		if !fa.IsValid() {
			return fmt.Errorf("a has no field named %s", v)
		}

		fb := reflect.ValueOf(b).FieldByName(v)
		if !fb.IsValid() {
			return fmt.Errorf("b has no field named %s", v)
		}

		if !reflect.DeepEqual(fa.Interface(), fb.Interface()) {
			return fmt.Errorf("a.%[1]s != b.%[1]s, (a.%[1]s=%[2]v, b.%[1]s=%[3]v)", v, fa.Interface(), fb.Interface())
		}
	}

	return nil
}
