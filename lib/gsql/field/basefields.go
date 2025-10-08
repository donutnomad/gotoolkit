package field

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/samber/lo"
	"gorm.io/gorm/clause"
)

type BaseFields []IField

// SetAliasWith 设置别名
func (b BaseFields) SetAliasWith(fn func(fieldName string) string) BaseFields {
	var ret BaseFields
	for _, base := range b {
		switch v := base.(type) {
		case Base:
			if v.IsExpr() {
				panic("[BaseFields.WithPrefix] cannot operate on expr Field")
			}
			clone := v
			clone.alias = fn(clone.columnName)
			ret = append(ret, clone)
		case *Base:
			if v.IsExpr() {
				panic("[BaseFields.WithPrefix] cannot operate on expr Field")
			}
			clone := &Base{
				tableName:  v.tableName,
				columnName: v.columnName,
				alias:      v.alias,
				sql:        v.sql,
				flags:      v.flags,
			}
			clone.alias = fn(clone.columnName)
			ret = append(ret, clone)
		default:
			val := reflect.ValueOf(base)
			withAlias, ok := val.Type().MethodByName("WithAlias")
			if !ok {
				panic(fmt.Sprintf("[BaseFields.WithPrefix] no WithAlias method found for %T", base))
			}
			oldFieldName := base.Name()
			newFieldName := fn(oldFieldName)
			rets := withAlias.Func.Call([]reflect.Value{val, reflect.ValueOf(newFieldName)})
			ret = append(ret, rets[0].Interface().(IField))
		}
	}
	return ret
}

func (b BaseFields) As(prefix string) IField {
	return b.AsWith(prefix)
}

func (b BaseFields) AsWith(prefix string, suffix ...string) BaseFields {
	return b.SetAliasWith(func(fieldName string) string {
		if len(suffix) > 0 {
			return prefix + fieldName + suffix[0]
		}
		return prefix + fieldName
	})
}

func (b BaseFields) Exclude(f ...IField) BaseFields {
	var names = lo.Map(f, func(item IField, index int) string {
		return item.Name()
	})
	var ret BaseFields
	for _, base := range b {
		if !slices.Contains(names, base.Name()) {
			ret = append(ret, base)
		}
	}
	return ret
}

func (b BaseFields) Include(f ...IField) BaseFields {
	var names = lo.Map(f, func(item IField, index int) string {
		return item.Name()
	})
	var ret BaseFields
	for _, base := range b {
		if slices.Contains(names, base.Name()) {
			ret = append(ret, base)
		}
	}
	return ret
}

func (b BaseFields) ToExpr() Expression {
	panic("BaseFields cannot ToExpr")
}

func (b BaseFields) ToColumn() clause.Column {
	panic("BaseFields cannot ToColumn")
}

func (b BaseFields) Name() string {
	panic("BaseFields cannot get Name")
}

func (b BaseFields) FullName() string {
	panic("BaseFields cannot get FullName")
}

func (b BaseFields) IsExpr() bool {
	return true
}

func (b BaseFields) Alias() string {
	panic("BaseFields cannot get Alias")
}
