package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &workspaceResource{}
	_ resource.ResourceWithConfigure   = &workspaceResource{}
	_ resource.ResourceWithImportState = &workspaceResource{}
)

// NewWorkspaceResource is a helper function to simplify the provider implementation.
func NewWorkspaceResource() resource.Resource {
	return &workspaceResource{}
}

// workspaceResource is the resource implementation.
type workspaceResource struct {
	MeshWorkspace client.MeshWorkspaceClient
}

// Metadata returns the resource type name.
func (r *workspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

// Configure adds the provider configured client to the resource.
func (r *workspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.MeshWorkspace = client.Workspace
	})...)
}

func (r *workspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack workspace.\n\n~> **Note:** Managing workspaces requires an API key with sufficient admin permissions.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Workspace datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v2"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshWorkspace`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshWorkspace"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Workspace identifier.",
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
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation date of the workspace.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Deletion date of the workspace.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags of the workspace.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Optional:            true,
						Computed:            true,
						Default:             mapdefault.StaticValue(types.MapValueMust(types.ListType{ElemType: types.StringType}, map[string]attr.Value{})),
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the workspace.",
						Required:            true,
					},
					"platform_builder_access_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether platform builder access is enabled for the workspace.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
				},
			},
		},
	}
}

func (r *workspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	workspace := client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &workspace.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &workspace.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &workspace.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("tags"), &workspace.Metadata.Tags)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createdWorkspace, err := r.MeshWorkspace.Create(ctx, &workspace)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Workspace",
			"Could not create workspace, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, createdWorkspace)...)
}

func (r *workspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	workspace, err := r.MeshWorkspace.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read workspace '%s'", name),
			err.Error(),
		)
		return
	}

	if workspace == nil {
		// The workspace was deleted outside of Terraform, so we remove it from the state
		resp.State.RemoveResource(ctx)
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, workspace)...)
}

func (r *workspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	workspace := client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &workspace.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &workspace.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &workspace.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("tags"), &workspace.Metadata.Tags)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updatedWorkspace, err := r.MeshWorkspace.Update(ctx, workspace.Metadata.Name, &workspace)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Workspace",
			"Could not update workspace, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, updatedWorkspace)...)
}

func (r *workspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.MeshWorkspace.Delete(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete workspace '%s'", name),
			err.Error(),
		)
		return
	}
}

func (r *workspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("name"), req, resp)
}
