package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ resource.Resource                = &workspaceTagResource{}
	_ resource.ResourceWithConfigure   = &workspaceTagResource{}
	_ resource.ResourceWithImportState = &workspaceTagResource{}
)

// Both dedicated tag resources are read-modify-write wrappers around the whole meshWorkspace
// meshObject.
const workspaceTagCaveats = "!> **Not recommended for general use.** Prefer managing tags inline via `metadata.tags` on " +
	"`meshstack_workspace`. Only reach for this resource when the workspace itself is not managed by your Terraform " +
	"configuration (for example it was created in meshPanel or by another team) and you understand the " +
	"trade-offs below. All of them follow from the same limitation: the meshObject API has no endpoint for individual " +
	"tags, so every create, update and delete here reads the entire `meshWorkspace` object and writes it back with the " +
	"tags replaced.\n\n" +

	"~> **The entire workspace is rewritten on every change.** The Terraform plan only shows a tag changing, but each " +
	"apply issues a full workspace update. Everything meshStack does on a workspace update — audit-log entries, " +
	"notifications etc. — happens every time, and any workspace field changed outside Terraform between this " +
	"resource's read and its write is written back with the value it read.\n\n" +

	"~> **Tags you do not manage are written back verbatim.** Preserving a workspace's other tags is what lets " +
	"several `meshstack_workspace_tag` resources coexist, but the object read back can carry entries nobody " +
	"declared here — notably the defaults meshStack injects for restricted tag definitions on a workspace " +
	"created in meshPanel — and every write sends all of them back. So this resource writes tag entries " +
	"you never configured, and if your meshStack rejects writing a restricted tag's value, the update fails and no " +
	"tag on that workspace can be managed with this resource — use inline `metadata.tags` on `meshstack_workspace` " +
	"instead. `meshstack_workspace_tags` is unaffected: it sends exactly the tags you configure.\n\n" +

	"~> **Race conditions.** The read-modify-write cycle is not atomic and the API offers no optimistic locking. A " +
	"concurrent write to the same workspace — another apply of these resources, the `meshstack_workspace` resource " +
	"itself, a panel user, or any other automation — can silently clobber this resource's tags or be clobbered by " +
	"them. **This includes a single apply:** Terraform walks resources that do not depend on each other in parallel " +
	"(`-parallelism`, 10 by default) and the provider does not serialize these writes, so two " +
	"`meshstack_workspace_tag` resources on the same workspace can each read the same tag map and then overwrite each " +
	"other — one tag silently goes missing while the apply reports success. Manage all tags of a workspace from a " +
	"single resource in a single Terraform state; if you must spread them across several `meshstack_workspace_tag` " +
	"resources, chain them with `depends_on` so they apply one after another. Never run two applies against the same " +
	"workspace in parallel.\n\n" +

	"~> **Cannot manage tags that are mandatory at workspace creation.** This resource can only set a tag on a " +
	"workspace that already exists, so it cannot supply tag values that meshStack requires when the workspace is " +
	"created — the `meshstack_workspace` create fails before this resource ever runs. Set mandatory tags inline via " +
	"`metadata.tags` on `meshstack_workspace`.\n\n" +

	"~> **Subject to change.** This resource is provisional. Once meshStack's meshObject API supports workspace tags " +
	"as individual meshObjects, it will be reworked in terms of that API."

func NewWorkspaceTagResource() resource.Resource {
	return &workspaceTagResource{}
}

type workspaceTagResource struct {
	meshWorkspaceClient client.MeshWorkspaceClient
}

type workspaceTagMetadata struct {
	WorkspaceIdentifier types.String `tfsdk:"workspace_identifier"`
	Key                 types.String `tfsdk:"key"`
}

type workspaceTagSpec struct {
	Values types.List `tfsdk:"values"`
}

type workspaceTagModel struct {
	Metadata workspaceTagMetadata `tfsdk:"metadata"`
	Spec     workspaceTagSpec     `tfsdk:"spec"`
}

func (r *workspaceTagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_tag"
}

func (r *workspaceTagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(c client.Client) {
		r.meshWorkspaceClient = c.Workspace
	})...)
}

func (r *workspaceTagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single dedicated tag for a meshStack workspace.\n\n" +
			workspaceTagCaveats + "\n\n" +
			"~> **Note:** Do not mix dedicated `meshstack_workspace_tag` resources with authoritative " +
			"`meshstack_workspace_tags` or inline `tags` on `meshstack_workspace` for the same workspace. This resource only " +
			"manages its own key; tags under other keys are read and written back unchanged.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Workspace tag metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"workspace_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of the target workspace.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"key": schema.StringAttribute{
						MarkdownDescription: "Key of the tag.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Workspace tag specification.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"values": schema.ListAttribute{
						MarkdownDescription: "List of values for this tag.",
						Required:            true,
						ElementType:         types.StringType,
					},
				},
			},
		},
	}
}

