package clientmock

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshPlatformClient struct {
	Store *Store[client.MeshPlatform]
}

func (m MeshPlatformClient) Read(_ context.Context, uuid string) (*client.MeshPlatform, error) {
	if platform, ok := m.Store.Get(uuid); ok {
		return platform, nil
	}
	return nil, nil
}

// List applies only the plain attribute filters from the query; it does not simulate marketplace
// visibility or permissions (the mock has no notion of the calling workspace). Entitlement and
// config-redaction behaviour is therefore acceptance-only.
func (m MeshPlatformClient) List(_ context.Context, query client.MeshPlatformListQuery) ([]client.MeshPlatform, error) {
	var result []client.MeshPlatform
	for _, platform := range m.Store.Values() {
		if query.OwnedByWorkspace != nil && platform.Metadata.OwnedByWorkspace != *query.OwnedByWorkspace {
			continue
		}
		if query.Identifier != nil && platform.Metadata.Name != *query.Identifier {
			continue
		}
		if query.LocationIdentifier != nil && platform.Spec.LocationRef.Name != *query.LocationIdentifier {
			continue
		}
		if query.DisplayName != nil && platform.Spec.DisplayName != *query.DisplayName {
			continue
		}
		if query.Restriction != nil && platform.Spec.Availability.Restriction != *query.Restriction {
			continue
		}
		if query.PublicationState != nil && platform.Spec.Availability.PublicationState != *query.PublicationState {
			continue
		}
		if query.ContributingWorkspace != nil && !slices.Contains(platform.Spec.ContributingWorkspaces, *query.ContributingWorkspace) {
			continue
		}
		// PlatformTypeIdentifier is not applied: the platform response carries no platform-type identifier
		// (and spec.config is redacted cross-workspace), so this filter is acceptance-only.
		result = append(result, *platform)
	}
	return result, nil
}

func (m MeshPlatformClient) Create(_ context.Context, platform client.MeshPlatform) (*client.MeshPlatform, error) {
	platformUuid := uuid.NewString()
	platform.Metadata.Uuid = &platformUuid
	backendSecretBehavior(true, &platform, nil)
	m.Store.Set(platformUuid, &platform)
	return &platform, nil
}

func (m MeshPlatformClient) Update(_ context.Context, uuid string, platform client.MeshPlatform) (*client.MeshPlatform, error) {
	if existing, ok := m.Store.Get(uuid); ok {
		backendSecretBehavior(false, &platform.Spec, &existing.Spec)
		existing.Spec = platform.Spec
		return existing, nil
	}
	return nil, fmt.Errorf("platform not found: %s", uuid)
}

func (m MeshPlatformClient) Delete(_ context.Context, uuid string) error {
	m.Store.Delete(uuid)
	return nil
}
