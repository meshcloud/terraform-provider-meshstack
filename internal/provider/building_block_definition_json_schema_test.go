package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// validateInputJsonSchemas only looks up "type" and "json_schema", so an input object carrying just
// those two is enough to drive it.
func inputObject(inputType string, jsonSchema attr.Value) types.Object {
	attrTypes := map[string]attr.Type{"type": types.StringType, "json_schema": types.StringType}

	return types.ObjectValueMust(attrTypes, map[string]attr.Value{
		"type":        types.StringValue(inputType),
		"json_schema": jsonSchema,
	})
}

func inputsMap(inputs map[string]types.Object) types.Map {
	elemType := types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "json_schema": types.StringType}}
	elems := make(map[string]attr.Value, len(inputs))
	for key, obj := range inputs {
		elems[key] = obj
	}

	return types.MapValueMust(elemType, elems)
}

func TestValidateInputJsonSchemas(t *testing.T) {
	t.Parallel()

	schema := types.StringValue(`{"type":"object"}`)

	for name, tc := range map[string]struct {
		inputs    types.Map
		wantError bool
	}{
		"json schema input with a schema": {
			inputs:    inputsMap(map[string]types.Object{"settings": inputObject("JSON_SCHEMA", schema)}),
			wantError: false,
		},
		"json schema input without a schema": {
			inputs:    inputsMap(map[string]types.Object{"settings": inputObject("JSON_SCHEMA", types.StringNull())}),
			wantError: true,
		},
		"json schema input whose schema is not known yet": {
			inputs:    inputsMap(map[string]types.Object{"settings": inputObject("JSON_SCHEMA", types.StringUnknown())}),
			wantError: false,
		},
		"other type carrying a schema": {
			inputs:    inputsMap(map[string]types.Object{"settings": inputObject("STRING", schema)}),
			wantError: true,
		},
		"other type without a schema": {
			inputs:    inputsMap(map[string]types.Object{"settings": inputObject("STRING", types.StringNull())}),
			wantError: false,
		},
		"no inputs declared": {
			inputs:    types.MapNull(types.StringType),
			wantError: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := &resource.ValidateConfigResponse{}
			validateInputJsonSchemas(tc.inputs, path.Root("version_spec").AtName("inputs"), resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantError {
				t.Fatalf("HasError() = %v, want %v (diagnostics: %v)", got, tc.wantError, resp.Diagnostics)
			}
		})
	}
}