func (r *workspaceTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workspaceTagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := plan.Metadata.WorkspaceIdentifier.ValueString()
	key := plan.Metadata.Key.ValueString()

	workspace, err := r.meshWorkspaceClient.Read(ctx, wsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Could not read workspace '%s': %v", wsName, err))
		return
	}
	if workspace == nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Workspace '%s' not found", wsName))
		return
	}

	var values []string
	resp.Diagnostics.Append(plan.Spec.Values.ElementsAs(ctx, &values, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags := make(map[string][]string, len(workspace.Metadata.Tags)+1)
	for k, v := range workspace.Metadata.Tags {
		tags[k] = v
	}
	tags[key] = values

	updatePayload := client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{Name: wsName, Tags: tags},
		Spec:     workspace.Spec,
	}
	if _, err := r.meshWorkspaceClient.Update(ctx, wsName, &updatePayload); err != nil {
		resp.Diagnostics.AddError("Error Updating Workspace Tag", fmt.Sprintf("Could not set tag '%s' on workspace '%s': %v", key, wsName, err))
		return
	}

	// Keep the values the user declared rather than what the API echoes back. The backend may
	// normalize or default them, and writing that into the Required spec.values would break
	// plan/apply consistency. Mirrors workspace_resource.go.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *workspaceTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workspaceTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := state.Metadata.WorkspaceIdentifier.ValueString()
	key := state.Metadata.Key.ValueString()

	workspace, err := r.meshWorkspaceClient.Read(ctx, wsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Could not read workspace '%s': %v", wsName, err))
		return
	}
	if workspace == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	vals, ok := workspace.Metadata.Tags[key]
	if !ok {
		// The API cannot represent a tag with no values — a PUT carrying `{"k": []}` comes back with
		// `"tags": {}` — so for a resource whose tracked value list is empty, "absent" is that same state
		// and must not be read as a deletion, or the resource is destroyed and recreated on every apply.
		// On import there is no prior state (values is null), so an absent key really is missing.
		declaredEmpty := !state.Spec.Values.IsNull() && len(state.Spec.Values.Elements()) == 0
		if !declaredEmpty {
			resp.State.RemoveResource(ctx)
			return
		}
		vals = []string{}
	}

	valList, diags := types.ListValueFrom(ctx, types.StringType, vals)
	resp.Diagnostics.Append(diags...)
	state.Spec.Values = valList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *workspaceTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workspaceTagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := plan.Metadata.WorkspaceIdentifier.ValueString()
	key := plan.Metadata.Key.ValueString()

	workspace, err := r.meshWorkspaceClient.Read(ctx, wsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Could not read workspace '%s': %v", wsName, err))
		return
	}
	if workspace == nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Workspace '%s' not found", wsName))
		return
	}

	var values []string
	resp.Diagnostics.Append(plan.Spec.Values.ElementsAs(ctx, &values, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags := make(map[string][]string, len(workspace.Metadata.Tags)+1)
	for k, v := range workspace.Metadata.Tags {
		tags[k] = v
	}
	tags[key] = values

	updatePayload := client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{Name: wsName, Tags: tags},
		Spec:     workspace.Spec,
	}
	if _, err := r.meshWorkspaceClient.Update(ctx, wsName, &updatePayload); err != nil {
		resp.Diagnostics.AddError("Error Updating Workspace Tag", fmt.Sprintf("Could not set tag '%s' on workspace '%s': %v", key, wsName, err))
		return
	}

	// Keep the declared values rather than the API's, mirroring Create.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *workspaceTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workspaceTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := state.Metadata.WorkspaceIdentifier.ValueString()
	key := state.Metadata.Key.ValueString()

	workspace, err := r.meshWorkspaceClient.Read(ctx, wsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Could not read workspace '%s': %v", wsName, err))
		return
	}
	if workspace == nil {
		// Workspace already gone; tag is implicitly deleted.
		return
	}

	tags := make(map[string][]string, len(workspace.Metadata.Tags))
	for k, v := range workspace.Metadata.Tags {
		if k != key {
			tags[k] = v
		}
	}

	updatePayload := client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{Name: wsName, Tags: tags},
		Spec:     workspace.Spec,
	}
	_, err = r.meshWorkspaceClient.Update(ctx, wsName, &updatePayload)
	if err != nil {
		resp.Diagnostics.AddError("Error Removing Workspace Tag", fmt.Sprintf("Could not remove tag '%s' from workspace '%s': %v", key, wsName, err))
		return
	}
}

func (r *workspaceTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace_identifier.key. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("workspace_identifier"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("key"), parts[1])...)

	// Leave spec.values null rather than empty: Read distinguishes "no prior state" (import) from a
	// tracked empty list, and only the latter may treat an absent tag key as still present. Setting the
	// attribute — even to null — materialises the spec object, which the decode in Read requires.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("values"), types.ListNull(types.StringType))...)
}
