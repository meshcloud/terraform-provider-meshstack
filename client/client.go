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

	"github.com/hashicorp/go-version"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

const MinMeshStackVersion = "2026.2.0"

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

	// Validate meshStack version compatibility
	meshInfo, err := httpClient.GetMeshInfo(ctx)
	if err != nil {
		return Client{}, fmt.Errorf("failed to retrieve meshStack version information from /mesh/info endpoint: %w", err)
	}

	minVersion, err := version.NewVersion(MinMeshStackVersion)
	if err != nil {
		return Client{}, fmt.Errorf("invalid minimum version format %s: %w", MinMeshStackVersion, err)
	}

	actualVersion, err := version.NewVersion(meshInfo.Version)
	if err != nil {
		return Client{}, fmt.Errorf("invalid meshStack version format %s: %w", meshInfo.Version, err)
	}

	if actualVersion.LessThan(minVersion) {
		return Client{}, fmt.Errorf("unsupported meshStack version: meshStack is running version %s, but this client requires version %s or higher", meshInfo.Version, MinMeshStackVersion)
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
