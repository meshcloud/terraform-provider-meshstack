package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/meshcloud/meshstack-cli/client"
)

const (
	SharedBuildingBlockRunnerUuid = "98520496-627d-43e6-82da-ce499179ff3f"
	defaultRunnerNotFound         = "Unable to find the default building block runner"
)

func defaultBuildingBlockRunnerUuid(ctx context.Context, runnerClient client.MeshBuildingBlockRunnerClient) (string, error) {
	runners, err := runnerClient.List(ctx, client.MeshBuildingBlockRunnerFilter{IsDefault: new(true)})
	if err != nil {
		return "", err
	}
	if len(runners) == 0 {
		return "", errors.New("meshStack reports no default building block runner")
	}
	var defaults []string
	for _, runner := range runners {
		if runner.Status == nil || runner.Status.IsDefault == nil {
			return SharedBuildingBlockRunnerUuid, nil
		}
		if *runner.Status.IsDefault && runner.Metadata.Uuid != nil {
			defaults = append(defaults, *runner.Metadata.Uuid)
		}
	}
	if len(defaults) != 1 {
		return "", fmt.Errorf("meshStack reports %d default building block runners %v, expected exactly one", len(defaults), defaults)
	}
	return defaults[0], nil
}

// subjectTemplateDescription documents the placeholder syntax of a runner's subject template. The
// placeholder set is a public contract for self-hosted runner operators, so the description spells
// it out in full; lead is the sentence that names the attribute.
func subjectTemplateDescription(lead string) string {
	return lead + ", for example `system:serviceaccount:my-namespace:workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}`. " +
		"Placeholders: `{{ workspaceIdentifier }}` (workspace owning the building block definition) and `{{ buildingBlockDefinitionUuid }}` (its `metadata.uuid`). " +
		"Without placeholders it is a fixed subject. meshStack fills it in per definition and reports the result as " +
		"`meshstack_building_block_definition.<name>.version_latest.workload_identity_federation.subject`."
}
