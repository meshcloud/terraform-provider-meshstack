package provider

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &workspaceGroupBindingResource{}
	_ resource.ResourceWithConfigure   = &workspaceGroupBindingResource{}
	_ resource.ResourceWithImportState = &workspaceGroupBindingResource{}
)

// NewWorkspaceGroupBindingResource is a helper function to simplify the provider implementation.
func NewWorkspaceGroupBindingResource() resource.Resource {
	return &workspaceGroupBindingResource{}
}

// workspaceGroupBindingResource is the resource implementation.
type workspaceGroupBindingResource struct {
	MeshWorkspaceGroupBinding client.MeshWorkspaceGroupBindingClient
}

// Metadata returns the resource type name.
func (r *workspaceGroupBindingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_group_binding"
}

// Configure adds the provider configured client to the resource.
func (r *workspaceGroupBindingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.MeshWorkspaceGroupBinding = client.WorkspaceGroupBinding
	})...)
}

// Schema defines the schema for the resource.
func (r *workspaceGroupBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workspace group binding assigns a group with a specific role to a workspace.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Workspace group binding datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v2"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshWorkspaceGroupBinding`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshWorkspaceGroupBinding"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshWorkspaceGroupBinding"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Workspace group binding metadata.",
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
				MarkdownDescription: "Selects the role to use for this workspace binding.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
				},
			},

			"target_ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Selects the workspace to which this binding applies.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Workspace identifier.",
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
func (r *workspaceGroupBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan client.MeshWorkspaceGroupBinding

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.MeshWorkspaceGroupBinding.Create(&plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workspace group binding",
			"Could not create workspace group binding, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, binding)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *workspaceGroupBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.MeshWorkspaceGroupBinding.Read(name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read workspace group binding", err.Error())
	}

	if binding == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *workspaceGroupBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Workspace group bindings can't be updated", "Unsupported operation: workspace group bindings can't be updated.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *workspaceGroupBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.MeshWorkspaceGroupBinding.Delete(name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting workspace group binding",
			"Could not delete workspace group binding, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *workspaceGroupBindingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("name"), req.ID)...)
}
