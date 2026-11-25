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
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

type tagTargetInfo struct {
	name              string
	title             string
	kind              string
	identityAttribute string
	importIdentifiers []string
	inlineTags        string
	injectedDefaultOn string
	extraCaveats      string
}

type tagTarget[I any] interface {
	info() tagTargetInfo
	identitySchema() schema.Attribute
	identityFromImport(ids []string) I
	withClient(c client.Client) tagTarget[I]
	lock(id I) (unlock func())
	read(ctx context.Context, id I) (*tagSnapshot, error)
	describe(id I) string
}

type tagSnapshot struct {
	tags  map[string][]string
	write func(ctx context.Context, tags map[string][]string) (stored map[string][]string, err error)
}

type workspaceTagTarget struct {
	workspaces client.MeshWorkspaceClient
}

func (workspaceTagTarget) info() tagTargetInfo {
	return tagTargetInfo{
		name:              "workspace",
		title:             "Workspace",
		kind:              client.MeshObjectKind.Workspace,
		identityAttribute: "workspace",
		importIdentifiers: []string{"workspace"},
		inlineTags:        "metadata.tags",
		injectedDefaultOn: "a workspace created in meshPanel",
	}
}

func (workspaceTagTarget) identitySchema() schema.Attribute {
	return requiredReplaceOnChangeString("Name (`metadata.name`) of the workspace to tag.")
}

func (workspaceTagTarget) identityFromImport(ids []string) string { return ids[0] }

func (workspaceTagTarget) withClient(c client.Client) tagTarget[string] {
	return workspaceTagTarget{workspaces: c.Workspace}
}

func (workspaceTagTarget) lock(name string) func() { return lockWorkspace(name) }

func (t workspaceTagTarget) read(ctx context.Context, name string) (*tagSnapshot, error) {
	workspace, err := t.workspaces.Read(ctx, name)
	if err != nil || workspace == nil {
		return nil, err
	}
	return &tagSnapshot{
		tags: workspace.Metadata.Tags,
		write: func(ctx context.Context, tags map[string][]string) (map[string][]string, error) {
			return writeWorkspaceTags(ctx, t.workspaces, workspace, tags)
		},
	}, nil
}

func (workspaceTagTarget) describe(name string) string {
	return fmt.Sprintf("workspace '%s'", name)
}

func writeWorkspaceTags(ctx context.Context, workspaces client.MeshWorkspaceClient, workspace *client.MeshWorkspace, tags map[string][]string) (map[string][]string, error) {
	updated, err := workspaces.Update(ctx, workspace.Metadata.Name, &client.MeshWorkspaceCreate{
		Metadata: client.MeshWorkspaceCreateMetadata{Name: workspace.Metadata.Name, Tags: tags},
		Spec:     workspace.Spec,
	})
	if err != nil {
		return nil, err
	}
	return updated.Metadata.Tags, nil
}

type projectTagTarget struct {
	projects client.MeshProjectClient
}

func (projectTagTarget) info() tagTargetInfo {
	return tagTargetInfo{
		name:              "project",
		title:             "Project",
		kind:              client.MeshObjectKind.Project,
		identityAttribute: "project_ref",
		importIdentifiers: []string{"workspace", "project"},
		inlineTags:        "spec.tags",
		injectedDefaultOn: "a project",
		extraCaveats: "~> **The payment methods are written back too.** Every write sends the project's display name, " +
			"`payment_method_identifier` and `substitute_payment_method_identifier` as it read them. If meshStack switches the " +
			"project to its substitute payment method between this resource's read and its write, the write puts the old " +
			"payment method back.",
	}
}

func (projectTagTarget) identitySchema() schema.Attribute {
	return meshProjectRef(meshRefOptions{Description: "Reference to the project to tag.", RequiresReplace: true})
}

func (projectTagTarget) identityFromImport(ids []string) projectRef {
	return newProjectRef(ids[0], ids[1])
}

func (projectTagTarget) withClient(c client.Client) tagTarget[projectRef] {
	return projectTagTarget{projects: c.Project}
}

func (projectTagTarget) lock(ref projectRef) func() {
	return lockProject(ref.OwnedByWorkspace, ref.Name)
}

func (t projectTagTarget) read(ctx context.Context, ref projectRef) (*tagSnapshot, error) {
	project, err := t.projects.Read(ctx, ref.OwnedByWorkspace, ref.Name)
	if err != nil || project == nil {
		return nil, err
	}
	return &tagSnapshot{
		tags: project.Spec.Tags,
		write: func(ctx context.Context, tags map[string][]string) (map[string][]string, error) {
			return writeProjectTags(ctx, t.projects, project, tags)
		},
	}, nil
}

