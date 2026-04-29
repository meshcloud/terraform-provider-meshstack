package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	"github.com/meshcloud/terraform-provider-meshstack/client/version"
)

var MinMeshStackVersion = version.MustParse("2026.10.0")

type Client struct {
	ApiKey                         MeshApiKeyClient
	BuildingBlock                  MeshBuildingBlockClient
	BuildingBlockV2                MeshBuildingBlockV2Client
	BuildingBlockDefinition        MeshBuildingBlockDefinitionClient
	BuildingBlockDefinitionVersion MeshBuildingBlockDefinitionVersionClient
	Integration                    MeshIntegrationClient
	LandingZone                    MeshLandingZoneClient
	Location                       MeshLocationClient
	PaymentMethod                  MeshPaymentMethodClient
	Platform                       MeshPlatformClient
	PlatformType                   MeshPlatformTypeClient
	Project                        MeshProjectClient
	ProjectGroupBinding            MeshProjectGroupBindingClient
	ProjectUserBinding             MeshProjectUserBindingClient
	ServiceInstance                MeshServiceInstanceClient
	TagDefinition                  MeshTagDefinitionClient
	Tenant                         MeshTenantClient
	TenantV4                       MeshTenantV4Client
	Workspace                      MeshWorkspaceClient
	WorkspaceGroupBinding          MeshWorkspaceGroupBindingClient
	WorkspaceUserBinding           MeshWorkspaceUserBindingClient
}

func New(ctx context.Context, rootUrl *url.URL, userAgent, apiKey, apiSecret string, apiToken string) (Client, error) {
	httpClient := &internal.HttpClient{
		Client:    http.Client{Timeout: 5 * time.Minute},
		RootUrl:   rootUrl,
		UserAgent: userAgent,

		// Putting authentication with meshStack API into HttpClient
		// saves use from passing ApiKey/ApiSecret down to client factory methods below.
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
	}

	if apiToken != "" {
		httpClient.Authorization = "Bearer " + apiToken

		if expiresAt, err := parseTokenExpiration(apiToken); err == nil {
			httpClient.AuthorizationExpiresAt = expiresAt
		} else {
			// If token has no expiration we assume it is valid for the default duration.
			httpClient.AuthorizationExpiresAt = time.Now().Add(6 * time.Hour)
		}
	}

	// Check meshStack version compatibility
	if meshInfo, err := httpClient.GetMeshInfo(ctx); err != nil {
		return Client{}, fmt.Errorf("failed to retrieve meshStack version information from /mesh/info endpoint: %w", err)
	} else if meshInfo.Version.Less(MinMeshStackVersion) {
		return Client{}, fmt.Errorf("unsupported meshStack version: meshStack is running version %s, but this client requires version %s or higher", meshInfo.Version, MinMeshStackVersion)
	}

	return Client{
		ApiKey:                         newApiKeyClient(ctx, httpClient),
		BuildingBlock:                  newBuildingBlockClient(ctx, httpClient),
		BuildingBlockV2:                newBuildingBlockV2Client(ctx, httpClient),
		BuildingBlockDefinition:        newBuildingBlockDefinitionClient(ctx, httpClient),
		BuildingBlockDefinitionVersion: newBuildingBlockDefinitionVersionClient(ctx, httpClient),
		Integration:                    newIntegrationClient(ctx, httpClient),
		LandingZone:                    newLandingZoneClient(ctx, httpClient),
		Location:                       newLocationClient(ctx, httpClient),
		PaymentMethod:                  newPaymentMethodClient(ctx, httpClient),
		Platform:                       newPlatformClient(ctx, httpClient),
		PlatformType:                   newPlatformTypeClient(ctx, httpClient),
		Project:                        newProjectClient(ctx, httpClient),
		ProjectGroupBinding:            newProjectGroupBindingClient(ctx, httpClient),
		ProjectUserBinding:             newProjectUserBindingClient(ctx, httpClient),
		ServiceInstance:                newServiceInstanceClient(ctx, httpClient),
		TagDefinition:                  newTagDefinitionClient(ctx, httpClient),
		Tenant:                         newTenantClient(ctx, httpClient),
		TenantV4:                       newTenantV4Client(ctx, httpClient),
		Workspace:                      newWorkspaceClient(ctx, httpClient),
		WorkspaceGroupBinding:          newWorkspaceGroupBindingClient(ctx, httpClient),
		WorkspaceUserBinding:           newWorkspaceUserBindingClient(ctx, httpClient),
	}, nil
}

func parseTokenExpiration(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, err
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, err
	}

	if claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("expiration claim missing")
	}

	return time.Unix(claims.Exp, 0), nil
}
