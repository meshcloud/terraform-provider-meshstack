package clientmock

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type meshBuildingBlockDefinitionClient struct {
	Store        *Store[client.MeshBuildingBlockDefinition]
	StoreVersion *Store[client.MeshBuildingBlockDefinitionVersion]
	StoreRunner  *Store[client.MeshBuildingBlockRunner]
}

// wifPlaceholder matches the placeholder syntax of a runner's subject template.
var wifPlaceholder = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

// setStatus mirrors the backend's status section as far as the provider reads it: the workload identity
// federation meshStack resolves from the runner of the definition's latest version.
func (m meshBuildingBlockDefinitionClient) setStatus(definition *client.MeshBuildingBlockDefinition) *client.MeshBuildingBlockDefinition {
	if definition.Status == nil {
		definition.Status = &client.MeshBuildingBlockDefinitionStatus{}
	}
	definition.Status.WorkloadIdentityFederation = m.resolveWif(definition)
	return definition
}

func (m meshBuildingBlockDefinitionClient) resolveWif(definition *client.MeshBuildingBlockDefinition) *client.MeshBuildingBlockDefinitionWorkloadIdentityFederation {
	latest := m.latestVersion(*definition.Metadata.Uuid)
	if latest == nil || latest.Spec.RunnerRef == nil {
		return nil
	}
	runner, ok := m.StoreRunner.Get(latest.Spec.RunnerRef.Uuid)
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
	return &client.MeshBuildingBlockDefinitionWorkloadIdentityFederation{
		Issuer:  issuer,
		Subject: resolved,
		Gcp:     wif.Gcp,
		Aws:     wif.Aws,
		Azure:   wif.Azure,
	}
}

func (m meshBuildingBlockDefinitionClient) latestVersion(definitionUuid string) (latest *client.MeshBuildingBlockDefinitionVersion) {
	for _, version := range m.StoreVersion.Values() {
		if version.Spec.BuildingBlockDefinitionRef == nil || version.Spec.BuildingBlockDefinitionRef.Uuid != definitionUuid {
			continue
		}
		if latest == nil || *version.Spec.VersionNumber > *latest.Spec.VersionNumber {
			latest = version
		}
	}
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
