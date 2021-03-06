package marshal

import (
	"reflect"
)

// UpdateNilFieldsWithZeroValue checks the fields with tag
// `default_zero_value:"true"` and updates the fields with zero value if they are nil.
// This function will walk through the struct recursively,
// if the tagged fields of struct have duplicated type in the same path,
// the function may cause infinite loop.
// Before calling this function, please make sure the struct get pass with
// function `shouldNotHaveDuplicatedTypeInSamePath` in the test case.
func UpdateNilFieldsWithZeroValue(i interface{}) {
	t := reflect.TypeOf(i).Elem()
	v := reflect.ValueOf(i).Elem()

	if t.Kind() != reflect.Struct {
		return
	}
	numField := t.NumField()
	for i := 0; i < numField; i++ {
		zerovalueTag := t.Field(i).Tag.Get("default_zero_value")
		if zerovalueTag != "true" {
			continue
		}

		field := v.Field(i)
		ft := t.Field(i)
		if field.Kind() == reflect.Ptr {
			ele := field.Elem()
			if !ele.IsValid() {
				ele = reflect.New(ft.Type.Elem())
				field.Set(ele)
			}
			UpdateNilFieldsWithZeroValue(field.Interface())
		}
	}
}

func ShouldNotHaveDuplicatedTypeInSamePath(i interface{}, pathSet map[string]interface{}) bool {
	t := reflect.TypeOf(i).Elem()
	v := reflect.ValueOf(i).Elem()

	if t.Kind() != reflect.Struct {
		return true
	}
	numField := t.NumField()
	for i := 0; i < numField; i++ {
		zerovalueTag := t.Field(i).Tag.Get("default_zero_value")
		if zerovalueTag != "true" {
			continue
		}

		field := v.Field(i)
		ft := t.Field(i)
		if field.Kind() == reflect.Ptr {
			ele := field.Elem()
			if !ele.IsValid() {
				ele = reflect.New(ft.Type.Elem())
				field.Set(ele)
			}
			typeName := ft.Type.String()
			if _, ok := pathSet[typeName]; ok {
				return false
			}
			newSet := copySet(pathSet)
			newSet[ft.Type.String()] = struct{}{}
			pass := ShouldNotHaveDuplicatedTypeInSamePath(field.Interface(), newSet)
			if !pass {
				return false
			}
		}
	}

	return true
}

func copySet(input map[string]interface{}) map[string]interface{} {
	output := map[string]interface{}{}
	for k := range input {
		output[k] = input[k]
	}

	return output
}
