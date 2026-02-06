package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &serviceInstanceDataSource{}
	_ datasource.DataSourceWithConfigure = &serviceInstanceDataSource{}
)

func NewServiceInstanceDataSource() datasource.DataSource {
	return &serviceInstanceDataSource{}
}

type serviceInstanceDataSource struct {
	meshServiceInstanceClient client.MeshServiceInstanceClient
}

func (d *serviceInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_instance"
}

func (d *serviceInstanceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Service instance by instance ID.",

		Attributes: serviceInstanceSchemaAttributes(false),
	}
}

func serviceInstanceSchemaAttributes(computed bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_version": schema.StringAttribute{
			MarkdownDescription: "Service instance API version.",
			Computed:            true,
		},

		"kind": schema.StringAttribute{
			MarkdownDescription: "meshObject type, always `meshServiceInstance`.",
			Computed:            true,
		},

		"metadata": schema.SingleNestedAttribute{
			MarkdownDescription: "Service instance metadata. Instance ID of the target service instance must be set here.",
			Required:            !computed,
			Computed:            computed,
			Attributes: map[string]schema.Attribute{
				"instance_id": schema.StringAttribute{
					MarkdownDescription: "Unique identifier of the service instance.",
					Required:            !computed,
					Computed:            computed,
				},
				"owned_by_workspace": schema.StringAttribute{
					MarkdownDescription: "Workspace that owns this service instance.",
					Computed:            true,
				},
				"owned_by_project": schema.StringAttribute{
					MarkdownDescription: "Project that owns this service instance.",
					Computed:            true,
				},
				"marketplace_identifier": schema.StringAttribute{
					MarkdownDescription: "Marketplace identifier.",
					Computed:            true,
				},
			},
		},

		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Service instance specification.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"creator": schema.StringAttribute{
					MarkdownDescription: "User who created this service instance.",
					Computed:            true,
				},
				"display_name": schema.StringAttribute{
					MarkdownDescription: "Human-readable display name.",
					Computed:            true,
				},
				"plan_id": schema.StringAttribute{
					MarkdownDescription: "Service plan identifier.",
					Computed:            true,
				},
				"service_id": schema.StringAttribute{
					MarkdownDescription: "Service identifier.",
					Computed:            true,
				},
			},
		},
	}
}

func (d *serviceInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshServiceInstanceClient = client.ServiceInstance
	})...)
}

func (d *serviceInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Get instance_id to query for service instance
	var instanceId string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("instance_id"), &instanceId)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceInstance, err := d.meshServiceInstanceClient.Read(ctx, instanceId)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read service instance", err.Error())
		return
	}

	if serviceInstance == nil {
		resp.Diagnostics.AddError("Service instance not found", fmt.Sprintf("Can't find service instance with ID '%s'.", instanceId))
		return
	}

	// Client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, serviceInstance)...)
}
