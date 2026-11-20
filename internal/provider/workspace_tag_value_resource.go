package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ resource.Resource                = &workspaceTagValueResource{}
	_ resource.ResourceWithConfigure   = &workspaceTagValueResource{}
	_ resource.ResourceWithImportState = &workspaceTagValueResource{}
)

func NewWorkspaceTagValueResource() resource.Resource {
	return &workspaceTagValueResource{}
}

type workspaceTagValueResource struct {
	meshWorkspaceClient client.MeshWorkspaceClient
}

type workspaceTagValueMetadata struct {
	WorkspaceIdentifier types.String `tfsdk:"workspace_identifier"`
	Key                 types.String `tfsdk:"key"`
}

type workspaceTagValueSpec struct {
	Value types.String `tfsdk:"value"`
}

type workspaceTagValueModel struct {
	Metadata workspaceTagValueMetadata `tfsdk:"metadata"`
	Spec     workspaceTagValueSpec     `tfsdk:"spec"`
}

func (r *workspaceTagValueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_tag_value"
}

func (r *workspaceTagValueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(c client.Client) {
		r.meshWorkspaceClient = c.Workspace
	})...)
}

func (r *workspaceTagValueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = tagResourceSchema(
		"Manages a single value of a multi-select tag on a meshStack workspace.\n\n"+
			workspaceTagCaveatsFor("meshstack_workspace_tag_value")+"\n\n"+
			tagValueNote("workspace"),
		"Workspace tag value",
		map[string]schema.Attribute{
			"workspace_identifier": requiredReplaceOnChangeString("Identifier of the target workspace."),
			"key":                  requiredReplaceOnChangeString("Key of the tag."),
		},
		map[string]schema.Attribute{"value": requiredReplaceOnChangeString("Value to add to the tag.")},
	)
}

func (r *workspaceTagValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workspaceTagValueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := plan.Metadata.WorkspaceIdentifier.ValueString()
	key := plan.Metadata.Key.ValueString()
	value := plan.Spec.Value.ValueString()

	defer lockWorkspace(wsName)()
	workspace := readTaggedWorkspace(ctx, r.meshWorkspaceClient, wsName, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if workspace == nil {
		resp.Diagnostics.AddError("Error Reading Workspace", fmt.Sprintf("Workspace '%s' not found", wsName))
		return
	}

	if tags, changed := withTagValue(workspace.Metadata.Tags, key, value); changed {
		if err := writeWorkspaceTags(ctx, r.meshWorkspaceClient, workspace, tags); err != nil {
			resp.Diagnostics.AddError("Error Adding Workspace Tag Value", fmt.Sprintf("Could not add value '%s' to tag '%s' on workspace '%s': %v", value, key, wsName, err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *workspaceTagValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workspaceTagValueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspace := readTaggedWorkspace(ctx, r.meshWorkspaceClient, state.Metadata.WorkspaceIdentifier.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if workspace == nil || !slices.Contains(workspace.Metadata.Tags[state.Metadata.Key.ValueString()], state.Spec.Value.ValueString()) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *workspaceTagValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workspaceTagValueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *workspaceTagValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workspaceTagValueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsName := state.Metadata.WorkspaceIdentifier.ValueString()
	key := state.Metadata.Key.ValueString()
	value := state.Spec.Value.ValueString()

	defer lockWorkspace(wsName)()
	workspace := readTaggedWorkspace(ctx, r.meshWorkspaceClient, wsName, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || workspace == nil {
		return
	}

	if tags, changed := withoutTagValue(workspace.Metadata.Tags, key, value); changed {
		if err := writeWorkspaceTags(ctx, r.meshWorkspaceClient, workspace, tags); err != nil {
			resp.Diagnostics.AddError("Error Removing Workspace Tag Value", fmt.Sprintf("Could not remove value '%s' from tag '%s' on workspace '%s': %v", value, key, wsName, err))
		}
	}
}

func (r *workspaceTagValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids, key, value, ok := parseTagValueImportID(req.ID, 1)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace_identifier.key=value. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("workspace_identifier"), ids[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("key"), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("value"), value)...)
}

func writeWorkspaceTags(ctx context.Context, workspaces client.MeshWorkspaceClient, workspace *client.MeshWorkspace, tags map[string][]string) error {
	_, err := workspaces.Update(ctx, workspace.Metadata.Name, &client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{Name: workspace.Metadata.Name, Tags: tags},
		Spec:     workspace.Spec,
	})
	return err
}

func tagValueNote(object string) string {
	return "~> **Note:** Use one `meshstack_" + object + "_tag_value` resource per value. The resource adds its value to the " +
		"tag and removes only that value again, so several of them can share a tag — but two resources declaring the same " +
		"value share it, too: destroying either removes it, and the other drops out of state and adds it again on the next " +
		"apply. A value that is already on the tag when this resource is created is adopted, and destroying the resource " +
		"removes it. meshStack decides which values a tag accepts; adding a second value to a tag that is not multi-select " +
		"fails when it rejects the write. Do not combine this resource with `meshstack_" + object + "_tag` for the same " +
		"key, or with `meshstack_" + object + "_tags` or inline `tags` on `meshstack_" + object + "` for the same " + object + "."
}
