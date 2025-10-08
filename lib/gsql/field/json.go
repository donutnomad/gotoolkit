package field

import (
	"fmt"

	"gorm.io/datatypes"
)

type JsonField[T any] struct {
	*datatypes.JSONQueryExpression
}

func NewJsonField[T any](tableName, name string) JsonField[T] {
	return JsonField[T]{
		JSONQueryExpression: datatypes.JSONQuery(fieldName(tableName, name)),
	}
}

func fieldName(tableName, name string) string {
	if tableName == "" {
		return fmt.Sprintf("`%s`", name)
	}
	return fmt.Sprintf("`%s`.`%s`", tableName, name)
}
