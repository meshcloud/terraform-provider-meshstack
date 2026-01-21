package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ resource.Resource                = &platformTypeResource{}
	_ resource.ResourceWithConfigure   = &platformTypeResource{}
	_ resource.ResourceWithImportState = &platformTypeResource{}
)

func NewPlatformTypeResource() resource.Resource {
	return &platformTypeResource{}
}

type platformTypeResource struct {
	meshPlatformTypeClient client.MeshPlatformTypeClient
}

type platformTypeRef struct {
	Name string `tfsdk:"name"`
}

type platformTypeResourceModel struct {
	client.MeshPlatformType
	Ref platformTypeRef `tfsdk:"ref"`
}

func (r *platformTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform_type"
}

func (r *platformTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshPlatformTypeClient = client.PlatformType
	})...)
}

func (r *platformTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack platform type.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "The version of the API used for the platform type. Defaults to `v1-preview`.",
				Computed:            true,
				Default:             stringdefault.StaticString("v1-preview"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "The kind of the object. Always `meshPlatformType`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshPlatformType"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.OneOf("meshPlatformType"),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Unique identifier of the platform type. Restricted to uppercase alphanumeric characters and dashes. Must be unique across all platform types.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.LengthAtMost(150),
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[A-Z0-9]+(-[A-Z0-9]+)*$`),
								"must be alphanumeric (uppercase only) with dashes, and have no leading, trailing or consecutive dashes",
							),
						},
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp of when the platform type was created.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the platform type.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the meshPlatformType shown in the UI.",
						Required:            true,
					},
					"category": schema.StringAttribute{
						MarkdownDescription: "Category of the platform type. Always `CUSTOM` for user-created platform types.",
						Computed:            true,
						Default:             stringdefault.StaticString("CUSTOM"),
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"default_endpoint": schema.StringAttribute{
						MarkdownDescription: "Default endpoint URL for platforms of this type.",
						Optional:            true,
					},
					"icon": schema.StringAttribute{
						MarkdownDescription: "Icon used to represent the platform type. Must be a base64 encoded data URI (e.g., `data:image/png;base64,...`).",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^data:image/[a-zA-Z0-9+.-]+;base64,`),
								"must be a valid base64 encoded data URI (starting with 'data:image/')",
							),
						},
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Status of the platform type.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"lifecycle": schema.SingleNestedAttribute{
						MarkdownDescription: "Lifecycle information of the platform type",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"state": schema.StringAttribute{
								MarkdownDescription: "Lifecycle state of the platform type. Either ACTIVE or DEACTIVATED.",
								Computed:            true,
							},
						},
					},
				},
			},

			"ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Reference to this platform type, can be used as input for `platform_type_ref` in platform resources.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Identifier of the platform type.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},
		},
	}
}

func (r *platformTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var apiVersion string
	var kind string
	var name string
	var displayName string
	var category string
	var defaultEndpoint *string
	var icon string

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &apiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("kind"), &kind)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("display_name"), &displayName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("category"), &category)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("default_endpoint"), &defaultEndpoint)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("icon"), &icon)...)

	if resp.Diagnostics.HasError() {
		return
	}

	platformType := client.MeshPlatformTypeCreate{
		ApiVersion: apiVersion,
		Kind:       kind,
		Metadata: client.MeshPlatformTypeCreateMetadata{
			Name: name,
		},
		Spec: client.MeshPlatformTypeSpec{
			DisplayName:     displayName,
			Category:        category,
			DefaultEndpoint: defaultEndpoint,
			Icon:            icon,
		},
	}

	createdPlatformType, err := r.meshPlatformTypeClient.Create(ctx, &platformType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Platform Type",
			"Could not create platform type, unexpected error: "+err.Error(),
		)
		return
	}

	state := platformTypeResourceModel{
		MeshPlatformType: *createdPlatformType,
		Ref: platformTypeRef{
			Name: createdPlatformType.Metadata.Name,
		},
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *platformTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	platformType, err := r.meshPlatformTypeClient.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read platform type '%s'", name),
			err.Error(),
		)
		return
	}

	if platformType == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state := platformTypeResourceModel{
		MeshPlatformType: *platformType,
		Ref: platformTypeRef{
			Name: platformType.Metadata.Name,
		},
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *platformTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planApiVersion string
	var planKind string
	var planName string
	var planDisplayName string
	var planCategory string
	var planDefaultEndpoint *string
	var planIcon string
	var stateName string

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &planApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("kind"), &planKind)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &planName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("display_name"), &planDisplayName)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("category"), &planCategory)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("default_endpoint"), &planDefaultEndpoint)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("icon"), &planIcon)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &stateName)...)

	if resp.Diagnostics.HasError() {
		return
	}

	platformType := client.MeshPlatformTypeCreate{
		ApiVersion: planApiVersion,
		Kind:       planKind,
		Metadata: client.MeshPlatformTypeCreateMetadata{
			Name: planName,
		},
		Spec: client.MeshPlatformTypeSpec{
			DisplayName:     planDisplayName,
			Category:        planCategory,
			DefaultEndpoint: planDefaultEndpoint,
			Icon:            planIcon,
		},
	}

	updatedPlatformType, err := r.meshPlatformTypeClient.Update(ctx, stateName, &platformType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Platform Type",
			"Could not update platform type, unexpected error: "+err.Error(),
		)
		return
	}

	state := platformTypeResourceModel{
		MeshPlatformType: *updatedPlatformType,
		Ref: platformTypeRef{
			Name: updatedPlatformType.Metadata.Name,
		},
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *platformTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.meshPlatformTypeClient.Delete(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete platform type '%s'", name),
			err.Error(),
		)
		return
	}
}

func (r *platformTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("name"), req, resp)
}
