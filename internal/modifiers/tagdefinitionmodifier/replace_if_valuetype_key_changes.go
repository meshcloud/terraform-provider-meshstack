package tagdefinitionmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

func ReplaceIfValueTypeKeyChanges(ctx context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
	planValue := req.PlanValue
	stateValue := req.StateValue

	for k, v := range planValue.Attributes() {
		if v.IsNull() && !stateValue.Attributes()[k].IsNull() {
			resp.RequiresReplace = true
			return
		}
	}
}
