package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ resource.Resource                = &workspaceUserGroupMemberResource{}
	_ resource.ResourceWithConfigure   = &workspaceUserGroupMemberResource{}
	_ resource.ResourceWithImportState = &workspaceUserGroupMemberResource{}
)

func NewWorkspaceUserGroupMemberResource() resource.Resource {
	return &workspaceUserGroupMemberResource{}
}

type workspaceUserGroupMemberResource struct {
	meshWorkspaceUserGroupClient client.MeshWorkspaceUserGroupClient
}

type workspaceUserGroupMemberModel struct {
	GroupRef workspaceScopedRef `tfsdk:"group_ref"`
	UserRef  client.NamedRef    `tfsdk:"user_ref"`
}

func (r *workspaceUserGroupMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_user_group_member"
}

func (r *workspaceUserGroupMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshWorkspaceUserGroupClient = client.WorkspaceUserGroup
	})...)
}

func (r *workspaceUserGroupMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "One member of a workspace user group. Use it to add a user to a group whose other members " +
			"are managed elsewhere, for example in meshPanel or in another Terraform configuration.\n\n" +
			"~> **Do not combine with `spec.members`.** If `meshstack_workspace_user_group` sets `spec.members` for the same " +
			"group, each apply of one resource removes the members the other adds.\n\n" +
			"~> **The whole group is written on every change.** The meshObject API has no endpoint for a single member, so " +
			"each create and delete reads the group and writes it back with the member added or removed. The provider " +
			"applies the changes to one group one after another within an apply, but a change to the group by anything " +
			"else between that read and write is lost.",

		Attributes: map[string]schema.Attribute{
			"group_ref": meshWorkspaceScopedRef(client.MeshObjectKind.WorkspaceUserGroup, meshRefOptions{
				Description:     "Reference to the workspace user group, such as `meshstack_workspace_user_group.example.ref`.",
				RequiresReplace: true,
			}),
			"user_ref": meshRefByName(meshRefOptions{
				Kind:            client.MeshObjectKind.User,
				Description:     "Reference to the user who becomes a member, such as `data.meshstack_user.example.ref`.",
				RequiresReplace: true,
			}),
		},
	}
}

func (r *workspaceUserGroupMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := generic.Get[workspaceUserGroupMemberModel](ctx, req.Plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.changeMembers(ctx, plan.GroupRef, true, &resp.Diagnostics, func(members []string) []string {
		if slices.Contains(members, plan.UserRef.Name) {
			return members
		}
		return append(members, plan.UserRef.Name)
	})
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan)...)
}

func (r *workspaceUserGroupMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := generic.Get[workspaceUserGroupMemberModel](ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.meshWorkspaceUserGroupClient.Read(ctx, client.NewMeshWorkspaceUserGroupId(state.GroupRef.OwnedByWorkspace, state.GroupRef.Name))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read workspace user group '%s'", client.NewMeshWorkspaceUserGroupId(state.GroupRef.OwnedByWorkspace, state.GroupRef.Name)),
			err.Error(),
		)
		return
	}
	if group == nil || !slices.Contains(group.Spec.Members, state.UserRef.Name) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, state)...)
}

func (r *workspaceUserGroupMemberResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Workspace user group member cannot be updated",
		"Every attribute of meshstack_workspace_user_group_member forces a new resource, so Terraform never updates one in place.")
}

func (r *workspaceUserGroupMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := generic.Get[workspaceUserGroupMemberModel](ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.changeMembers(ctx, state.GroupRef, false, &resp.Diagnostics, func(members []string) []string {
		return slices.DeleteFunc(members, func(member string) bool { return member == state.UserRef.Name })
	})
}

func (r *workspaceUserGroupMemberResource) changeMembers(ctx context.Context, groupRef workspaceScopedRef, groupRequired bool, diags *diag.Diagnostics, change func([]string) []string) {
	identifier := client.NewMeshWorkspaceUserGroupId(groupRef.OwnedByWorkspace, groupRef.Name)
	defer lockWorkspaceUserGroup(groupRef.OwnedByWorkspace, groupRef.Name)()

	group, err := r.meshWorkspaceUserGroupClient.Read(ctx, identifier)
	switch {
	case err != nil:
		diags.AddError(fmt.Sprintf("Could not read workspace user group '%s'", identifier), err.Error())
		return
	case group == nil && groupRequired:
		diags.AddError(fmt.Sprintf("Workspace user group '%s' not found", identifier),
			fmt.Sprintf("No workspace user group '%s' exists in workspace '%s', or your API key cannot see it.", groupRef.Name, groupRef.OwnedByWorkspace))
		return
	case group == nil:
		return
	}

	members := change(slices.Clone(group.Spec.Members))
	if slices.Equal(members, group.Spec.Members) {
		return
	}
	group.Spec.Members = members
	group.Status = nil
	if _, err := r.meshWorkspaceUserGroupClient.Update(ctx, *group); err != nil {
		diags.AddError(fmt.Sprintf("Could not change the members of workspace user group '%s'", identifier), err.Error())
	}
}

func (r *workspaceUserGroupMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ".", 3)
	if len(parts) != 3 || slices.Contains(parts, "") {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace.group.username. Got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, workspaceUserGroupMemberModel{
		GroupRef: newWorkspaceUserGroupRef(parts[0], parts[1]),
		UserRef:  client.NamedRef{Kind: client.MeshObjectKind.User, Name: parts[2]},
	})...)
}
