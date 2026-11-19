package provider

import (
	"context"
	"fmt"
	"maps"
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
	_ resource.Resource                = &projectTagResource{}
	_ resource.ResourceWithConfigure   = &projectTagResource{}
	_ resource.ResourceWithImportState = &projectTagResource{}
)

var projectTagCaveats = tagResourceCaveats(taggedObject{
	name:              "project",
	kind:              "meshProject",
	ownerResource:     "meshstack_project",
	inlineTags:        "spec.tags",
	tagResource:       "meshstack_project_tag",
	tagsResource:      "meshstack_project_tags",
	injectedDefaultOn: "a project",
}) + "\n\n" +
	"~> **The payment methods are written back too.** Every write sends the project's display name, " +
	"`payment_method_identifier` and `substitute_payment_method_identifier` as it read them. If meshStack switches the " +
	"project to its substitute payment method between this resource's read and its write, the write puts the old " +
	"payment method back."

func NewProjectTagResource() resource.Resource {
	return &projectTagResource{}
}

type projectTagResource struct {
	meshProjectClient client.MeshProjectClient
}

type projectTagMetadata struct {
	WorkspaceIdentifier types.String `tfsdk:"workspace_identifier"`
	ProjectIdentifier   types.String `tfsdk:"project_identifier"`
	Key                 types.String `tfsdk:"key"`
}

type projectTagSpec struct {
	Values types.List `tfsdk:"values"`
}

type projectTagModel struct {
	Metadata projectTagMetadata `tfsdk:"metadata"`
	Spec     projectTagSpec     `tfsdk:"spec"`
}

func (r *projectTagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_tag"
}

func (r *projectTagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(c client.Client) {
		r.meshProjectClient = c.Project
	})...)
}

func (r *projectTagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single dedicated tag for a meshStack project.\n\n" +
			projectTagCaveats + "\n\n" +
			"~> **Note:** Do not mix dedicated `meshstack_project_tag` resources with authoritative " +
			"`meshstack_project_tags` or inline `tags` on `meshstack_project` for the same project. This resource only " +
			"manages its own key; tags under other keys are read and written back unchanged.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Project tag metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"workspace_identifier": requiredReplaceOnChangeString("Identifier of the workspace that owns the target project."),
					"project_identifier":   requiredReplaceOnChangeString("Identifier of the target project."),
					"key":                  requiredReplaceOnChangeString("Key of the tag."),
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Project tag specification.",
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

func (r *projectTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.setTag(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *projectTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := state.Metadata.Key.ValueString()
	project := readTaggedProject(ctx, r.meshProjectClient, state.Metadata.WorkspaceIdentifier.ValueString(), state.Metadata.ProjectIdentifier.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if project == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	vals, ok := project.Spec.Tags[key]
	if !ok {
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

func (r *projectTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.setTag(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *projectTagResource) setTag(ctx context.Context, tfPlan tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan projectTagModel
	diags.Append(tfPlan.Get(ctx, &plan)...)
	if diags.HasError() {
		return
	}

	workspace := plan.Metadata.WorkspaceIdentifier.ValueString()
	name := plan.Metadata.ProjectIdentifier.ValueString()
	key := plan.Metadata.Key.ValueString()

	defer lockProject(workspace, name)()
	project := readTaggedProject(ctx, r.meshProjectClient, workspace, name, diags)
	if diags.HasError() {
		return
	}
	if project == nil {
		diags.AddError("Error Reading Project", fmt.Sprintf("Project '%s' not found in workspace '%s'", name, workspace))
		return
	}

	var values []string
	diags.Append(plan.Spec.Values.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return
	}

	tags := make(map[string][]string, len(project.Spec.Tags)+1)
	maps.Copy(tags, project.Spec.Tags)
	tags[key] = values

	if err := writeProjectTags(ctx, r.meshProjectClient, project, tags); err != nil {
		diags.AddError("Error Updating Project Tag", fmt.Sprintf("Could not set tag '%s' on project '%s' in workspace '%s': %v", key, name, workspace, err))
		return
	}

	diags.Append(state.Set(ctx, &plan)...)
}

func (r *projectTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspace := state.Metadata.WorkspaceIdentifier.ValueString()
	name := state.Metadata.ProjectIdentifier.ValueString()
	key := state.Metadata.Key.ValueString()

	defer lockProject(workspace, name)()
	project := readTaggedProject(ctx, r.meshProjectClient, workspace, name, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || project == nil {
		return
	}

	tags := maps.Clone(project.Spec.Tags)
	if tags == nil {
		tags = map[string][]string{}
	}
	delete(tags, key)

	if err := writeProjectTags(ctx, r.meshProjectClient, project, tags); err != nil {
		resp.Diagnostics.AddError("Error Removing Project Tag", fmt.Sprintf("Could not remove tag '%s' from project '%s' in workspace '%s': %v", key, name, workspace, err))
	}
}

func (r *projectTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ".", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace_identifier.project_identifier.key. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("workspace_identifier"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("project_identifier"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("key"), parts[2])...)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("values"), types.ListNull(types.StringType))...)
}
