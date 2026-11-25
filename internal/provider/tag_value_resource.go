package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ resource.ResourceWithConfigure   = &tagValueResource[string]{}
	_ resource.ResourceWithImportState = &tagValueResource[projectRef]{}
)

func NewWorkspaceTagValueResource() resource.Resource {
	return &tagValueResource[string]{tagResourceBase[string]{target: workspaceTagTarget{}, suffix: "tag_value"}}
}

func NewProjectTagValueResource() resource.Resource {
	return &tagValueResource[projectRef]{tagResourceBase[projectRef]{target: projectTagTarget{}, suffix: "tag_value"}}
}

type tagValueResource[I any] struct {
	tagResourceBase[I]
}

type tagValueModel struct {
	Key   string `tfsdk:"key"`
	Value string `tfsdk:"value"`
}

func (r *tagValueResource[I]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	name := r.target.info().name
	resp.Schema = r.schema(
		"Manages a single value of a multi-select tag on a meshStack "+name+".\n\n"+
			r.caveats("tag_value")+"\n\n"+
			tagValueNote(name),
		map[string]schema.Attribute{
			"key":   requiredReplaceOnChangeString("Key of the tag."),
			"value": requiredReplaceOnChangeString("Value to add to the tag."),
		},
	)
}

func (r *tagValueResource[I]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	id := r.identity(ctx, req.Plan, &resp.Diagnostics)
	model := generic.Get[tagValueModel](ctx, req.Plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	defer r.target.lock(id)()
	snapshot := r.readExisting(ctx, id, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if tags, changed := withTagValue(snapshot.tags, model.Key, model.Value); changed {
		r.write(ctx, id, snapshot, tags, fmt.Sprintf("add value '%s' to tag '%s'", model.Value, model.Key), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.State.Raw = req.Plan.Raw
}

func (r *tagValueResource[I]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	id := r.identity(ctx, req.State, &resp.Diagnostics)
	model := generic.Get[tagValueModel](ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshot := r.read(ctx, id, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if snapshot == nil || !slices.Contains(snapshot.tags[model.Key], model.Value) {
		resp.State.RemoveResource(ctx)
	}
}

func (r *tagValueResource[I]) Update(_ context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.State.Raw = req.Plan.Raw
}

func (r *tagValueResource[I]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	id := r.identity(ctx, req.State, &resp.Diagnostics)
	model := generic.Get[tagValueModel](ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	defer r.target.lock(id)()
	snapshot := r.read(ctx, id, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || snapshot == nil {
		return
	}

	if tags, changed := withoutTagValue(snapshot.tags, model.Key, model.Value); changed {
		r.write(ctx, id, snapshot, tags, fmt.Sprintf("remove value '%s' from tag '%s'", model.Value, model.Key), &resp.Diagnostics)
	}
}

func (r *tagValueResource[I]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids, key, value, ok := parseTagValueImportID(req.ID, r.importIdentifiers())
	if !ok {
		r.importFailed(req.ID, "key=value", &resp.Diagnostics)
		return
	}
	r.importIdentity(ctx, ids, &resp.State, &resp.Diagnostics)
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("key"), key)...)
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("value"), value)...)
}
