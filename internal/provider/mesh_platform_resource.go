package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &meshPlatformResource{}
	_ resource.ResourceWithConfigure   = &meshPlatformResource{}
	_ resource.ResourceWithImportState = &meshPlatformResource{}
)

// NewMeshPlatformResource is a helper function to simplify the provider implementation.
func NewMeshPlatformResource() resource.Resource {
	return &meshPlatformResource{}
}

// meshPlatformResource is the resource implementation.
type meshPlatformResource struct {
	client *client.MeshStackProviderClient
}

// Metadata returns the resource type name.
func (r *meshPlatformResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mesh_platform"
}

// Configure adds the provider configured client to the resource.
func (r *meshPlatformResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *meshPlatformResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack platform.\n\n~> **Note:** Managing platforms requires an API key with sufficient admin permissions.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Platform datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v1"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshPlatform`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshPlatform"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Platform identifier.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[a-zA-Z0-9]+([._-][a-zA-Z0-9]+)*$`),
								"must be alphanumeric with dots, dashes or underscores",
							),
						},
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation date of the platform.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Deletion date of the platform.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the platform.",
						Required:            true,
					},
					"platform_type": schema.StringAttribute{
						MarkdownDescription: "Type of the platform (e.g., 'OpenStack', 'Azure', 'AWS', 'GCP', etc.).",
						Required:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description of the platform.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Platform specification tags.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Optional:            true,
						Computed:            true,
						Default:             mapdefault.StaticValue(types.MapValueMust(types.ListType{ElemType: types.StringType}, map[string]attr.Value{})),
					},
					"config": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform-specific configuration options.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"aws": schema.SingleNestedAttribute{
								MarkdownDescription: "AWS platform configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"account_id": schema.StringAttribute{
										MarkdownDescription: "AWS Account ID.",
										Required:            true,
									},
									"region": schema.StringAttribute{
										MarkdownDescription: "AWS Region.",
										Required:            true,
									},
									"endpoint_url": schema.StringAttribute{
										MarkdownDescription: "AWS API endpoint URL (optional, defaults to standard AWS endpoints).",
										Optional:            true,
									},
									"role_arn": schema.StringAttribute{
										MarkdownDescription: "IAM Role ARN for cross-account access.",
										Optional:            true,
									},
									"external_id": schema.StringAttribute{
										MarkdownDescription: "External ID for role assumption (used with role_arn).",
										Optional:            true,
									},
									"assume_role_session_name": schema.StringAttribute{
										MarkdownDescription: "Session name for role assumption.",
										Optional:            true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *meshPlatformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	platform := client.MeshPlatformCreate{
		Metadata: client.MeshPlatformCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &platform.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &platform.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &platform.Metadata.Name)...)

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

	resp.Diagnostics.Append(resp.State.Set(ctx, createdPlatform)...)
}

func (r *meshPlatformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var identifier string

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &identifier)...)

	if resp.Diagnostics.HasError() {
		return
	}

	platform, err := r.client.ReadPlatform(identifier)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read platform '%s'", identifier),
			err.Error(),
		)
		return
	}

	if platform == nil {
		// The platform was deleted outside of Terraform, so we remove it from the state
		resp.State.RemoveResource(ctx)
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, platform)...)
}

func (r *meshPlatformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	platform := client.MeshPlatformCreate{
		Metadata: client.MeshPlatformCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &platform.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &platform.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &platform.Metadata.Name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updatedPlatform, err := r.client.UpdatePlatform(platform.Metadata.Name, &platform)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Platform",
			"Could not update platform, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, updatedPlatform)...)
}

func (r *meshPlatformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var identifier string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &identifier)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePlatform(identifier)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete platform '%s'", identifier),
			err.Error(),
		)
		return
	}
}

func (r *meshPlatformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("name"), req, resp)
}
