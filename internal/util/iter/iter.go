package iter

import (
	"iter"
	"slices"
)

func PickFirst[V1, V2 any](seq2 iter.Seq2[V1, V2]) iter.Seq[V1] {
	return func(yield func(V1) bool) {
		for v1 := range seq2 {
			if !yield(v1) {
				return
			}
		}
	}
}

func Map[V, U any](seq iter.Seq[V], mapper func(V) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		for t := range seq {
			if !yield(mapper(t)) {
				return
			}
		}
	}
}

func MapAndSortBy[T any, Comparable interface{ Compare(Comparable) int }](mapper func(T) Comparable, seq iter.Seq[T]) []Comparable {
	return slices.SortedStableFunc(Map(seq, mapper), Comparable.Compare)
}
