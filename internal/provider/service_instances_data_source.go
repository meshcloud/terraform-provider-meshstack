package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &serviceInstancesDataSource{}
	_ datasource.DataSourceWithConfigure = &serviceInstancesDataSource{}
)

func NewServiceInstancesDataSource() datasource.DataSource {
	return &serviceInstancesDataSource{}
}

type serviceInstancesDataSource struct {
	meshServiceInstanceClient client.MeshServiceInstanceClient
}

func (d *serviceInstancesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_instances"
}

func (d *serviceInstancesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List of service instances.",

		Attributes: map[string]schema.Attribute{
			"workspace_identifier": schema.StringAttribute{
				MarkdownDescription: "Workspace identifier to filter service instances.",
				Optional:            true,
			},
			"project_identifier": schema.StringAttribute{
				MarkdownDescription: "Project identifier to filter service instances.",
				Optional:            true,
			},
			"marketplace_identifier": schema.StringAttribute{
				MarkdownDescription: "Marketplace or service broker identifier to filter service instances.",
				Optional:            true,
			},
			"service_identifier": schema.StringAttribute{
				MarkdownDescription: "Service definition identifier to filter service instances.",
				Optional:            true,
			},
			"plan_identifier": schema.StringAttribute{
				MarkdownDescription: "Service plan identifier to filter service instances.",
				Optional:            true,
			},
			"service_instances": schema.ListNestedAttribute{
				MarkdownDescription: "List of service instances.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: serviceInstanceSchemaAttributes(true),
				},
			},
		},
	}
}

func (d *serviceInstancesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshServiceInstanceClient = client.ServiceInstance
	})...)
}

func (d *serviceInstancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var workspaceIdentifier *string
	var projectIdentifier *string
	var marketplaceIdentifier *string
	var serviceIdentifier *string
	var planIdentifier *string

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("workspace_identifier"), &workspaceIdentifier)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_identifier"), &projectIdentifier)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("marketplace_identifier"), &marketplaceIdentifier)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("service_identifier"), &serviceIdentifier)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("plan_identifier"), &planIdentifier)...)

	if resp.Diagnostics.HasError() {
		return
	}

	filter := &client.MeshServiceInstanceFilter{
		WorkspaceIdentifier:   workspaceIdentifier,
		ProjectIdentifier:     projectIdentifier,
		MarketplaceIdentifier: marketplaceIdentifier,
		ServiceIdentifier:     serviceIdentifier,
		PlanIdentifier:        planIdentifier,
	}

	serviceInstances, err := d.meshServiceInstanceClient.List(ctx, filter)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read service instances", err.Error())
		return
	}

	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("service_instances"), serviceInstances, withValueFromConverterForClientTypeAny())...)
}
