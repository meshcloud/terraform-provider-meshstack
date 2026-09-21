package clientmock

import (
	"cmp"
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/google/uuid"
	"github.com/meshcloud/meshstack-cli/client"
)

type meshBuildingBlockDefinitionClient struct {
	Store        *Store[client.MeshBuildingBlockDefinition]
	StoreVersion *Store[client.MeshBuildingBlockDefinitionVersion]
	StoreRunner  *Store[client.MeshBuildingBlockRunner]
}

// wifPlaceholder matches the placeholder syntax of a runner's subject template.
var wifPlaceholder = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

// setStatus mirrors the backend's status section as far as the provider reads it: one entry per version,
// each with the workload identity meshStack resolves from the runner of that version.
func (m meshBuildingBlockDefinitionClient) setStatus(definition *client.MeshBuildingBlockDefinition) *client.MeshBuildingBlockDefinition {
	if definition.Status == nil {
		definition.Status = &client.MeshBuildingBlockDefinitionStatus{}
	}
	versions := m.versions(*definition.Metadata.Uuid)
	definition.Status.Versions = make([]client.MeshBuildingBlockDefinitionStatusVersion, len(versions))
	for i, version := range versions {
		definition.Status.Versions[i] = client.MeshBuildingBlockDefinitionStatusVersion{
			VersionUuid:                version.Metadata.Uuid,
			VersionNumber:              *version.Spec.VersionNumber,
			State:                      *version.Spec.State,
			WorkloadIdentityFederation: m.resolveWif(definition, version),
		}
	}
	if latest := len(versions) - 1; latest >= 0 {
		definition.Status.LatestVersion = *versions[latest].Spec.VersionNumber
		definition.Status.LatestVersionUuid = versions[latest].Metadata.Uuid
	}
	return definition
}

func (m meshBuildingBlockDefinitionClient) resolveWif(definition *client.MeshBuildingBlockDefinition, version *client.MeshBuildingBlockDefinitionVersion) *client.MeshBuildingBlockDefinitionWif {
	// A version that names no runner runs on the hosted one, as on a real meshStack.
	runnerUuid := SharedBuildingBlockRunnerUuid
	if version.Spec.RunnerRef != nil {
		runnerUuid = version.Spec.RunnerRef.Uuid
	}
	runner, ok := m.StoreRunner.Get(runnerUuid)
	if !ok || runner.Spec.WorkloadIdentityFederation == nil {
		return nil
	}

	wif := runner.Spec.WorkloadIdentityFederation
	template := ""
	if wif.SubjectTemplate != nil {
		template = *wif.SubjectTemplate
	}
	issuer := ""
	if wif.Issuer != nil {
		issuer = *wif.Issuer
	}
	resolved := wifPlaceholder.ReplaceAllStringFunc(template, func(placeholder string) string {
		switch wifPlaceholder.FindStringSubmatch(placeholder)[1] {
		case "workspaceIdentifier":
			return definition.Metadata.OwnedByWorkspace
		case "buildingBlockDefinitionUuid":
			return *definition.Metadata.Uuid
		default:
			return placeholder
		}
	})
	return &client.MeshBuildingBlockDefinitionWif{
		Issuer:  issuer,
		Subject: resolved,
		Gcp:     cloudWif(wif.Gcp),
		Aws:     cloudWif(wif.Aws),
		Azure:   cloudWif(wif.Azure),
	}
}

func cloudWif(runnerConfig *client.MeshRunnerWifProviderConfig) *client.MeshBuildingBlockDefinitionCloudWif {
	if runnerConfig == nil {
		return nil
	}
	return &client.MeshBuildingBlockDefinitionCloudWif{Audience: runnerConfig.Audience, TokenPath: runnerConfig.TokenPath}
}

// versions returns the definition's versions in ascending version number order.
func (m meshBuildingBlockDefinitionClient) versions(definitionUuid string) (versions []*client.MeshBuildingBlockDefinitionVersion) {
	for _, version := range m.StoreVersion.Values() {
		if version.Spec.BuildingBlockDefinitionRef != nil && version.Spec.BuildingBlockDefinitionRef.Uuid == definitionUuid {
			versions = append(versions, version)
		}
	}
	slices.SortFunc(versions, func(a, b *client.MeshBuildingBlockDefinitionVersion) int {
		return cmp.Compare(*a.Spec.VersionNumber, *b.Spec.VersionNumber)
	})
	return
}

func (m meshBuildingBlockDefinitionClient) List(_ context.Context, workspaceIdentifier *string) ([]client.MeshBuildingBlockDefinition, error) {
	var result []client.MeshBuildingBlockDefinition
	for _, def := range m.Store.Values() {
		if workspaceIdentifier == nil || def.Metadata.OwnedByWorkspace == *workspaceIdentifier {
			result = append(result, *m.setStatus(def))
		}
	}
	return result, nil
}

func (m meshBuildingBlockDefinitionClient) Read(_ context.Context, uuid string) (*client.MeshBuildingBlockDefinition, error) {
	if def, ok := m.Store.Get(uuid); ok {
		return m.setStatus(def), nil
	}
	return nil, nil
}

func (m meshBuildingBlockDefinitionClient) Create(_ context.Context, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	definitionUuid := uuid.NewString()
	definition.Metadata.Uuid = new(definitionUuid)
	if definition.Spec.Symbol == nil {
		definition.Spec.Symbol = new("mock-default-symbol")
	}
	m.Store.Set(definitionUuid, &definition)

	// Create initial empty version (as the backend does)
	versionUuid := uuid.NewString()
	m.StoreVersion.Set(versionUuid, &client.MeshBuildingBlockDefinitionVersion{
		Metadata: client.MeshBuildingBlockDefinitionVersionMetadata{
			Uuid:             versionUuid,
			OwnedByWorkspace: definition.Metadata.OwnedByWorkspace,
		},
		Spec: client.MeshBuildingBlockDefinitionVersionSpec{
			BuildingBlockDefinitionRef: &client.UuidRef{
				Uuid: definitionUuid,
				Kind: "meshBuildingBlockDefinition",
			},
			DeletionMode:  client.BuildingBlockDeletionModeDelete.Unwrap(),
			VersionNumber: new(int64(1)),
			State:         client.MeshBuildingBlockDefinitionVersionStateDraft.Ptr(),
		},
	})
	return m.setStatus(&definition), nil
}

func (m meshBuildingBlockDefinitionClient) Update(_ context.Context, uuid string, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	if existing, ok := m.Store.Get(uuid); ok {
		existing.Spec = definition.Spec
		existing.Metadata.Tags = definition.Metadata.Tags
		return m.setStatus(existing), nil
	}
	return nil, fmt.Errorf("building block definition not found: %s", uuid)
}

func (m meshBuildingBlockDefinitionClient) Delete(_ context.Context, uuid string) error {
	m.Store.Delete(uuid)
	return nil
}
