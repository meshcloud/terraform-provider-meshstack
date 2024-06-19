package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &projectUserBindingResource{}
	_ resource.ResourceWithConfigure   = &projectUserBindingResource{}
	_ resource.ResourceWithImportState = &projectUserBindingResource{}
)

// NewProjectUserBindingResource is a helper function to simplify the provider implementation.
func NewProjectUserBindingResource() resource.Resource {
	return &projectUserBindingResource{}
}

// projectUserBindingResource is the resource implementation.
type projectUserBindingResource struct {
	client *MeshStackProviderClient
}

// Metadata returns the resource type name.
func (r *projectUserBindingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_user_binding"
}

// Configure adds the provider configured client to the resource.
func (r *projectUserBindingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*MeshStackProviderClient)

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
func (r *projectUserBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Project user binding assigns a user with a specific role to a project.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Project user binding datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v3"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshProjectUserBinding`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshProjectUserBinding"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshProjectUserBinding"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Project user binding metadata.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "The name identifies the binding and must be unique across the meshStack. A UUID will automatically be used if left unset.",
						Optional:            true,
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

type projectUserBindingCreate struct {
	Metadata struct {
		Name *string `tfsdk:"name"`
	} `tfsdk:"metadata"`

	RoleRef struct {
		Name string `tfsdk:"name"`
	} `tfsdk:"role_ref"`

	TargetRef struct {
		Name             string `tfsdk:"name"`
		OwnedByWorkspace string `tfsdk:"owned_by_workspace"`
	} `tfsdk:"target_ref"`

	Subject struct {
		Name string `tfsdk:"name"`
	} `tfsdk:"subject"`
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectUserBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MeshProjectUserBinding

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.client.CreateProjectUserBinding(&plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project user binding",
			"Could not create project user binding, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, binding)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *projectUserBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.client.ReadProjectUserBinding(name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read project user binding", err.Error())
	}

	if binding == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectUserBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Project user bindings can't be updated", "Unsupported operation: project user bindings can't be updated.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectUserBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProjecUserBinding(name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			"Could not delete project, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *projectUserBindingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("name"), req.ID)...)
}
