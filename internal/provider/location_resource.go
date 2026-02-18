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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ resource.Resource                = &locationResource{}
	_ resource.ResourceWithConfigure   = &locationResource{}
	_ resource.ResourceWithImportState = &locationResource{}
)

func NewLocationResource() resource.Resource {
	return &locationResource{}
}

type locationResource struct {
	meshLocationClient client.MeshLocationClient
}

type locationRef struct {
	Name string `tfsdk:"name"`
}

type locationResourceModel struct {
	client.MeshLocation
	Ref locationRef `tfsdk:"ref"`
}

func (r *locationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_location"
}

func (r *locationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshLocationClient = client.Location
	})...)
}

func (r *locationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack location.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Location identifier. Must be unique across all locations.",
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
						MarkdownDescription: "Identifier of the workspace that owns this location.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"uuid": schema.StringAttribute{
						MarkdownDescription: "Unique identifier of the location (server-generated).",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "The human-readable display name of the location.",
						Required:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "The description of the location.",
						Required:            true,
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"is_public": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether the location has any public platform instances associated with it.",
						Computed:            true,
					},
				},
			},

			"ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Reference to this location, can be used as input for `location_ref` in platform resources.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Identifier of the Location.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},
		},
	}
}

func (r *locationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var name string
	var ownedByWorkspace string
	var displayName string
	var description string

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &ownedByWorkspace)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("display_name"), &displayName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("description"), &description)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location := client.MeshLocationCreate{
		ApiVersion: "v1",
		Metadata: client.MeshLocationCreateMetadata{
			Name:             name,
			OwnedByWorkspace: ownedByWorkspace,
		},
		Spec: client.MeshLocationSpec{
			DisplayName: displayName,
			Description: description,
		},
	}

	createdLocation, err := r.meshLocationClient.Create(ctx, &location)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Location",
			"Could not create location, unexpected error: "+err.Error(),
		)
		return
	}

	state := locationResourceModel{
		MeshLocation: *createdLocation,
		Ref: locationRef{
			Name: createdLocation.Metadata.Name,
		},
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *locationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location, err := r.meshLocationClient.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read location '%s'", name),
			err.Error(),
		)
		return
	}

	if location == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state := locationResourceModel{
		MeshLocation: *location,
		Ref: locationRef{
			Name: location.Metadata.Name,
		},
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *locationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planName string
	var planOwnedByWorkspace string
	var planDisplayName string
	var planDescription string
	var stateName string

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &planName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &planOwnedByWorkspace)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("display_name"), &planDisplayName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("description"), &planDescription)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &stateName)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location := client.MeshLocationCreate{
		ApiVersion: "v1",
		Metadata: client.MeshLocationCreateMetadata{
			Name:             planName,
			OwnedByWorkspace: planOwnedByWorkspace,
		},
		Spec: client.MeshLocationSpec{
			DisplayName: planDisplayName,
			Description: planDescription,
		},
	}

	updatedLocation, err := r.meshLocationClient.Update(ctx, stateName, &location)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Location",
			"Could not update location, unexpected error: "+err.Error(),
		)
		return
	}

	state := locationResourceModel{
		MeshLocation: *updatedLocation,
		Ref: locationRef{
			Name: updatedLocation.Metadata.Name,
		},
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *locationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.meshLocationClient.Delete(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete location '%s'", name),
			err.Error(),
		)
		return
	}
}

func (r *locationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("name"), req, resp)
}
