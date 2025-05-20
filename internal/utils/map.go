package utils

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"iter"
	"sort"
)

func FilterMapEntries[K1 comparable, V1 any, K2 comparable, V2 any](m map[K1]V1, mapper func(k K1, v V1) (K2, V2, bool)) map[K2]V2 {
	var out = make(map[K2]V2)
	for k, v := range m {
		if k2, v2, ok := mapper(k, v); ok {
			out[k2] = v2
		}
	}
	return out
}

func CollectMap[K comparable, V any](it iter.Seq2[K, V]) map[K]V {
	var out = make(map[K]V)
	for k, v := range it {
		out[k] = v
	}
	return out
}

func IterSortMap[K constraints.Ordered, V any](m map[K]V) iter.Seq2[K, V] {
	keys := maps.Keys(m)
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return func(yield func(K, V) bool) {
		for _, key := range keys {
			if !yield(key, m[key]) {
				return
			}
		}
	}
}

func DefSlice[T any](items ...T) []T {
	return items
}