func (projectTagTarget) describe(ref projectRef) string {
	return fmt.Sprintf("project '%s' in workspace '%s'", ref.Name, ref.OwnedByWorkspace)
}

func writeProjectTags(ctx context.Context, projects client.MeshProjectClient, project *client.MeshProject, tags map[string][]string) (map[string][]string, error) {
	spec := project.Spec
	spec.Tags = tags
	updated, err := projects.Update(ctx, &client.MeshProjectCreate{
		Metadata: client.MeshProjectCreateMetadata{Name: project.Metadata.Name, OwnedByWorkspace: project.Metadata.OwnedByWorkspace},
		Spec:     spec,
	})
	if err != nil {
		return nil, err
	}
	return updated.Spec.Tags, nil
}

type tagResourceBase[I any] struct {
	target    tagTarget[I]
	suffix    string
	version   int64
	upgraders map[int64]resource.StateUpgrader
}

func (r *tagResourceBase[I]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.target.info().name + "_" + r.suffix
}

func (r *tagResourceBase[I]) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(c client.Client) {
		r.target = r.target.withClient(c)
	})...)
}

func (r *tagResourceBase[I]) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return r.upgraders
}

func (r *tagResourceBase[I]) resourceType(suffix string) string {
	return "meshstack_" + r.target.info().name + "_" + suffix
}

func (r *tagResourceBase[I]) schema(description string, attributes map[string]schema.Attribute) schema.Schema {
	attributes[r.target.info().identityAttribute] = r.target.identitySchema()
	return schema.Schema{
		MarkdownDescription: description,
		Version:             r.version,
		Attributes:          attributes,
	}
}

func (r *tagResourceBase[I]) caveats(tagResourceSuffix string) string {
	caveats := tagResourceCaveats(r.target.info(), r.resourceType(tagResourceSuffix))
	if extra := r.target.info().extraCaveats; extra != "" {
		caveats += "\n\n" + extra
	}
	return caveats
}

func (r *tagResourceBase[I]) identity(ctx context.Context, getter generic.AttributeGetter, diags *diag.Diagnostics) I {
	return generic.GetAttribute[I](ctx, getter, path.Root(r.target.info().identityAttribute), diags)
}

func (r *tagResourceBase[I]) read(ctx context.Context, id I, diags *diag.Diagnostics) *tagSnapshot {
	snapshot, err := r.target.read(ctx, id)
	if err != nil {
		diags.AddError("Error Reading "+r.target.info().title, fmt.Sprintf("Could not read %s: %v", r.target.describe(id), err))
	}
	return snapshot
}

func (r *tagResourceBase[I]) readExisting(ctx context.Context, id I, diags *diag.Diagnostics) *tagSnapshot {
	snapshot := r.read(ctx, id, diags)
	if snapshot == nil && !diags.HasError() {
		diags.AddError("Error Reading "+r.target.info().title, fmt.Sprintf("The %s was not found.", r.target.describe(id)))
	}
	return snapshot
}

func (r *tagResourceBase[I]) write(ctx context.Context, id I, snapshot *tagSnapshot, tags map[string][]string, action string, diags *diag.Diagnostics) map[string][]string {
	stored, err := snapshot.write(ctx, tags)
	if err != nil {
		diags.AddError("Error Updating "+r.target.info().title+" Tags", fmt.Sprintf("Could not %s on %s: %v", action, r.target.describe(id), err))
	}
	return stored
}

func (r *tagResourceBase[I]) importFailed(id, trailing string, diags *diag.Diagnostics) {
	format := strings.Join(append(r.target.info().importIdentifiers, trailing), ".")
	diags.AddError("Unexpected Import Identifier", fmt.Sprintf("Expected import identifier with format: %s. Got: %q", strings.TrimSuffix(format, "."), id))
}

func (r *tagResourceBase[I]) importIdentity(ctx context.Context, ids []string, state *tfsdk.State, diags *diag.Diagnostics) {
	diags.Append(generic.SetAttributeTo(ctx, state, path.Root(r.target.info().identityAttribute), r.target.identityFromImport(ids))...)
}

func (r *tagResourceBase[I]) importIdentifiers() int {
	return len(r.target.info().importIdentifiers)
}
