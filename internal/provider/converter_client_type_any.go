package provider

import (
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

// Converters for clientTypes.Any fields, shared by any resource or data source that exposes an
// unstructured value. A clientTypes.Any holds an arbitrary decoded JSON value (string, number, bool,
// object, array, null); these converters marshal it to/from a JSON string so it round-trips through a
// Terraform jsontypes.Normalized attribute. JSON-encoding preserves type information (numbers stay
// numbers, objects keep their structure) that a plain string attribute would lose; callers read the
// value back with jsondecode(...).

// withValueFromConverterForClientTypeAny converts a clientTypes.Any (Go) into its JSON-string Terraform
// representation during generic.Set.
func withValueFromConverterForClientTypeAny() generic.ConverterOption {
	clientTypeAny := reflect.TypeFor[clientTypes.Any]()
	return generic.WithValueFromConverter(func(attributePath path.Path, in reflect.Value, haveNil, haveUnknown bool) (out tftypes.Value, matched bool, err error) {
		if in.Type() == clientTypeAny {
			matched = true
			var marshalled []byte
			marshalled, err = json.Marshal(in.Interface())
			if err != nil {
				return
			}
			out, err = generic.ValueFrom(string(marshalled))
		}
		return
	})
}

// withValueToConverterForClientTypeAny is the Terraform->Go counterpart of
// withValueFromConverterForClientTypeAny, needed wherever a model is read back via generic.Get (e.g. a
// resource reading its own state) so the clientTypes.Any value can be converted in this direction too —
// without it the generic walker panics on the unsupported interface kind.
//
// It matches the field type *exactly* (mirroring the From converter's in.Type() == clientTypeAny guard)
// rather than via generic.WithValueToConverterFor, because clientTypes.Any is an interface that every
// type is AssignableTo — a WithValueToConverterFor[clientTypes.Any] would greedily match the root model.
func withValueToConverterForClientTypeAny() generic.ConverterOption {
	clientTypeAny := reflect.TypeFor[clientTypes.Any]()
	return generic.WithValueToConverter(func(_ path.Path, in tftypes.Value, out reflect.Value) (matched bool, err error) {
		if out.Type() != clientTypeAny {
			return // matched=false
		}
		matched = true
		if in.IsKnown() && !in.IsNull() {
			var jsonValue string
			if err = in.As(&jsonValue); err != nil {
				return
			}
			var v clientTypes.Any
			if err = json.Unmarshal([]byte(jsonValue), &v); err != nil {
				return
			}
			if v != nil {
				out.Set(reflect.ValueOf(v))
			}
		}
		return
	})
}
