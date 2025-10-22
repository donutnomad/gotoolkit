package scopes8

import (
	"github.com/donutnomad/gotoolkit/lib/gsql"
)

// List
// Deprecated: use scopes.List instead
func List[Model any](db gsql.IDB, query *gsql.QueryBuilderG[Model], paginate gsql.Paginate, scopes ...gsql.ScopeFuncG[Model]) ([]*Model, int64, error) {
	total, err := query.Count(db)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Paginate(paginate).ScopeG(scopes...).Find(db)
	if err != nil {
		return nil, 0, err
	}
	return pos, total, nil
}

// ListMap
// Deprecated: use scopes.ListMap instead
func ListMap[Model any, OUT any](db gsql.IDB, query *gsql.QueryBuilderG[Model], paginate gsql.Paginate, mapper func([]*Model) []*OUT, scopes ...gsql.ScopeFunc) ([]*OUT, int64, error) {
	total, err := query.Count(db)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Paginate(paginate).Scope(scopes...).Find(db)
	if err != nil {
		return nil, 0, err
	}
	return mapper(pos), total, nil
}

// ListAndMap
// Deprecated: use scopes.ListAndMap instead
func ListAndMap[Model any, OUT any](db gsql.IDB, query *gsql.QueryBuilderG[Model], mapper func([]*Model) []*OUT, scopes ...gsql.ScopeFunc) ([]*OUT, int64, error) {
	total, err := query.Count(db)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Scope(scopes...).Find(db)
	if err != nil {
		return nil, 0, err
	}
	return mapper(pos), total, nil
}
