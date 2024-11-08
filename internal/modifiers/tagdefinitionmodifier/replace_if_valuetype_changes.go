package tagdefinitionmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// ReplaceIfValueTypeChanges is a plan modifier that requires a replace if value type changes, but not when .
// e.g.:
//
//	if <spec.value_type.string> changes to <spec.value_type.integer> -> replace
//	if <spec.value_type.string.default_value> changes -> don't replace
func ReplaceIfValueTypeChanges(ctx context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
	planValue := req.PlanValue
	stateValue := req.StateValue

	// Loops over value_type attributes and checks if they are already set in the state.
	// If a value type is set in the state, and plan is to set its value to null (i.e. implies that <value_type.key> is modified, or removed)
	// then it requires a replace.
	for k, v := range planValue.Attributes() {
		if v.IsNull() && !stateValue.Attributes()[k].IsNull() {
			resp.RequiresReplace = true
			return
		}
	}
}
