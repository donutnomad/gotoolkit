package utils

import "reflect"

func IsNeedBracket(vars []any) bool {
	for _, v := range vars {
		switch v := v.(type) {
		case []any:
			if len(v) > 0 {
				return false
			}
		default:
			switch rv := reflect.ValueOf(v); rv.Kind() {
			case reflect.Slice, reflect.Array:
				if rv.Len() == 0 {
				} else if rv.Type().Elem() == reflect.TypeOf(uint8(0)) {
				} else {
					return false
				}
			default:
			}
		}
	}
	return true
}
