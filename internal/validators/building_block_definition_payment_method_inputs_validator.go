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

var _ validator.Map = BuildingBlockDefinitionPaymentMethodInputs{}

// BuildingBlockDefinitionPaymentMethodInputs repeats meshStack's rules for a Payment Method input at plan
// time, for the same reason as BuildingBlockDefinitionTagInputs.
//
// Whether a value names a Payment Method of the ordering workspace is checked on the building block, so
// that one is left to meshStack.
type BuildingBlockDefinitionPaymentMethodInputs struct{}

var paymentMethodInputForbiddenAttributes = map[string]string{
	"default_value":                  "a Payment Method belongs to one workspace, while the definition is offered to many",
	"sensitive":                      "a Payment Method identifier is no secret, and meshPanel could not preselect a hidden value",
	"selectable_values":              "meshStack offers the Payment Methods of the ordering workspace",
	"value_validation_regex":         "meshStack checks the value against the Payment Methods of the ordering workspace",
	"validation_regex_error_message": "meshStack checks the value against the Payment Methods of the ordering workspace",
}

func (v BuildingBlockDefinitionPaymentMethodInputs) Description(_ context.Context) string {
	return fmt.Sprintf("Ensures every %s input follows the rules meshStack sets for it",
		client.MeshBuildingBlockInputAssignmentTypePaymentMethod)
}

func (v BuildingBlockDefinitionPaymentMethodInputs) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v BuildingBlockDefinitionPaymentMethodInputs) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var targetType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("target_type"), &targetType)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// target_type defaults after validation, so a null value here is the schema default.
	configuredTargetType, targetTypeKnown := configuredString(targetType, client.MeshBuildingBlockTypeWorkspaceLevel.String())

	for key, element := range req.ConfigValue.Elements() {
		input, ok := element.(types.Object)
		if !ok || input.IsNull() || input.IsUnknown() {
			continue
		}
		attributes := input.Attributes()
		assignmentType, assignmentTypeKnown := configuredString(attributes["assignment_type"], "")
		if !assignmentTypeKnown || assignmentType != client.MeshBuildingBlockInputAssignmentTypePaymentMethod.String() {
			continue
		}
		inputPath := req.Path.AtMapKey(key)

		if targetTypeKnown && configuredTargetType != client.MeshBuildingBlockTypeWorkspaceLevel.String() {
			resp.Diagnostics.AddAttributeError(inputPath.AtName("assignment_type"),
				"A Payment Method input needs a workspace building block",
				fmt.Sprintf("Input %q reads a Payment Method, which meshStack offers only on a %s building block, not on a %s one.",
					key, client.MeshBuildingBlockTypeWorkspaceLevel, configuredTargetType))
		}

		inputType, inputTypeKnown := configuredString(attributes["type"], "")
		if inputTypeKnown && inputType != "" && inputType != client.MeshBuildingBlockIOTypeCode.String() {
			resp.Diagnostics.AddAttributeError(inputPath.AtName("type"),
				"A Payment Method input must be a code input",
				fmt.Sprintf("The run receives the Payment Method as a JSON object, so input %q must set type to %s, not %s.",
					key, client.MeshBuildingBlockIOTypeCode, inputType))
		}

		for attributeName, reason := range paymentMethodInputForbiddenAttributes {
			if isSet(attributes[attributeName]) {
				resp.Diagnostics.AddAttributeError(inputPath.AtName(attributeName),
					"A Payment Method input cannot set "+attributeName,
					fmt.Sprintf("Remove %s from input %q: %s.", attributeName, key, reason))
			}
		}
	}
}

// isSet is false for an unknown value too, because it may still turn out null.
func isSet(value attr.Value) bool {
	return value != nil && !value.IsNull() && !value.IsUnknown()
}
