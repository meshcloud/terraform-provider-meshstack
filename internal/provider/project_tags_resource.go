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
	_ resource.Resource                = &projectTagsResource{}
	_ resource.ResourceWithConfigure   = &projectTagsResource{}
	_ resource.ResourceWithImportState = &projectTagsResource{}
)

func NewProjectTagsResource() resource.Resource {
	return &projectTagsResource{}
}

type projectTagsResource struct {
	meshProjectClient client.MeshProjectClient
}

type projectTagsMetadata struct {
	WorkspaceIdentifier types.String `tfsdk:"workspace_identifier"`
	ProjectIdentifier   types.String `tfsdk:"project_identifier"`
}

type projectTagsSpec struct {
	Tags types.Map `tfsdk:"tags"`
}

type projectTagsModel struct {
	Metadata projectTagsMetadata `tfsdk:"metadata"`
	Spec     projectTagsSpec     `tfsdk:"spec"`
}

func (r *projectTagsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_tags"
}

func (r *projectTagsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(c client.Client) {
		r.meshProjectClient = c.Project
	})...)
}

func (r *projectTagsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = tagResourceSchema(
		"Authoritatively manages all tags for a meshStack project.\n\n"+
			projectTagCaveats+"\n\n"+
			"~> **Note:** This resource is authoritative on write: every apply replaces **all** tags on the target "+
			"project with the ones configured here, and destroying it removes them all. It is not authoritative on "+
			"read — refresh only tracks the keys you configure, so a tag added under another key outside Terraform is "+
			"not reported as drift, and is removed by the next apply that writes this resource. This is deliberate: "+
			"a project carries the defaults meshStack injects for restricted tag definitions, and adopting those "+
			"would produce a plan that never converges. Note also that a tag declared with an empty value list is kept "+
			"in state, because the API returns no entry for it at all. Do not mix `meshstack_project_tags` with inline "+
			"`tags` on `meshstack_project` or with `meshstack_project_tag` resources on the same project.",
		"Project tags",
		map[string]schema.Attribute{
			"workspace_identifier": requiredReplaceOnChangeString("Identifier of the workspace that owns the project."),
			"project_identifier":   requiredReplaceOnChangeString("Identifier of the project."),
		},
		map[string]schema.Attribute{"tags": tagsMapAttribute("project")},
	)
}

func (r *projectTagsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.setTags(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *projectTagsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectTagsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := readTaggedProject(ctx, r.meshProjectClient, state.Metadata.WorkspaceIdentifier.ValueString(), state.Metadata.ProjectIdentifier.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if project == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	tags := reconcileTrackedTags(ctx, req.State, path.Root("spec").AtName("tags"), project.Spec.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if tags == nil {
		tags = map[string][]string{}
	}

	tagsMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, tags)
	resp.Diagnostics.Append(diags...)
	state.Spec.Tags = tagsMap

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectTagsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.setTags(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *projectTagsResource) setTags(ctx context.Context, tfPlan tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan projectTagsModel
	diags.Append(tfPlan.Get(ctx, &plan)...)
	if diags.HasError() {
		return
	}

	workspace := plan.Metadata.WorkspaceIdentifier.ValueString()
	name := plan.Metadata.ProjectIdentifier.ValueString()
	defer lockProject(workspace, name)()
	project := readTaggedProject(ctx, r.meshProjectClient, workspace, name, diags)
	if diags.HasError() {
		return
	}
	if project == nil {
		diags.AddError("Error Reading Project", fmt.Sprintf("Project '%s' not found in workspace '%s'", name, workspace))
		return
	}

	tags := make(map[string][]string)
	if !plan.Spec.Tags.IsNull() {
		diags.Append(plan.Spec.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return
		}
	}

	if err := writeProjectTags(ctx, r.meshProjectClient, project, tags); err != nil {
		diags.AddError("Error Updating Project Tags", fmt.Sprintf("Could not update tags for project '%s' in workspace '%s': %v", name, workspace, err))
		return
	}

	diags.Append(state.Set(ctx, &plan)...)
}

func (r *projectTagsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectTagsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspace := state.Metadata.WorkspaceIdentifier.ValueString()
	name := state.Metadata.ProjectIdentifier.ValueString()
	defer lockProject(workspace, name)()
	project := readTaggedProject(ctx, r.meshProjectClient, workspace, name, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || project == nil {
		return
	}

	if err := writeProjectTags(ctx, r.meshProjectClient, project, map[string][]string{}); err != nil {
		resp.Diagnostics.AddError("Error Clearing Project Tags", fmt.Sprintf("Could not clear tags for project '%s' in workspace '%s': %v", name, workspace, err))
	}
}

func (r *projectTagsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace_identifier.project_identifier. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("workspace_identifier"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("project_identifier"), parts[1])...)

	nullTags := types.MapNull(types.ListType{ElemType: types.StringType})
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("tags"), nullTags)...)
}

func readTaggedProject(ctx context.Context, projects client.MeshProjectClient, workspace, name string, diags *diag.Diagnostics) *client.MeshProject {
	project, err := projects.Read(ctx, workspace, name)
	if err != nil {
		diags.AddError("Error Reading Project", fmt.Sprintf("Could not read project '%s' in workspace '%s': %v", name, workspace, err))
	}
	return project
}

func writeProjectTags(ctx context.Context, projects client.MeshProjectClient, project *client.MeshProject, tags map[string][]string) error {
	spec := project.Spec
	spec.Tags = tags
	_, err := projects.Update(ctx, &client.MeshProjectCreate{
		Metadata: client.MeshProjectCreateMetadata{Name: project.Metadata.Name, OwnedByWorkspace: project.Metadata.OwnedByWorkspace},
		Spec:     spec,
	})
	return err
}
