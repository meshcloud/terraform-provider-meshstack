package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/modifiers/platformtypemodifier"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &platformResource{}
	_ resource.ResourceWithConfigure   = &platformResource{}
	_ resource.ResourceWithImportState = &platformResource{}
)

// NewPlatformResource is a helper function to simplify the provider implementation.
func NewPlatformResource() resource.Resource {
	return &platformResource{}
}

// platformResource is the resource implementation.
type platformResource struct {
	client *client.MeshStackProviderClient
}

// Metadata returns the resource type name.
func (r *platformResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform"
}

// Configure adds the provider configured client to the resource.
func (r *platformResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *platformResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	markdownDescription := "Represents a meshStack platform.\n\n" +
		"Please note that the meshPlatform API endpoints are still in preview state and therefore the following limitations apply:\n" +
		"* Deleting and re-creating a platform with the same identifier is not possible. Once you have used a platform identifier, you cannot use it again, even if the platform has been deleted. You may run into this issue when you attempt to modify an immutable attribute and terraform therefore attempts to replace (i.e., delete and recreate) the entire platform, which will result in an error with a status code of `409` due to the identifier already being used by a deleted platform.\n" +
		"* Changing the owning workspace of a platform (`metadata.owned_by_workspace`) is not possible. To transfer the ownership of a platform, you must use meshPanel."

	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescription,
		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Platform datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v2-preview"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshPlatform`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshPlatform"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshPlatform"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

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
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation timestamp of the platform (server-generated).",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp when the meshPlatform was deleted, null if not deleted.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
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
								Default:             stringdefault.StaticString("meshLocation"),
								Validators: []validator.String{
									stringvalidator.OneOf([]string{"meshLocation"}...),
								},
								PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
							},
							"name": schema.StringAttribute{
								MarkdownDescription: "Identifier of the Location.",
								Required:            true,
							},
						},
					},
					"contributing_workspaces": schema.ListAttribute{
						MarkdownDescription: "A list of workspace identifiers that may contribute to this meshPlatform.",
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
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
							"restricted_to_workspaces": schema.ListAttribute{
								MarkdownDescription: "If the restriction is set to `RESTRICTED`, you can specify the workspace identifiers this meshPlatform is restricted to.",
								ElementType:         types.StringType,
								Optional:            true,
								Computed:            true,
								Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
							},
						},
					},
					"quota_definitions": schema.ListAttribute{
						MarkdownDescription: "List of quota definitions for the platform.",
						Required:            true,
						Sensitive:           false,
						ElementType: types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"quota_key":               types.StringType,
								"label":                   types.StringType,
								"description":             types.StringType,
								"unit":                    types.StringType,
								"min_value":               types.Int64Type,
								"max_value":               types.Int64Type,
								"auto_approval_threshold": types.Int64Type,
							},
						},
					},
					"config": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform-specific configuration settings.",
						Required:            true,
						Sensitive:           false,
						PlanModifiers: []planmodifier.Object{
							platformtypemodifier.ValidateSinglePlatform(),
						},
						Attributes: map[string]schema.Attribute{
							"aws":        awsPlatformSchema(),
							"aks":        aksPlatformSchema(),
							"azure":      azurePlatformSchema(),
							"azurerg":    azureRgPlatformSchema(),
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
				MarkdownDescription: "Number of days after which timelines should be compacted.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(30),
			},
			"delete_raw_data_after_days": schema.Int64Attribute{
				MarkdownDescription: "Number of days after which raw data should be deleted.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(65),
			},
		},
	}
}

func (r *platformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	platform := client.MeshPlatformCreate{
		Metadata: client.MeshPlatformCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &platform.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &platform.Spec)...)

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &platform.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &platform.Metadata.OwnedByWorkspace)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createdPlatform, err := r.client.CreatePlatform(&platform)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Platform",
			"Could not create platform, unexpected error: "+err.Error(),
		)
		return
	}

	handleObfuscatedSecrets(&createdPlatform.Spec.Config, &platform.Spec.Config, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, createdPlatform)...)
}

func (r *platformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the resource ID (which should be the UUID)
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	readPlatform, err := r.client.ReadPlatform(uuid)
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

	statePlatformSpec := client.MeshPlatformSpec{}
	req.State.GetAttribute(ctx, path.Root("spec"), &statePlatformSpec)
	if resp.Diagnostics.HasError() {
		return
	}

	handleObfuscatedSecrets(&readPlatform.Spec.Config, &statePlatformSpec.Config, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, readPlatform)...)
}

func (r *platformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	platform := client.MeshPlatformUpdate{
		Metadata: client.MeshPlatformUpdateMetadata{},
	}

	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	if uuid == "" {
		resp.Diagnostics.AddError(
			"Resource ID Missing",
			"The resource ID is missing. This should not happen.",
		)
		return
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &platform.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &platform.Spec)...)

	// Handle metadata fields including UUID for updates
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &platform.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &platform.Metadata.OwnedByWorkspace)...)
	platform.Metadata.Uuid = uuid

	if resp.Diagnostics.HasError() {
		return
	}

	updatedPlatform, err := r.client.UpdatePlatform(uuid, &platform)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Platform",
			"Could not update platform, unexpected error: "+err.Error(),
		)
		return
	}

	handleObfuscatedSecrets(&updatedPlatform.Spec.Config, &platform.Spec.Config, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, updatedPlatform)...)
}

func (r *platformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePlatform(uuid)
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
