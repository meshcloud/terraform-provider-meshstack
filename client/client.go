package client

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type Client struct {
	BuildingBlock         MeshBuildingBlockClient
	BuildingBlockV2       MeshBuildingBlockV2Client
	Integration           MeshIntegrationClient
	LandingZone           MeshLandingZoneClient
	Location              MeshLocationClient
	PaymentMethod         MeshPaymentMethodClient
	Platform              MeshPlatformClient
	Project               MeshProjectClient
	ProjectGroupBinding   MeshProjectGroupBindingClient
	ProjectUserBinding    MeshProjectUserBindingClient
	TagDefinition         MeshTagDefinitionClient
	Tenant                MeshTenantClient
	TenantV4              MeshTenantV4Client
	Workspace             MeshWorkspaceClient
	WorkspaceGroupBinding MeshWorkspaceGroupBindingClient
	WorkspaceUserBinding  MeshWorkspaceUserBindingClient
	PlatformType          MeshPlatformTypeClient
}

func New(ctx context.Context, rootUrl *url.URL, userAgent, apiKey, apiSecret string) Client {
	httpClient := &internal.HttpClient{
		Client:    http.Client{Timeout: 5 * time.Minute},
		RootUrl:   rootUrl,
		UserAgent: userAgent,

		// Putting authentication with meshStack API into HttpClient
		// saves use from passing ApiKey/ApiSecret down to client factory methods below.
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
	}
	return Client{
		newBuildingBlockClient(ctx, httpClient),
		newBuildingBlockV2Client(ctx, httpClient),
		newIntegrationClient(ctx, httpClient),
		newLandingZoneClient(ctx, httpClient),
		newLocationClient(ctx, httpClient),
		newPaymentMethodClient(ctx, httpClient),
		newPlatformClient(ctx, httpClient),
		newProjectClient(ctx, httpClient),
		newProjectGroupBindingClient(ctx, httpClient),
		newProjectUserBindingClient(ctx, httpClient),
		newTagDefinitionClient(ctx, httpClient),
		newTenantClient(ctx, httpClient),
		newTenantV4Client(ctx, httpClient),
		newWorkspaceClient(ctx, httpClient),
		newWorkspaceGroupBindingClient(ctx, httpClient),
		newWorkspaceUserBindingClient(ctx, httpClient),
		newPlatformTypeClient(ctx, httpClient),
	}
}
