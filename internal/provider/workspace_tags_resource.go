package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ resource.Resource                = &workspaceTagsResource{}
	_ resource.ResourceWithConfigure   = &workspaceTagsResource{}
	_ resource.ResourceWithImportState = &workspaceTagsResource{}
)

func NewWorkspaceTagsResource() resource.Resource {
	return &workspaceTagsResource{}
}

type workspaceTagsResource struct {
	meshWorkspaceClient client.MeshWorkspaceClient
}

type workspaceTagsMetadata struct {
	WorkspaceIdentifier types.String `tfsdk:"workspace_identifier"`
}

type workspaceTagsSpec struct {
	Tags types.Map `tfsdk:"tags"`
}

type workspaceTagsModel struct {
	Metadata workspaceTagsMetadata `tfsdk:"metadata"`
	Spec     workspaceTagsSpec     `tfsdk:"spec"`
}

func (r *workspaceTagsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_tags"
}

func (r *workspaceTagsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(c client.Client) {
		r.meshWorkspaceClient = c.Workspace
	})...)
}

func (r *workspaceTagsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Authoritatively manages all tags for a meshStack workspace.\n\n" +
			workspaceTagCaveats + "\n\n" +
			"~> **Note:** This resource is authoritative on write: every apply replaces **all** tags on the target " +
			"workspace with the ones configured here, and destroying it removes them all. It is not authoritative on " +
			"read — refresh only tracks the keys you configure, so a tag added under another key outside Terraform is " +
			"not reported as drift, and is removed by the next apply that writes this resource. This is deliberate: " +
			"meshStack injects defaults for restricted tag definitions into every workspace, and adopting those would " +
			"produce a plan that never converges. Do not mix `meshstack_workspace_tags` with inline `tags` on " +
			"`meshstack_workspace` or with `meshstack_workspace_tag` resources on the same workspace.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Workspace tags metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"workspace_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Workspace tags specification.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"tags": schema.MapAttribute{
						MarkdownDescription: "All tags to set on the workspace. This map is authoritative: every apply writes " +
							"exactly these tags, so any tag not listed here is removed. Only these keys are tracked on refresh — " +
							"tags added under other keys outside Terraform do not show up as drift.",
						ElementType: types.ListType{ElemType: types.StringType},
						Required:    true,
					},
				},
			},
		},
	}
}

func (r *workspaceTagsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workspaceTagsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := plan.Metadata.WorkspaceIdentifier.ValueString()
	workspace, err := r.meshWorkspaceClient.Read(ctx, wsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Could not read workspace '%s': %v", wsName, err))
		return
	}
	if workspace == nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Workspace '%s' not found", wsName))
		return
	}

	tags := make(map[string][]string)
	if !plan.Spec.Tags.IsNull() {
		resp.Diagnostics.Append(plan.Spec.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updatePayload := client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{Name: wsName, Tags: tags},
		Spec:     workspace.Spec,
	}
	if _, err := r.meshWorkspaceClient.Update(ctx, wsName, &updatePayload); err != nil {
		resp.Diagnostics.AddError("Error Updating Workspace Tags", fmt.Sprintf("Could not update tags for workspace '%s': %v", wsName, err))
		return
	}

	// Keep the tags the user declared rather than the superset the API returns (an entry for every
	// defined tag property plus injected restricted-tag defaults), which would break plan/apply
	// consistency on the Required spec.tags. Mirrors workspace_resource.go.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *workspaceTagsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workspaceTagsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := state.Metadata.WorkspaceIdentifier.ValueString()
	workspace, err := r.meshWorkspaceClient.Read(ctx, wsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Could not read workspace '%s': %v", wsName, err))
		return
	}
	if workspace == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Track only the keys this resource declares. The API returns a superset — an entry for every
	// defined tag property (empty list when unset) plus the defaults meshStack injects for restricted
	// tag definitions — and adopting those would produce a plan that wants to delete a tag the backend
	// re-injects on the next apply, forever. The trade-off: a tag added under an untracked key outside
	// Terraform is not reported as drift; the next apply removes it, because Update writes exactly the
	// declared tags. Mirrors workspace_resource.go.
	tags := reconcileTrackedTags(ctx, req.State, path.Root("spec").AtName("tags"), workspace.Metadata.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// spec.tags is Required and must never be null; a workspace with no tags at all yields a nil map.
	if tags == nil {
		tags = map[string][]string{}
	}

	tagsMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, tags)
	resp.Diagnostics.Append(diags...)
	state.Spec.Tags = tagsMap

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *workspaceTagsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workspaceTagsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := plan.Metadata.WorkspaceIdentifier.ValueString()
	workspace, err := r.meshWorkspaceClient.Read(ctx, wsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Could not read workspace '%s': %v", wsName, err))
		return
	}
	if workspace == nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Workspace '%s' not found", wsName))
		return
	}

	tags := make(map[string][]string)
	if !plan.Spec.Tags.IsNull() {
		resp.Diagnostics.Append(plan.Spec.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updatePayload := client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{Name: wsName, Tags: tags},
		Spec:     workspace.Spec,
	}
	if _, err := r.meshWorkspaceClient.Update(ctx, wsName, &updatePayload); err != nil {
		resp.Diagnostics.AddError("Error Updating Workspace Tags", fmt.Sprintf("Could not update tags for workspace '%s': %v", wsName, err))
		return
	}

	// Keep the declared tags rather than the API's superset, mirroring Create.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *workspaceTagsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workspaceTagsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := state.Metadata.WorkspaceIdentifier.ValueString()
	workspace, err := r.meshWorkspaceClient.Read(ctx, wsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Could not read workspace '%s': %v", wsName, err))
		return
	}
	if workspace == nil {
		// Workspace already gone; all tags are implicitly deleted.
		return
	}

	// Authoritative delete removes all tags by updating workspace with an empty tags map.
	updatePayload := client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{Name: wsName, Tags: make(map[string][]string)},
		Spec:     workspace.Spec,
	}
	_, err = r.meshWorkspaceClient.Update(ctx, wsName, &updatePayload)
	if err != nil {
		resp.Diagnostics.AddError("Error Clearing Workspace Tags", fmt.Sprintf("Could not clear tags for workspace '%s': %v", wsName, err))
		return
	}
}

func (r *workspaceTagsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("workspace_identifier"), req.ID)...)

	// Leave spec.tags null rather than empty: Read passes the API's tags through unchanged when there is
	// no prior state, so an import round-trips every tag on the workspace instead of reconciling against
	// an empty tracked set and importing nothing. Setting the attribute — even to null — materialises the
	// spec object, which the null-into-value-struct decode in Read requires.
	nullTags := types.MapNull(types.ListType{ElemType: types.StringType})
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("tags"), nullTags)...)
}
