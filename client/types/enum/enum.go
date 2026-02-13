package enum

import (
	"fmt"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
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

func (e Enum[T]) Markdown() string {
	return strings.Join(e.to(Entry[T].Markdown), ", ")
}

type Entry[T ~string] string

func (ee Entry[T]) Ptr() *T {
	return ptr.To(ee.Unwrap())
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
