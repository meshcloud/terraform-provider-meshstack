package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ resource.Resource                = &workspaceTagResource{}
	_ resource.ResourceWithConfigure   = &workspaceTagResource{}
	_ resource.ResourceWithImportState = &workspaceTagResource{}
)

var workspaceTagCaveats = tagResourceCaveats(taggedObject{
	name:              "workspace",
	kind:              "meshWorkspace",
	ownerResource:     "meshstack_workspace",
	inlineTags:        "metadata.tags",
	tagResource:       "meshstack_workspace_tag",
	tagsResource:      "meshstack_workspace_tags",
	injectedDefaultOn: "a workspace created in meshPanel",
})

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
	resp.Schema = tagResourceSchema(
		"Manages a single dedicated tag for a meshStack workspace.\n\n"+
			workspaceTagCaveats+"\n\n"+
			"~> **Note:** Do not mix dedicated `meshstack_workspace_tag` resources with authoritative "+
			"`meshstack_workspace_tags` or inline `tags` on `meshstack_workspace` for the same workspace. This resource only "+
			"manages its own key; tags under other keys are read and written back unchanged.",
		"Workspace tag",
		map[string]schema.Attribute{
			"workspace_identifier": requiredReplaceOnChangeString("Identifier of the target workspace."),
			"key":                  requiredReplaceOnChangeString("Key of the tag."),
		},
		map[string]schema.Attribute{"values": tagValuesAttribute()},
	)
}

func (r *workspaceTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.setTag(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *workspaceTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workspaceTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := state.Metadata.WorkspaceIdentifier.ValueString()
	key := state.Metadata.Key.ValueString()

	workspace := readTaggedWorkspace(ctx, r.meshWorkspaceClient, wsName, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if workspace == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	values, found, diags := trackedTagValues(ctx, workspace.Metadata.Tags, key, state.Spec.Values)
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(diags...)
	state.Spec.Values = values

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *workspaceTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.setTag(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *workspaceTagResource) setTag(ctx context.Context, tfPlan tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan workspaceTagModel
	diags.Append(tfPlan.Get(ctx, &plan)...)
	if diags.HasError() {
		return
	}

	wsName := plan.Metadata.WorkspaceIdentifier.ValueString()
	key := plan.Metadata.Key.ValueString()

	defer lockWorkspace(wsName)()
	workspace := readTaggedWorkspace(ctx, r.meshWorkspaceClient, wsName, diags)
	if diags.HasError() {
		return
	}
	if workspace == nil {
		diags.AddError("Error Reading Workspace", fmt.Sprintf("Workspace '%s' not found", wsName))
		return
	}

	var values []string
	diags.Append(plan.Spec.Values.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
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
		diags.AddError("Error Updating Workspace Tag", fmt.Sprintf("Could not set tag '%s' on workspace '%s': %v", key, wsName, err))
		return
	}

	// Keep the values the user declared rather than what the API echoes back. The backend may
	// normalize or default them, and writing that into the Required spec.values would break
	// plan/apply consistency. Mirrors workspace_resource.go.
	diags.Append(state.Set(ctx, &plan)...)
}

func (r *workspaceTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workspaceTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := state.Metadata.WorkspaceIdentifier.ValueString()
	key := state.Metadata.Key.ValueString()

	defer lockWorkspace(wsName)()
	workspace := readTaggedWorkspace(ctx, r.meshWorkspaceClient, wsName, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
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
	if _, err := r.meshWorkspaceClient.Update(ctx, wsName, &updatePayload); err != nil {
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
