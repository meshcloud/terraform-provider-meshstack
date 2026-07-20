package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ datasource.DataSource              = &landingZonesDataSource{}
	_ datasource.DataSourceWithConfigure = &landingZonesDataSource{}
)

func NewLandingZonesDataSource() datasource.DataSource {
	return &landingZonesDataSource{}
}

type landingZonesDataSource struct {
	meshLandingZoneClient client.MeshLandingZoneClient
}

func (d *landingZonesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_landingzones"
}

func (d *landingZonesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshLandingZoneClient = client.LandingZone
	})...)
}

func (d *landingZonesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List landing zones with optional filters. Each element has the same shape as the " +
			"`meshstack_landingzone` data source, including a computed `ref` usable as `landing_zone_ref` in " +
			"tenant resources. Use `platform_uuid` to list the landing zones of a chosen platform. " +
			"\n\n" +
			"Requires the `LANDINGZONE_LIST` (workspace-scoped) or `ADM_LANDINGZONE_LIST` API-key right. " +
			"The workspace-scoped right also surfaces landing zones of platforms **published** to the caller's workspace.",
		Attributes: map[string]schema.Attribute{
			"platform_uuid": schema.StringAttribute{
				MarkdownDescription: "Filter to the landing zones of the platform with this uuid.",
				Optional:            true,
			},
			"identifier": schema.StringAttribute{
				MarkdownDescription: "Filter by landing zone identifier (`metadata.name`).",
				Optional:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Filter by display name.",
				Optional:            true,
			},
			"restricted": schema.BoolAttribute{
				MarkdownDescription: "Filter by restriction: `true` returns only restricted landing zones, `false` only unrestricted ones.",
				Optional:            true,
			},
			"owned_by_workspace": schema.StringAttribute{
				MarkdownDescription: "Filter by the identifier of the workspace that owns the landing zone.",
				Optional:            true,
			},
			"landing_zones": schema.ListNestedAttribute{
				MarkdownDescription: "Matching landing zones.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ref": meshRefByName(meshRefOptions{
							Kind:        client.MeshObjectKind.LandingZone,
							Description: "Reference to this landing zone, can be used as `landing_zone_ref` in tenant resources. The landing zone name is only unique together with its platform, so a `meshstack_tenant` references both `platform_ref` and `landing_zone_ref`.",
							Output:      true,
						}),
						"metadata": landingZoneMetadataDataSourceSchema(true),
						"spec":     landingZoneSpecDataSourceSchema(),
						"status":   landingZoneStatusDataSourceSchema(),
					},
				},
			},
		},
	}
}

func (d *landingZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var query client.MeshLandingZoneListQuery

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("platform_uuid"), &query.PlatformUuid)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("identifier"), &query.Identifier)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("display_name"), &query.DisplayName)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("restricted"), &query.Restricted)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("owned_by_workspace"), &query.OwnedByWorkspace)...)

	if resp.Diagnostics.HasError() {
		return
	}

	landingZones, err := d.meshLandingZoneClient.List(ctx, query)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list meshLandingZones", err.Error())
		return
	}

	models := make([]landingZoneModel, len(landingZones))
	for i := range landingZones {
		models[i] = landingZoneModelFrom(&landingZones[i])
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("landing_zones"), &models)...)
}
