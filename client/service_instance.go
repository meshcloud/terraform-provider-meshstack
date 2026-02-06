package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshServiceInstance struct {
	ApiVersion string                      `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                      `json:"kind" tfsdk:"kind"`
	Metadata   MeshServiceInstanceMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshServiceInstanceSpec     `json:"spec" tfsdk:"spec"`
}

type MeshServiceInstanceMetadata struct {
	OwnedByProject        string `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace      string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	MarketplaceIdentifier string `json:"marketplaceIdentifier" tfsdk:"marketplace_identifier"`
	InstanceId            string `json:"instanceId" tfsdk:"instance_id"`
}

type MeshServiceInstanceSpec struct {
	Creator     string `json:"creator" tfsdk:"creator"`
	DisplayName string `json:"displayName" tfsdk:"display_name"`
	PlanId      string `json:"planId" tfsdk:"plan_id"`
	ServiceId   string `json:"serviceId" tfsdk:"service_id"`
}

type MeshServiceInstanceClient struct {
	meshObject internal.MeshObjectClient[MeshServiceInstance]
}

type MeshServiceInstanceFilter struct {
	WorkspaceIdentifier   *string
	ProjectIdentifier     *string
	MarketplaceIdentifier *string
	ServiceIdentifier     *string
	PlanIdentifier        *string
}

func newServiceInstanceClient(ctx context.Context, httpClient *internal.HttpClient) MeshServiceInstanceClient {
	return MeshServiceInstanceClient{internal.NewMeshObjectClient[MeshServiceInstance](ctx, httpClient, "v2")}
}

func (c MeshServiceInstanceClient) Read(ctx context.Context, instanceId string) (*MeshServiceInstance, error) {
	return c.meshObject.Get(ctx, instanceId)
}

func (c MeshServiceInstanceClient) List(ctx context.Context, filter *MeshServiceInstanceFilter) ([]MeshServiceInstance, error) {
	var options []internal.RequestOption
	if filter != nil {
		if filter.WorkspaceIdentifier != nil {
			options = append(options, internal.WithUrlQuery("workspaceIdentifier", *filter.WorkspaceIdentifier))
		}
		if filter.ProjectIdentifier != nil {
			options = append(options, internal.WithUrlQuery("projectIdentifier", *filter.ProjectIdentifier))
		}
		if filter.MarketplaceIdentifier != nil {
			options = append(options, internal.WithUrlQuery("marketplaceIdentifier", *filter.MarketplaceIdentifier))
		}
		if filter.ServiceIdentifier != nil {
			options = append(options, internal.WithUrlQuery("serviceIdentifier", *filter.ServiceIdentifier))
		}
		if filter.PlanIdentifier != nil {
			options = append(options, internal.WithUrlQuery("planIdentifier", *filter.PlanIdentifier))
		}
	}
	return c.meshObject.List(ctx, options...)
}
