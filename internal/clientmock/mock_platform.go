package clientmock

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshPlatformClient struct {
	Store Store[client.MeshPlatform]
}

func (m MeshPlatformClient) Read(_ context.Context, uuid string) (*client.MeshPlatform, error) {
	if platform, ok := m.Store[uuid]; ok {
		return platform, nil
	}
	return nil, nil
}

func (m MeshPlatformClient) Create(_ context.Context, platform client.MeshPlatform) (*client.MeshPlatform, error) {
	platformUuid := acctest.RandString(32)
	platform.Kind = "meshPlatform"
	platform.Metadata.Uuid = &platformUuid
	backendSecretBehavior(true, &platform, nil)
	m.Store[platformUuid] = &platform
	return &platform, nil
}

func (m MeshPlatformClient) Update(_ context.Context, uuid string, platform client.MeshPlatform) (*client.MeshPlatform, error) {
	if existing, ok := m.Store[uuid]; ok {
		backendSecretBehavior(false, &platform.Spec, &existing.Spec)
		existing.Spec = platform.Spec
		return existing, nil
	}
	return nil, fmt.Errorf("platform not found: %s", uuid)
}

func (m MeshPlatformClient) Delete(_ context.Context, uuid string) error {
	delete(m.Store, uuid)
	return nil
}
