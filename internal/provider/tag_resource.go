package provider

import (
	"context"
	"fmt"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ resource.ResourceWithConfigure    = &tagResource[string]{}
	_ resource.ResourceWithImportState  = &tagResource[workspaceScopedRef]{}
	_ resource.ResourceWithUpgradeState = &tagResource[string]{}
)

func NewWorkspaceTagResource() resource.Resource {
	return &tagResource[string]{tagResourceBase[string]{
		target:  workspaceTagTarget{},
		suffix:  "tag",
		version: 1,
		upgraders: map[int64]resource.StateUpgrader{0: rawStateUpgrader(
			moveStateAttribute([]string{"metadata", "workspace_identifier"}, []string{"workspace"}),
			moveStateAttribute([]string{"metadata", "key"}, []string{"key"}),
			moveStateAttribute([]string{"spec", "values"}, []string{"values"}),
			removeStateAttribute("metadata"),
			removeStateAttribute("spec"),
		)},
	}}
}

func NewProjectTagResource() resource.Resource {
	return &tagResource[workspaceScopedRef]{tagResourceBase[workspaceScopedRef]{target: projectTagTarget{}, suffix: "tag"}}
}

type tagResource[I any] struct {
	tagResourceBase[I]
}

type tagModel struct {
	Key    string   `tfsdk:"key"`
	Values []string `tfsdk:"values"`
}

func (r *tagResource[I]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	name := r.target.info().name
	resp.Schema = r.schema(
		"Manages a single dedicated tag for a meshStack "+name+".\n\n"+
			r.caveats("tag")+"\n\n"+
			"~> **Note:** Do not mix dedicated `meshstack_"+name+"_tag` resources with authoritative "+
			"`meshstack_"+name+"_tags` or inline `tags` on `meshstack_"+name+"` for the same "+name+", or with "+
			"`meshstack_"+name+"_tag_value` resources for the same key. This resource only "+
			"manages its own key; tags under other keys are read and written back unchanged.",
		map[string]schema.Attribute{
			"key":    requiredReplaceOnChangeString("Key of the tag."),
			"values": tagValuesAttribute(),
		},
	)
}

func (r *tagResource[I]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.setTag(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *tagResource[I]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	id := r.identity(ctx, req.State, &resp.Diagnostics)
	state := generic.Get[tagModel](ctx, req.State, &resp.Diagnostics)
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

	values, found := trackedTagValues(snapshot.tags, state.Key, state.Values)
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("values"), values)...)
}

func (r *tagResource[I]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.setTag(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *tagResource[I]) setTag(ctx context.Context, plan tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	id := r.identity(ctx, plan, diags)
	model := generic.Get[tagModel](ctx, plan, diags)
	if diags.HasError() {
		return
	}

	defer r.target.lock(id)()
	snapshot := r.readExisting(ctx, id, diags)
	if diags.HasError() {
		return
	}

	tags := maps.Clone(snapshot.tags)
	if tags == nil {
		tags = map[string][]string{}
	}
	tags[model.Key] = model.Values

	stored := r.write(ctx, id, snapshot, tags, fmt.Sprintf("set tag '%s'", model.Key), diags)
	if diags.HasError() {
		return
	}
	r.verifyStored(id, map[string][]string{model.Key: model.Values}, stored, diags)
	if diags.HasError() {
		return
	}

	state.Raw = plan.Raw
}

func (r *tagResource[I]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	id := r.identity(ctx, req.State, &resp.Diagnostics)
	model := generic.Get[tagModel](ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	defer r.target.lock(id)()
	snapshot := r.read(ctx, id, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || snapshot == nil {
		return
	}

	tags := maps.Clone(snapshot.tags)
	if tags == nil {
		tags = map[string][]string{}
	}
	delete(tags, model.Key)
	r.write(ctx, id, snapshot, tags, fmt.Sprintf("remove tag '%s'", model.Key), &resp.Diagnostics)
}

func (r *tagResource[I]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids, key, ok := splitTagImportID(req.ID, r.importIdentifiers(), true)
	if !ok {
		r.importFailed(req.ID, "key", &resp.Diagnostics)
		return
	}
	r.importIdentity(ctx, ids, &resp.State, &resp.Diagnostics)
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("key"), key)...)
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("values"), []string(nil))...)
}
