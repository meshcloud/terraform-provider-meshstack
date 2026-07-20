package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

var (
	_ datasource.DataSource              = &platformsDataSource{}
	_ datasource.DataSourceWithConfigure = &platformsDataSource{}
)

func NewPlatformsDataSource() datasource.DataSource {
	return &platformsDataSource{}
}

type platformsDataSource struct {
	meshPlatformClient client.MeshPlatformClient
}

func (d *platformsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platforms"
}

func (d *platformsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshPlatformClient = client.Platform
	})...)
}

func (d *platformsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List platforms with optional filters. Each element has the same shape as the " +
			"`meshstack_platform` data source, including a computed `ref` usable as `platform_ref` in " +
			"landing zone and tenant resources." + platformConfigVisibilityNote,
		Attributes: map[string]schema.Attribute{
			"owned_by_workspace": schema.StringAttribute{
				MarkdownDescription: "Filter by the identifier of the workspace that owns the platform.",
				Optional:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "Filter by platform identifier (`metadata.name`).",
				Optional:            true,
			},
			"location_identifier": schema.StringAttribute{
				MarkdownDescription: "Filter by location identifier.",
				Optional:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Filter by display name.",
				Optional:            true,
			},
			"restriction": schema.StringAttribute{
				MarkdownDescription: "Filter by access restriction. One of: `PUBLIC`, `PRIVATE`, `RESTRICTED`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("PUBLIC", "PRIVATE", "RESTRICTED"),
				},
			},
			"publication_state": schema.StringAttribute{
				MarkdownDescription: "Filter by marketplace publication state. One of: `PUBLISHED`, `UNPUBLISHED`, `REQUESTED`, `REJECTED`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("PUBLISHED", "UNPUBLISHED", "REQUESTED", "REJECTED"),
				},
			},
			"contributing_workspace": schema.StringAttribute{
				MarkdownDescription: "Filter by a contributing workspace identifier.",
				Optional:            true,
			},
			"platform_type_identifier": schema.StringAttribute{
				MarkdownDescription: "Filter by platform type identifier (the platform type's `metadata.name`), matched server-side. " +
					"Consistent with `meshstack_tenant`'s `status.platform_type_identifier`.",
				Optional: true,
			},
			"platforms": schema.ListNestedAttribute{
				MarkdownDescription: "Matching platforms.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"metadata":   platformMetadataDataSourceSchema(true),
						"identifier": platformIdentifierDataSourceSchema(),
						"ref":        meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.Platform, Description: "Reference to this platform, can be used as `platform_ref` in landing zone and tenant resources.", Output: true}),
						"spec":       platformSpecDataSourceSchema(),
					},
				},
			},
		},
	}
}

func (d *platformsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var query client.MeshPlatformListQuery

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("owned_by_workspace"), &query.OwnedByWorkspace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("identifier"), &query.Identifier)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("location_identifier"), &query.LocationIdentifier)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("display_name"), &query.DisplayName)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("restriction"), &query.Restriction)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("publication_state"), &query.PublicationState)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("contributing_workspace"), &query.ContributingWorkspace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("platform_type_identifier"), &query.PlatformTypeIdentifier)...)

	if resp.Diagnostics.HasError() {
		return
	}

	platforms, err := d.meshPlatformClient.List(ctx, query)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list meshPlatforms", err.Error())
		return
	}

	models := make([]platformModel, len(platforms))
	for i := range platforms {
		models[i] = platformModelFromDto(&platforms[i])
	}

	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("platforms"), models,
		secret.WithDatasourceConverter(),
		generic.WithSliceTypeAsSet(clientTypes.IsSet),
	)...)
}
