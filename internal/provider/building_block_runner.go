package provider

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
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
		return "", errors.New("meshStack must have one default building block runner, but it returned none")
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
		return "", fmt.Errorf("meshStack must have one default building block runner, but it returned %d: %v", len(defaults), defaults)
	}
	return defaults[0], nil
}

func runnersShareKey(ctx context.Context, runnerClient client.MeshBuildingBlockRunnerClient, uuid, otherUuid string) bool {
	if runnerClient == nil {
		return false
	}
	runner, err := runnerClient.Read(ctx, uuid)
	if err != nil || runner == nil {
		return false
	}
	other, err := runnerClient.Read(ctx, otherUuid)
	if err != nil || other == nil {
		return false
	}
	return samePublicKey(runner.Spec.PublicKey, other.Spec.PublicKey)
}

func samePublicKey(key, otherKey string) bool {
	if key == otherKey {
		return true
	}
	parsed, ok := parsePublicKey(key)
	if !ok {
		return false
	}
	otherParsed, ok := parsePublicKey(otherKey)
	if !ok {
		return false
	}
	equatable, ok := parsed.(interface{ Equal(crypto.PublicKey) bool })
	return ok && equatable.Equal(otherParsed)
}

func parsePublicKey(keyOrCertificate string) (crypto.PublicKey, bool) {
	block, _ := pem.Decode([]byte(keyOrCertificate))
	if block == nil {
		return nil, false
	}
	if certificate, err := x509.ParseCertificate(block.Bytes); err == nil {
		return certificate.PublicKey, true
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	return key, err == nil
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
