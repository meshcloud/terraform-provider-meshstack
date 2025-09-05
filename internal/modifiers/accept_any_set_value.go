package modifiers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

func AcceptAnySetValue() planmodifier.Set {
	return acceptAnySetValueModifier{}
}

// useStateForUnknownModifier implements the plan modifier.
type acceptAnySetValueModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m acceptAnySetValueModifier) Description(_ context.Context) string {
	return "todo"
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m acceptAnySetValueModifier) MarkdownDescription(_ context.Context) string {
	return "todo"
}

func (m acceptAnySetValueModifier) PlanModifySet(_ context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	resp.Diagnostics.AddWarning("req.PlanValue req.ConfigValue resp.PlanValue", fmt.Sprintf("%v %v %v", req.PlanValue, req.ConfigValue, resp.PlanValue))
}
