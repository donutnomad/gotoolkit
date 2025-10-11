package scopes8

import (
	gsql "github.com/donutnomad/gotoolkit/lib/gsql"
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
)

func Sort[Model any](field field.IField, asc ...bool) gsql.ScopeFuncG[Model] {
	return func(b *gsql.QueryBuilderG[Model]) {
		b.Order(field, asc...)
	}
}

//func Pluck[FieldType any, Field interface {
//	field.IFieldType[FieldType]
//	field.IField
//}, Model any](db gsql.IDB, query *gsql.QueryBuilderG[Model], field Field) ([]FieldType, error) {
//	var dest []FieldType
//	err := query.Clone().ClearSelects().Tx(db).Pluck(field.Column().Expr, &dest).Error
//	return dest, err
//}

func List[Model any](db gsql.IDB, query *gsql.QueryBuilderG[Model], paginate gsql.Paginate, scopes ...gsql.ScopeFuncG[Model]) ([]*Model, int64, error) {
	total, err := query.Count(db)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Paginate(paginate).Scopes(scopes...).Find(db)
	if err != nil {
		return nil, 0, err
	}
	return pos, total, nil
}

func ListMap[Model any, OUT any](db gsql.IDB, query *gsql.QueryBuilderG[Model], paginate gsql.Paginate, mapper func([]*Model) []*OUT, scopes ...gsql.ScopeFuncG[Model]) ([]*OUT, int64, error) {
	total, err := query.Count(db)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Paginate(paginate).Scopes(scopes...).Find(db)
	if err != nil {
		return nil, 0, err
	}
	return mapper(pos), total, nil
}

func ListAndMap[Model any, OUT any](db gsql.IDB, query *gsql.QueryBuilderG[Model], mapper func([]*Model) []*OUT, scopes ...gsql.ScopeFuncG[Model]) ([]*OUT, int64, error) {
	total, err := query.Count(db)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Scopes(scopes...).Find(db)
	if err != nil {
		return nil, 0, err
	}
	return mapper(pos), total, nil
}
