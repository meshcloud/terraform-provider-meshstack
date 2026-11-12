package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
)

// inputsTargetType is the definition's spec.target_type; known is false while it is unknown.
type inputsTargetType struct {
	value string
	known bool
}

// forEachInputAssigned calls validate for every input whose assignment_type is known to be assignmentType.
func forEachInputAssigned(
	ctx context.Context,
	req validator.MapRequest,
	resp *validator.MapResponse,
	assignmentType fmt.Stringer,
	validate func(key string, inputPath path.Path, attributes map[string]attr.Value, target inputsTargetType),
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var targetType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("target_type"), &targetType)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// target_type defaults after validation, so a null value here is the schema default.
	var target inputsTargetType
	target.value, target.known = configuredString(targetType, client.MeshBuildingBlockTypeWorkspaceLevel.String())

	for key, element := range req.ConfigValue.Elements() {
		input, ok := element.(types.Object)
		if !ok || input.IsNull() || input.IsUnknown() {
			continue
		}
		attributes := input.Attributes()
		configured, known := configuredString(attributes["assignment_type"], "")
		if !known || configured != assignmentType.String() {
			continue
		}
		validate(key, req.Path.AtMapKey(key), attributes, target)
	}
}
