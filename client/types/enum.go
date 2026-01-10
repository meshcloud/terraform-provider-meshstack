package types

import (
	"fmt"
	"strings"
)

type Enum[T ~string] []EnumEntry[T]

func (e *Enum[T]) Entry(v string) (ee EnumEntry[T]) {
	ee = EnumEntry[T](v)
	*e = append(*e, ee)
	return
}

func (e Enum[T]) to(mapper func(entry EnumEntry[T]) string) (result []string) {
	for _, ee := range e {
		result = append(result, mapper(ee))
	}
	return
}

func (e Enum[T]) Strings() []string {
	return e.to(EnumEntry[T].String)
}

func (e Enum[T]) Markdown() string {
	return strings.Join(e.to(EnumEntry[T].Markdown), ", ")
}

type EnumEntry[T ~string] string

func (ee EnumEntry[T]) Ptr() *T {
	return PtrTo[T](ee.Unwrap())
}

func (ee EnumEntry[T]) Unwrap() T {
	return T(ee)
}

func (ee EnumEntry[T]) String() string {
	return string(ee)
}

func (ee EnumEntry[T]) Markdown() string {
	return fmt.Sprintf("`%s`", ee)
}
