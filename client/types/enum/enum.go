package enum

import (
	"fmt"
	"slices"
	"strings"
)

func Of[T ~string](entries ...Entry[T]) Enum[T] {
	return entries
}

type Enum[T ~string] []Entry[T]

func (e *Enum[T]) Entry(v string) (ee Entry[T]) {
	ee = Entry[T](v)
	*e = append(*e, ee)
	return
}

func (e Enum[T]) to(mapper func(entry Entry[T]) string) (result []string) {
	for _, ee := range e {
		result = append(result, mapper(ee))
	}
	return
}

func (e Enum[T]) Strings() []string {
	return e.to(Entry[T].String)
}

// Except returns the enum minus the given entries, preserving order. Deriving a subset this way keeps a
// single source of truth: adding an entry to the base enum flows into the subset automatically.
func (e Enum[T]) Except(excluded ...Entry[T]) Enum[T] {
	return slices.DeleteFunc(slices.Clone(e), func(ee Entry[T]) bool {
		return slices.Contains(excluded, ee)
	})
}

func (e Enum[T]) Markdown() string {
	return strings.Join(e.to(Entry[T].Markdown), ", ")
}

type Entry[T ~string] string

func (ee Entry[T]) Ptr() *T {
	return new(ee.Unwrap())
}

func (ee Entry[T]) Unwrap() T {
	return T(ee)
}

func (ee Entry[T]) String() string {
	return string(ee)
}

func (ee Entry[T]) Markdown() string {
	return fmt.Sprintf("`%s`", ee)
}
