package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &landingZoneDataSource{}
	_ datasource.DataSourceWithConfigure = &landingZoneDataSource{}
)

func NewLandingZoneDataSource() datasource.DataSource {
	return &landingZoneDataSource{}
}

type landingZoneDataSource struct {
	client *client.MeshStackProviderClient
}

func (d *landingZoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_landingzone"
}

// Schema defines the schema for the data source.
func (d *landingZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single landing zone by identifier.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Landing Zone API version.",
				Computed:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshLandingZone`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshLandingZone"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Landing Zone identifier.",
						Required:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags of the landing zone.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Computed:            true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the landing zone.",
						Computed:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description of the landing zone.",
						Computed:            true,
					},
					"automate_deletion_approval": schema.BoolAttribute{
						MarkdownDescription: "Whether deletion approval is automated for this landing zone.",
						Computed:            true,
					},
					"automate_deletion_replication": schema.BoolAttribute{
						MarkdownDescription: "Whether deletion replication is automated for this landing zone.",
						Computed:            true,
					},
					"info_link": schema.StringAttribute{
						MarkdownDescription: "Link to additional information about the landing zone.",
						Computed:            true,
					},
					"platform_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the platform this landing zone belongs to.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the platform.",
								Computed:            true,
							},
							"kind": schema.StringAttribute{
								MarkdownDescription: "Must always be set to meshPlatform",
								Computed:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("meshPlatform"),
								},
							},
						},
					},
					"platform_properties": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform-specific configuration options.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"aws":        awsPlatformConfigSchema(),
							"aks":        aksPlatformConfigSchema(),
							"azure":      azurePlatformConfigSchema(),
							"azurerg":    azureRgPlatformConfigSchema(),
							"gcp":        gcpPlatformConfigSchema(),
							"kubernetes": kubernetesPlatformConfigSchema(),
							"openshift":  openShiftPlatformConfigSchema(),
							"type": schema.StringAttribute{
								MarkdownDescription: "Type of the platform. This field is automatically inferred from which platform configuration is provided and cannot be set manually.",
								Computed:            true,
							},
						},
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Current Landing Zone status.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"disabled": schema.BoolAttribute{
						MarkdownDescription: "True if the landing zone is disabled.",
						Computed:            true,
					},
					"restricted": schema.BoolAttribute{
						MarkdownDescription: "If true, users will be unable to select this landing zone in meshPanel. " +
							"Only Platform teams can create tenants using restricted landing zones with the meshObject API.",
						Computed: true,
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *landingZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *landingZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name string

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	landingZone, err := d.client.ReadLandingZone(name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read landing zone '%s'", name),
			err.Error(),
		)
		return
	}

	if landingZone == nil {
		resp.Diagnostics.AddError(
			"Landing zone not found",
			fmt.Sprintf("The requested landingZone '%s' was not found.", name),
		)
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, landingZone)...)
}
