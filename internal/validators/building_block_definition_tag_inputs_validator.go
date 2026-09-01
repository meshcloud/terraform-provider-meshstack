package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var _ validator.Map = BuildingBlockDefinitionTagInputs{}

// BuildingBlockDefinitionTagInputs repeats meshStack's rules for a tag-sourced input at plan time, for
// the same reason as the two validators above: the resource writes the definition before the version
// that carries the inputs, so a backend rejection arrives after the definition already exists.
//
// It stops at what the configuration can prove. Whether the named tag exists in the tag schema takes a
// read against meshStack, so that one is left there.
type BuildingBlockDefinitionTagInputs struct{}

func (v BuildingBlockDefinitionTagInputs) Description(_ context.Context) string {
	return fmt.Sprintf("Ensures every %s input reads a tag the building block can resolve",
		client.MeshBuildingBlockInputAssignmentTypeTag)
}

func (v BuildingBlockDefinitionTagInputs) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v BuildingBlockDefinitionTagInputs) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
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
		if !assignmentTypeKnown || assignmentType != client.MeshBuildingBlockInputAssignmentTypeTag.String() {
			continue
		}
		inputPath := req.Path.AtMapKey(key)

		inputType, inputTypeKnown := configuredString(attributes["type"], "")
		if inputTypeKnown && inputType != "" && inputType != client.MeshBuildingBlockIOTypeCode.String() {
			resp.Diagnostics.AddAttributeError(inputPath.AtName("type"),
				"A tag input must be a code input",
				fmt.Sprintf("A meshStack tag value is a list of strings that arrives as a JSON array, so input %q must set type to %s, not %s.",
					key, client.MeshBuildingBlockIOTypeCode, inputType))
		}

		if sensitive, ok := attributes["sensitive"].(types.Object); ok && !sensitive.IsNull() && !sensitive.IsUnknown() {
			resp.Diagnostics.AddAttributeError(inputPath.AtName("sensitive"),
				"A tag input cannot be sensitive",
				fmt.Sprintf("Tags are visible metadata in meshStack, so encrypting the resolved value of input %q would hide nothing. Remove the sensitive block.", key))
		}

		if targetTypeKnown {
			validateTagInputArgument(key, inputPath, attributes["argument"], configuredTargetType, resp)
		}
	}
}

// validateTagInputArgument checks the `<target>.<tag key>` reference the argument carries, and that the
// building block can reach the named target at all.
func validateTagInputArgument(
	key string,
	inputPath path.Path,
	argument attr.Value,
	targetType string,
	resp *validator.MapResponse,
) {
	argumentPath := inputPath.AtName("argument")
	expectedFormat := fmt.Sprintf("Set it to `jsonencode(\"<target>.<tag key>\")`, for example `jsonencode(\"%s.costCenter\")`.",
		client.MeshBuildingBlockTagInputTargetWorkspace)

	encoded, ok := argument.(jsontypes.Normalized)
	if !ok || encoded.IsUnknown() {
		return
	}
	if encoded.IsNull() {
		resp.Diagnostics.AddAttributeError(argumentPath,
			"A tag input requires an argument naming the tag to read",
			fmt.Sprintf("Input %q has no argument. %s", key, expectedFormat))
		return
	}

	var reference string
	if err := json.Unmarshal([]byte(encoded.ValueString()), &reference); err != nil {
		resp.Diagnostics.AddAttributeError(argumentPath,
			"A tag input argument must be an encoded string",
			fmt.Sprintf("The argument of input %q is not a JSON string. %s", key, expectedFormat))
		return
	}

	target, tagKey, found := strings.Cut(reference, client.TagInputTargetSeparator)
	if !found || target == "" || tagKey == "" {
		resp.Diagnostics.AddAttributeError(argumentPath,
			"A tag input argument must name a target and a tag key",
			fmt.Sprintf("The argument of input %q is %q. %s", key, reference, expectedFormat))
		return
	}

	readableTargets := client.TagInputTargetsFor(client.MeshBuildingBlockType(targetType))
	if slices.Contains(readableTargets.Strings(), target) {
		return
	}
	if slices.Contains(client.MeshBuildingBlockTagInputTargets.Strings(), target) {
		resp.Diagnostics.AddAttributeError(argumentPath,
			"A tag input cannot read this target",
			fmt.Sprintf("Input %q reads a %s tag, which a %s building block has no relation to. It can read tags of: %s.",
				key, target, targetType, strings.Join(readableTargets.Strings(), ", ")))
		return
	}
	resp.Diagnostics.AddAttributeError(argumentPath,
		"A tag input argument must name a known target",
		fmt.Sprintf("The argument of input %q names the target %q, which is none of: %s.",
			key, target, strings.Join(client.MeshBuildingBlockTagInputTargets.Strings(), ", ")))
}
