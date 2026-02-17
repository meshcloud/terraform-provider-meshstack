package generic

import (
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/path"

	reflectwalk "github.com/meshcloud/terraform-provider-meshstack/internal/util/reflect"
)

// WithAttributePath sets a start path for conversion.
// Useful when calling ValueTo/ValueFrom from converters itself.
func WithAttributePath(attributePath path.Path) ConverterOption {
	return func(c *converter) {
		c.AttributePath = attributePath.Copy()
	}
}

type ConverterOption func(*converter)

type ConverterOptions []ConverterOption

func (opts ConverterOptions) Append(options ...ConverterOption) ConverterOptions {
	return append(opts, options...)
}

func (opts ConverterOptions) newConverter() (conv converter) {
	for _, opt := range opts {
		opt(&conv)
	}
	return
}

type converter struct {
	AttributePath           path.Path
	ValueToConverters       []ValueToConverter
	ValueFromConverters     []ValueFromConverter
	ValueFromEmptyContainer ValueFromEmptyContainerHandler
	SetUnknownValueToZero   bool
	SetElemTypes            []reflect.Type
}

type ValueFromEmptyContainerHandler func(attributePath path.Path) (haveNil bool, err error)

func (conv converter) walkPathToAttributePath(walkPath reflectwalk.WalkPath) (result path.Path) {
	result = conv.AttributePath
	for _, step := range walkPath {
		switch s := step.(type) {
		case reflectwalk.StructField:
			result = result.AtName(s.Tag.Get("tfsdk"))
		case reflectwalk.MapKey:
			if !s.EmptyContainer() {
				result = result.AtMapKey(s.Name())
			} else {
				result = result.AtMapKey("<empty map>")
			}
		case reflectwalk.SliceIndex:
			if !s.EmptyContainer() {
				result = result.AtListIndex(int(s))
			} else {
				result = result.AtListIndex(-1)
			}
		default:
			// do nothing
		}
	}
	return
}
