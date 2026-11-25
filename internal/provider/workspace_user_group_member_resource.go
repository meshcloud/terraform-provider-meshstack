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

type workspaceUserGroupMemberMetadataModel struct {
	WorkspaceIdentifier string `tfsdk:"workspace_identifier"`
	GroupIdentifier     string `tfsdk:"group_identifier"`
	Username            string `tfsdk:"username"`
}

type workspaceUserGroupMemberModel struct {
	Metadata workspaceUserGroupMemberMetadataModel `tfsdk:"metadata"`
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
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "The group and the user.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"workspace_identifier": requiredReplaceOnChangeString("Identifier of the workspace that owns the group."),
					"group_identifier":     requiredReplaceOnChangeString("Identifier (`metadata.name`) of the workspace user group."),
					"username":             requiredReplaceOnChangeString("Username of the member. It must belong to an existing meshUser."),
				},
			},
		},
	}
}

func (r *workspaceUserGroupMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workspaceUserGroupMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.changeMembers(ctx, plan.Metadata, true, &resp.Diagnostics, func(members []string) []string {
		if slices.Contains(members, plan.Metadata.Username) {
			return members
		}
		return append(members, plan.Metadata.Username)
	})
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *workspaceUserGroupMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workspaceUserGroupMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.meshWorkspaceUserGroupClient.Read(ctx, client.NewMeshWorkspaceUserGroupId(state.Metadata.WorkspaceIdentifier, state.Metadata.GroupIdentifier))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read workspace user group '%s'", client.NewMeshWorkspaceUserGroupId(state.Metadata.WorkspaceIdentifier, state.Metadata.GroupIdentifier)),
			err.Error(),
		)
		return
	}
	if group == nil || !slices.Contains(group.Spec.Members, state.Metadata.Username) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *workspaceUserGroupMemberResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Workspace user group member cannot be updated",
		"Every attribute of meshstack_workspace_user_group_member forces a new resource, so Terraform never updates one in place.")
}

func (r *workspaceUserGroupMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workspaceUserGroupMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.changeMembers(ctx, state.Metadata, false, &resp.Diagnostics, func(members []string) []string {
		return slices.DeleteFunc(members, func(member string) bool { return member == state.Metadata.Username })
	})
}

func (r *workspaceUserGroupMemberResource) changeMembers(ctx context.Context, metadata workspaceUserGroupMemberMetadataModel, groupRequired bool, diags *diag.Diagnostics, change func([]string) []string) {
	identifier := client.NewMeshWorkspaceUserGroupId(metadata.WorkspaceIdentifier, metadata.GroupIdentifier)
	defer lockWorkspaceUserGroup(metadata.WorkspaceIdentifier, metadata.GroupIdentifier)()

	group, err := r.meshWorkspaceUserGroupClient.Read(ctx, identifier)
	switch {
	case err != nil:
		diags.AddError(fmt.Sprintf("Could not read workspace user group '%s'", identifier), err.Error())
		return
	case group == nil && groupRequired:
		diags.AddError(fmt.Sprintf("Workspace user group '%s' not found", identifier),
			fmt.Sprintf("No workspace user group '%s' exists in workspace '%s', or your API key cannot see it.", metadata.GroupIdentifier, metadata.WorkspaceIdentifier))
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
	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceUserGroupMemberModel{
		Metadata: workspaceUserGroupMemberMetadataModel{WorkspaceIdentifier: parts[0], GroupIdentifier: parts[1], Username: parts[2]},
	})...)
}
