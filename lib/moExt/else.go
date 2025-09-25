package moExt

import (
	"github.com/samber/mo"
	"gorm.io/datatypes"
)

func Else[T any, O any](input mo.Option[T], mapper func(T) O) mo.Option[O] {
	if input.IsAbsent() {
		return mo.None[O]()
	}
	return mo.Some(mapper(input.MustGet()))
}

func ElseJsonType[T any](input mo.Option[T]) mo.Option[datatypes.JSONType[T]] {
	return Else(input, func(t T) datatypes.JSONType[T] {
		return datatypes.NewJSONType(t)
	})
}

func ElseJsonSlice[T any](input mo.Option[[]T]) mo.Option[datatypes.JSONSlice[T]] {
	return Else(input, func(t []T) datatypes.JSONSlice[T] {
		return t
	})
}

func ElseUnix[T interface {
	Unix() int64
}](input mo.Option[T]) mo.Option[int64] {
	return Else(input, func(t T) int64 {
		return t.Unix()
	})
}

func ElseUnixMilli[T interface {
	UnixMilli() int64
}](input mo.Option[T]) mo.Option[int64] {
	return Else(input, func(t T) int64 {
		return t.UnixMilli()
	})
}

func ElseUnixMicro[T interface {
	UnixMicro() int64
}](input mo.Option[T]) mo.Option[int64] {
	return Else(input, func(t T) int64 {
		return t.UnixMicro()
	})
}

func ElseUnixNano[T interface {
	UnixNano() int64
}](input mo.Option[T]) mo.Option[int64] {
	return Else(input, func(t T) int64 {
		return t.UnixNano()
	})
}
