package client

import (
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
}

func New(rootUrl *url.URL, userAgent, apiKey, apiSecret string) Client {
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
		newBuildingBlockClient(httpClient),
		newBuildingBlockV2Client(httpClient),
		newIntegrationClient(httpClient),
		newLandingZoneClient(httpClient),
		newLocationClient(httpClient),
		newPaymentMethodClient(httpClient),
		newPlatformClient(httpClient),
		newProjectClient(httpClient),
		newProjectGroupBindingClient(httpClient),
		newProjectUserBindingClient(httpClient),
		newTagDefinitionClient(httpClient),
		newTenantClient(httpClient),
		newTenantV4Client(httpClient),
		newWorkspaceClient(httpClient),
		newWorkspaceGroupBindingClient(httpClient),
		newWorkspaceUserBindingClient(httpClient),
	}
}
