package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ resource.ResourceWithConfigure    = &tagsResource[string]{}
	_ resource.ResourceWithImportState  = &tagsResource[projectRef]{}
	_ resource.ResourceWithUpgradeState = &tagsResource[string]{}
)

func NewWorkspaceTagsResource() resource.Resource {
	return &tagsResource[string]{tagResourceBase[string]{
		target:  workspaceTagTarget{},
		suffix:  "tags",
		version: 1,
		upgraders: map[int64]resource.StateUpgrader{0: rawStateUpgrader(
			moveStateAttribute([]string{"metadata", "workspace_identifier"}, []string{"workspace"}),
			moveStateAttribute([]string{"spec", "tags"}, []string{"tags"}),
			removeStateAttribute("metadata"),
			removeStateAttribute("spec"),
		)},
	}}
}

func NewProjectTagsResource() resource.Resource {
	return &tagsResource[projectRef]{tagResourceBase[projectRef]{target: projectTagTarget{}, suffix: "tags"}}
}

type tagsResource[I any] struct {
	tagResourceBase[I]
}

type tagsModel struct {
	Tags map[string][]string `tfsdk:"tags"`
}

func (r *tagsResource[I]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	info := r.target.info()
	name := info.name
	resp.Schema = r.schema(
		"Authoritatively manages all tags for a meshStack "+name+".\n\n"+
			r.caveats("tag")+"\n\n"+
			"~> **Note:** This resource is authoritative on write: every apply replaces **all** tags on the target "+
			name+" with the ones configured here, and destroying it removes them all. It is not authoritative on "+
			"read — refresh only tracks the keys you configure, so a tag added under another key outside Terraform is "+
			"not reported as drift, and is removed by the next apply that writes this resource. This is deliberate: "+
			info.injectedDefaultOn+" carries the defaults meshStack injects for restricted tag definitions, and adopting "+
			"those would produce a plan that never converges. Note also that a tag declared with an empty value list is "+
			"kept in state, because the API returns no entry for it at all. Do not mix `meshstack_"+name+"_tags` with "+
			"inline `tags` on `meshstack_"+name+"` or with `meshstack_"+name+"_tag` or `meshstack_"+name+"_tag_value` "+
			"resources on the same "+name+".",
		map[string]schema.Attribute{"tags": tagsMapAttribute(name)},
	)
}

func (r *tagsResource[I]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.setTags(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *tagsResource[I]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	id := r.identity(ctx, req.State, &resp.Diagnostics)
	state := generic.Get[tagsModel](ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshot := r.read(ctx, id, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if snapshot == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	tags := snapshot.tags
	if state.Tags != nil {
		tags = reconcileTags(state.Tags, tags)
	}
	if tags == nil {
		tags = map[string][]string{}
	}
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("tags"), tags)...)
}

func (r *tagsResource[I]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.setTags(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *tagsResource[I]) setTags(ctx context.Context, plan tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	id := r.identity(ctx, plan, diags)
	model := generic.Get[tagsModel](ctx, plan, diags)
	if diags.HasError() {
		return
	}

	defer r.target.lock(id)()
	snapshot := r.readExisting(ctx, id, diags)
	if diags.HasError() {
		return
	}

	tags := model.Tags
	if tags == nil {
		tags = map[string][]string{}
	}
	r.write(ctx, id, snapshot, tags, "update tags", diags)
	if diags.HasError() {
		return
	}

	state.Raw = plan.Raw
}

func (r *tagsResource[I]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	id := r.identity(ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	defer r.target.lock(id)()
	snapshot := r.read(ctx, id, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || snapshot == nil {
		return
	}
	r.write(ctx, id, snapshot, map[string][]string{}, "clear tags", &resp.Diagnostics)
}

func (r *tagsResource[I]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids, _, ok := splitTagImportID(req.ID, r.importIdentifiers(), false)
	if !ok {
		r.importFailed(req.ID, "", &resp.Diagnostics)
		return
	}
	r.importIdentity(ctx, ids, &resp.State, &resp.Diagnostics)
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("tags"), map[string][]string(nil))...)
}
