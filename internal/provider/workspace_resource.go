package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/meshcloud/meshstack-cli/client"
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
	meshWorkspaceClient client.MeshWorkspaceClient
}

// Metadata returns the resource type name.
func (r *workspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

// Configure adds the provider configured client to the resource.
func (r *workspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshWorkspaceClient = client.Workspace
	})...)
}

func (r *workspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack workspace.\n\n" +
			"~> **Note:** Managing workspaces requires an API key with sufficient admin permissions.\n\n" +
			"~> **Tag Management:** Manage a workspace's tags inline via `metadata.tags` here. This is the recommended " +
			"approach and the only one that can set tags that are mandatory at workspace creation. The dedicated " +
			"`meshstack_workspace_tags` / `meshstack_workspace_tag` resources exist for workspaces this configuration " +
			"does not create, and carry caveats (full-workspace rewrites, races, provisional schema) documented on those " +
			"resources — reach for them only when necessary. Whichever approach you choose, use exactly one per " +
			"workspace; **do not mix inline `tags` with the dedicated tag resources**, as doing so causes state drift and " +
			"plan conflicts.",

		Attributes: map[string]schema.Attribute{
			"ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.Workspace, Description: "Reference to this workspace, can be used as `target_ref` in building block resources.", Output: true}),

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
					"tags": tagsAttribute(tagsOptions{Kind: client.MeshObjectKind.Workspace}),
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
						MarkdownDescription: "When enabled, you can open the platform builder in meshPanel while visiting this workspace.",
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
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &workspace.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &workspace.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("tags"), &workspace.Metadata.Tags)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createdWorkspace, err := r.meshWorkspaceClient.Create(ctx, &workspace)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Workspace",
			"Could not create workspace, unexpected error: "+err.Error(),
		)
		return
	}

	// Keep the tags the user declared rather than what the API echoes back: it may carry entries the
	// caller never sent (injected restricted-tag defaults) and returns no entry at all for a tag declared
	// with an empty value list, either of which would break plan/apply consistency. Mirrors the project /
	// landing zone resources.
	createdWorkspace.Metadata.Tags = workspace.Metadata.Tags

	resp.Diagnostics.Append(resp.State.Set(ctx, newWorkspaceModel(createdWorkspace))...)
}

func (r *workspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	workspace, err := r.meshWorkspaceClient.Read(ctx, name)
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

	// Keep only the tags we already track. The API may return entries the caller never sent and cannot
	// manage (injected restricted-tag defaults), so mirroring it verbatim would surface as drift. On
	// import there is no prior state (tags is null); we keep the full set so a normal import round-trips.
	workspace.Metadata.Tags = reconcileTrackedTags(ctx, req.State, path.Root("metadata").AtName("tags"), workspace.Metadata.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, newWorkspaceModel(workspace))...)
}

func (r *workspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	workspace := client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &workspace.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &workspace.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("tags"), &workspace.Metadata.Tags)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updatedWorkspace, err := r.meshWorkspaceClient.Update(ctx, workspace.Metadata.Name, &workspace)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Workspace",
			"Could not update workspace, unexpected error: "+err.Error(),
		)
		return
	}

	// Keep the tags the user declared rather than the superset the API returns, mirroring Create.
	updatedWorkspace.Metadata.Tags = workspace.Metadata.Tags

	resp.Diagnostics.Append(resp.State.Set(ctx, newWorkspaceModel(updatedWorkspace))...)
}

func (r *workspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.meshWorkspaceClient.Delete(ctx, name)
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
