package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &workspaceUserBindingResource{}
	_ resource.ResourceWithConfigure   = &workspaceUserBindingResource{}
	_ resource.ResourceWithImportState = &workspaceUserBindingResource{}
)

// NewWorkspaceUserBindingResource is a helper function to simplify the provider implementation.
func NewWorkspaceUserBindingResource() resource.Resource {
	return &workspaceUserBindingResource{}
}

// workspaceUserBindingResource is the resource implementation.
type workspaceUserBindingResource struct {
	client *client.MeshStackProviderClient
}

// Metadata returns the resource type name.
func (r *workspaceUserBindingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_user_binding"
}

// Configure adds the provider configured client to the resource.
func (r *workspaceUserBindingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema defines the schema for the resource.
func (r *workspaceUserBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workspace user binding assigns a user with a specific role to a workspace.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Workspace user binding datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v3"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshWorkspaceUserBinding`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshWorkspaceUserBinding"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshWorkspaceUserBinding"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Workspace user binding metadata.",
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
				MarkdownDescription: "Selects the user for this binding.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Username.",
						Required:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *workspaceUserBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan client.MeshWorkspaceUserBinding

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.client.CreateWorkspaceUserBinding(&plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workspace user binding",
			"Could not create workspace user binding, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, binding)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *workspaceUserBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.client.ReadWorkspaceUserBinding(name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read workspace user binding", err.Error())
	}

	if binding == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *workspaceUserBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Workspace user bindings can't be updated", "Unsupported operation: workspace user bindings can't be updated.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *workspaceUserBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWorkspaceUserBinding(name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting workspace user binding",
			"Could not delete workspace, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *workspaceUserBindingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("name"), req.ID)...)
}
