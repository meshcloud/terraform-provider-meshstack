package provider

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

const buildingBlockInputUpdateableByConsumerDescription = "Whether a consumer can change the input after the building block " +
	"is created, as the building block definition version declares it. Null when meshStack does not report it."

const operatorPermissions = "admin or platform operator permissions (`ADM_BUILDINGBLOCK_SAVE`, or " +
	"`MANAGED_BUILDINGBLOCK_SAVE` for the definition's owning workspace)"

const replaceBuildingBlockHint = "`tofu apply -replace=<resource address>` (or `terraform apply -replace=...`) replaces " +
	"the building block on the next apply: it destroys the building block and creates it again with the new inputs."

func buildingBlockInputPath(key string) path.Path {
	return path.Root("spec").AtName("inputs").AtMapKey(key)
}

// isNonUpdateableUserInput is false when meshStack does not report the flag, so an older meshStack keeps the in-place update.
func (in buildingBlockAllInput) isNonUpdateableUserInput() bool {
	return in.AssignmentType == client.MeshBuildingBlockInputAssignmentTypeUserInput &&
		in.UpdateableByConsumer != nil && !*in.UpdateableByConsumer
}

// changedNonUpdateableUserInputs compares against all_inputs, the values meshStack itself compares a change with.
func changedNonUpdateableUserInputs(
	ctx context.Context,
	plan, state generic.AttributeGetter,
	planned map[string]*client.MeshBuildingBlockInput,
	current map[string]buildingBlockAllInput,
	diags *diag.Diagnostics,
) (keys []string) {
	for key, plannedInput := range planned {
		currentInput, known := current[key]
		if !known || !currentInput.isNonUpdateableUserInput() {
			continue
		}
		changed := false
		if plannedInput.IsSensitive {
			changed = secret.VersionChanged(ctx, plan, state, buildingBlockInputPath(key).AtName("sensitive"), diags)
		} else {
			var currentValue any
			if currentInput.Value != nil {
				if err := json.Unmarshal([]byte(*currentInput.Value), &currentValue); err != nil {
					diags.AddError("Unable to compare the planned inputs with the building block", fmt.Sprintf("input %q: %v", key, err))
					return nil
				}
			}
			changed = !reflect.DeepEqual(plannedInput.Value.Y, currentValue)
		}
		if changed {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func nonUpdateableUserInputChangeRejected(inputs []string, underlying error) (summary, detail string) {
	quoted := make([]string, len(inputs))
	for i, key := range inputs {
		quoted[i] = fmt.Sprintf("%q", key)
	}
	return "Building block input is not updateable by consumers",
		fmt.Sprintf("meshStack rejected the update with a 403. The building block definition does not let consumers change "+
			"these user inputs after the building block is created: %s. Only an API key with %s may change them in place.\n\n"+
			"Choose one:\n"+
			"- Ask the platform team that owns the building block definition to make these inputs updateable by consumers, "+
			"then apply again.\n"+
			"- Replace the building block: %s\n\n"+
			"Underlying error: %s",
			strings.Join(quoted, ", "), operatorPermissions, replaceBuildingBlockHint, underlying.Error())
}
