package provider

import (
	"context"

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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectGroupBindingResource{}
	_ resource.ResourceWithConfigure   = &projectGroupBindingResource{}
	_ resource.ResourceWithImportState = &projectGroupBindingResource{}
)

// NewProjectGroupBindingResource is a helper function to simplify the provider implementation.
func NewProjectGroupBindingResource() resource.Resource {
	return &projectGroupBindingResource{}
}

// projectGroupBindingResource is the resource implementation.
type projectGroupBindingResource struct {
	MeshProjectGroupBinding client.MeshProjectGroupBindingClient
}

// Metadata returns the resource type name.
func (r *projectGroupBindingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_group_binding"
}

// Configure adds the provider configured client to the resource.
func (r *projectGroupBindingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.MeshProjectGroupBinding = client.ProjectGroupBinding
	})...)
}

// Schema defines the schema for the resource.
func (r *projectGroupBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Project group binding assigns a group with a specific role to a project.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Project group binding datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v3"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshProjectGroupBinding`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshProjectGroupBinding"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshProjectGroupBinding"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Project group binding metadata.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "The name identifies the binding and must be unique across the meshStack.",
						Required:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 45),
						},
					},
				},
			},

			"role_ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Selects the role to use for this project binding.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
				},
			},

			"target_ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Selects the project to which this binding applies.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Project identifier.",
						Required:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of workspace containing the target project.",
						Required:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
				},
			},

			"subject": schema.SingleNestedAttribute{
				MarkdownDescription: "Selects the group for this binding.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Groupname.",
						Required:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectGroupBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan client.MeshProjectGroupBinding

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.MeshProjectGroupBinding.Create(&plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project group binding",
			"Could not create project group binding, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, binding)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *projectGroupBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.MeshProjectGroupBinding.Read(name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read project group binding", err.Error())
	}

	if binding == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectGroupBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Project group bindings can't be updated", "Unsupported operation: project group bindings can't be updated.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectGroupBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.MeshProjectGroupBinding.Delete(name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project group binding",
			"Could not delete project group binding, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *projectGroupBindingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("name"), req.ID)...)
}
