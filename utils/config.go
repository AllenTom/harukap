package util

import (
	"github.com/spf13/viper"
	"reflect"
)

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && isZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && isZero(v.Field(i))
		}
		return z
	}
	// Compare other types directly:
	z := reflect.Zero(v.Type())
	return v.Interface() == z.Interface()
}
func SetDefaultWithKeys(viper *viper.Viper, target string, aliasKeys ...string) bool {
	for _, aliasKey := range aliasKeys {
		value := viper.Get(aliasKey)
		if isZero(reflect.ValueOf(value)) {
			continue
		} else {
			viper.SetDefault(target, value)
			return true
		}
	}
	return false
}
