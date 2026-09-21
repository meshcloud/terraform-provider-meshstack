package maps

import (
	"iter"
	"maps"
	"slices"
)

func SortedFunc[K comparable, V any](m map[K]V, cmp func(K, K) int) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range slices.SortedFunc(maps.Keys(m), cmp) {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
