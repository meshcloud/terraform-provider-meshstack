package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/modifiers/platformtypemodifier"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
	"github.com/meshcloud/terraform-provider-meshstack/internal/validators"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &platformResource{}
	_ resource.ResourceWithConfigure   = &platformResource{}
	_ resource.ResourceWithImportState = &platformResource{}
	_ resource.ResourceWithModifyPlan  = &platformResource{}
)

// NewPlatformResource is a helper function to simplify the provider implementation.
func NewPlatformResource() resource.Resource {
	return &platformResource{}
}

// platformResource is the resource implementation.
type platformResource struct {
	meshPlatformClient client.MeshPlatformClient
}

// Metadata returns the resource type name.
func (r *platformResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform"
}

// Configure adds the provider configured client to the resource.
func (r *platformResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshPlatformClient = client.Platform
	})...)
}

func (r *platformResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	markdownDescription := "Represents a meshStack platform.\n\n" +
		"Please note that for the meshPlatform, the following limitations apply:\n" +
		"* Deleting and re-creating a platform with the same identifier is not possible. Once you have used a platform identifier, you cannot use it again, even if the platform has been deleted. You may run into this issue when you attempt to modify an immutable attribute and terraform therefore attempts to replace (i.e., delete and recreate) the entire platform, which will result in an error with a status code of `409` due to the identifier already being used by a deleted platform.\n" +
		"* Changing the owning workspace of a platform (`metadata.owned_by_workspace`) is not possible. To transfer the ownership of a platform, you must use meshPanel."

	quotaDefinitionAttrTypes := map[string]attr.Type{
		"quota_key":               types.StringType,
		"label":                   types.StringType,
		"description":             types.StringType,
		"unit":                    types.StringType,
		"min_value":               types.Int64Type,
		"max_value":               types.Int64Type,
		"auto_approval_threshold": types.Int64Type,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescription,
		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "Unique identifier of the platform (server-generated).",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Make sure you use a unique platform identifier within a Location. Location + Platform identifiers are being used to uniquely identify a platform in meshStack. You cannot change this identifier after creation of a platform.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`),
								"must be alphanumeric with dashes, must be lowercase, and have no leading, trailing or consecutive dashes",
							),
						},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "The identifier of the workspace that owns this meshPlatform.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "The human-readable display name of the meshPlatform.",
						Required:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description of the meshPlatform.",
						Required:            true,
					},
					"endpoint": schema.StringAttribute{
						MarkdownDescription: "The web console URL endpoint of the platform.",
						Required:            true,
					},
					"support_url": schema.StringAttribute{
						MarkdownDescription: "URL for platform support documentation.",
						Optional:            true,
					},
					"documentation_url": schema.StringAttribute{
						MarkdownDescription: "URL for platform documentation.",
						Optional:            true,
					},
					"location_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the location where this platform is situated.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"kind": schema.StringAttribute{
								MarkdownDescription: "meshObject type, always `meshLocation`.",
								Computed:            true,
								Optional:            true,
								Default:             stringdefault.StaticString("meshLocation"),
								PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
							},
							"name": schema.StringAttribute{
								MarkdownDescription: "Identifier of the Location.",
								Required:            true,
							},
						},
					},
					"contributing_workspaces": schema.SetAttribute{
						MarkdownDescription: "A list of workspace identifiers that may contribute to this meshPlatform.",
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
					},
					"availability": schema.SingleNestedAttribute{
						MarkdownDescription: "Availability configuration for the meshPlatform.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"restriction": schema.StringAttribute{
								MarkdownDescription: "Access restriction for the platform. Must be one of: `PUBLIC`, `PRIVATE`, `RESTRICTED`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("PUBLIC", "PRIVATE", "RESTRICTED"),
								},
							},
							"publication_state": schema.StringAttribute{
								MarkdownDescription: "Marketplace publication state of the platform. Must be one of: `PUBLISHED`, `UNPUBLISHED`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("PUBLISHED", "UNPUBLISHED"),
								},
							},
							// TODO: check this is not empty if set to restricted and that it's set to the owner if private
							"restricted_to_workspaces": schema.SetAttribute{
								MarkdownDescription: "If the restriction is set to `RESTRICTED`, you can specify the workspace identifiers this meshPlatform is restricted to.",
								ElementType:         types.StringType,
								Optional:            true,
								Computed:            true,
								Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
							},
						},
					},
					"quota_definitions": schema.SetAttribute{
						MarkdownDescription: "List of quota definitions for the platform.",
						Optional:            true,
						Computed:            true,
						Sensitive:           false,
						ElementType: types.ObjectType{
							AttrTypes: quotaDefinitionAttrTypes,
						},
						Default: setdefault.StaticValue(types.SetValueMust(types.ObjectType{
							AttrTypes: quotaDefinitionAttrTypes,
						}, []attr.Value{})),
					},
					"config": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform-specific configuration settings.",
						Required:            true,
						Sensitive:           false,
						Validators: []validator.Object{
							validators.ExactlyOneAttributeValidator{},
						},
						Attributes: map[string]schema.Attribute{
							"aws":        awsPlatformSchema(),
							"aks":        aksPlatformSchema(),
							"azure":      azurePlatformSchema(),
							"azurerg":    azureRgPlatformSchema(),
							"custom":     customPlatformSchema(),
							"gcp":        gcpPlatformSchema(),
							"kubernetes": kubernetesPlatformSchema(),
							"openshift":  openShiftPlatformSchema(),
							"type": schema.StringAttribute{
								MarkdownDescription: "Type of the platform. This field is automatically inferred from which platform configuration is provided and cannot be set manually.",
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.RequiresReplace(),
									platformtypemodifier.SetTypeFromPlatform(),
								},
							},
						},
					},
				},
			},
		},
	}
}

func meteringProcessingConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Processing configuration for metering",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"compact_timelines_after_days": schema.Int64Attribute{
				MarkdownDescription: "Defines the number of days after which timelines are compacted to save database space. This means that meshMetering will only retain actual state changes instead of every single observation point. The default of 30 days is usually sufficient.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(30),
			},
			"delete_raw_data_after_days": schema.Int64Attribute{
				MarkdownDescription: "Defines the number of days meshMetering retains raw data, such as states and events. This enables data reprocessing as long as the raw data is available. Although usually not relevant after chargeback statements are generated, a grace period is provided by default. The default of 65 days is usually sufficient.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(65),
			},
		},
	}
}

func platformConverterOptions(ctx context.Context, config, plan, state generic.AttributeGetter) generic.ConverterOptions {
	return secret.WithConverterSupport(ctx, config, plan, state).Append(
		generic.WithUseSetForElementsOf[clientTypes.StringSetElem](),
		generic.WithUseSetForElementsOf[client.QuotaDefinition](),
		generic.WithUseSetForElementsOf[client.AzureRoleMapping](),
		generic.WithUseSetForElementsOf[client.OpenShiftPlatformRoleMapping](),
	)
}

func (r *platformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	converterOptions := platformConverterOptions(ctx, req.Config, req.Plan, nil)
	createDto := generic.Get[client.MeshPlatform](ctx, req.Plan, &resp.Diagnostics, converterOptions.Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}
	createdPlatform, err := r.meshPlatformClient.Create(ctx, createDto)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Platform",
			"Could not create platform, unexpected error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, createdPlatform, converterOptions...)...)
}

func (r *platformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the resource ID (which should be the UUID)
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	readPlatform, err := r.meshPlatformClient.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read platform with UUID '%s'", uuid),
			err.Error(),
		)
		return
	}
	if readPlatform == nil {
		// The platform was deleted outside of Terraform, so we remove it from the state
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, readPlatform, platformConverterOptions(ctx, nil, nil, req.State)...)...)
}

func (r *platformResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// do nothing in case of delete
		return
	}
	secret.WalkSecretPathsIn(req.Plan.Raw, &resp.Diagnostics, func(attributePath path.Path, diags *diag.Diagnostics) {
		secret.SetHashToUnknownIfVersionChanged(ctx, req.Plan, req.State, &resp.Plan)(attributePath, diags)
	})
}

func (r *platformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	converterOptions := platformConverterOptions(ctx, req.Config, req.Plan, req.State)

	platform := generic.Get[client.MeshPlatform](ctx, req.Plan, &resp.Diagnostics, converterOptions...)

	updatedPlatform, err := r.meshPlatformClient.Update(ctx, *platform.Metadata.Uuid, platform)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Platform",
			"Could not update platform, unexpected error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, updatedPlatform, converterOptions...)...)
}

func (r *platformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.meshPlatformClient.Delete(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete platform with UUID '%s'", uuid),
			err.Error(),
		)
		return
	}
}

func (r *platformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
}
