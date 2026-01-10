package client

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshStackProviderClient struct {
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

func NewClient(rootUrl *url.URL, providerVersion, apiKey, apiSecret string) MeshStackProviderClient {
	httpClient := &internal.HttpClient{
		Client:    http.Client{Timeout: 5 * time.Minute},
		RootUrl:   rootUrl,
		UserAgent: fmt.Sprintf("terraform-provider-meshstack/%s", providerVersion),

		// Putting authentication with meshStack API into HttpClient
		// saves use from passing ApiKey/ApiSecret down to client factory methods below.
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
	}
	return MeshStackProviderClient{
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
