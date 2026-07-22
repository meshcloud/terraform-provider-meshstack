package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nestedObjectToObjectType converts a schema.NestedAttributeObject into a types.ObjectType
// by extracting the attr.Type from each attribute.
func nestedObjectToObjectType(nested schema.NestedAttributeObject) types.ObjectType {
	attrTypes := make(map[string]attr.Type, len(nested.Attributes))
	for name, attribute := range nested.Attributes {
		attrTypes[name] = attribute.GetType()
	}
	return types.ObjectType{AttrTypes: attrTypes}
}

// emptySetDefault returns an empty set whose element type is derived from the given
// schema.NestedAttributeObject as a default value.
func emptySetDefault(nested schema.NestedAttributeObject) defaults.Set {
	return setdefault.StaticValue(types.SetValueMust(nestedObjectToObjectType(nested), nil))
}

// meshRefOptions configures a meshObject reference attribute. The zero value yields a required input
// reference: block and identifier are both Required, and `kind` is lenient — optional, defaulted to
// the single valid value, and OneOf-validated. The three opt-in variants (output, omittable-computed,
// in-set) each cost one field.
type meshRefOptions struct {
	Kind        string
	Description string

	// Output makes the reference a computed-only output — a resource's own `.ref` or a data-source
	// attribute. Block, kind and identifier are all Computed, and kind carries no OneOf validator.
	Output bool

	// OptionalComputed marks an input the user may omit because meshStack defaults it: the block and
	// its identifier become Optional+Computed. Ignored when Output is set.
	OptionalComputed bool

	// InSet marks a reference hashed as an opaque set element — nested inside a SetNestedAttribute
	// object (e.g. project_role_ref in a role-mapping set) or used as a set's element type itself
	// (e.g. mandatory_building_block_refs). Terraform hashes set elements by whole value, so an
	// element whose identifier is still unknown at plan can't be hashed and a plain Required
	// identifier fails with "Missing Configuration for Required Attribute". For such refs the block
	// stays Required but the identifier is Optional+Computed with an AlsoRequires guard that enforces
	// presence while tolerating the unknown. Ignored when Output or OptionalComputed is set.
	InSet bool

	// RequiresReplace forces replacement of the resource when the reference block changes — for an
	// input ref that cannot be changed in place (e.g. a tenant's platform_ref / landing_zone_ref).
	RequiresReplace bool
}

// reconcileTrackedTags restricts apiTags to the tag keys already tracked in state at tagsPath. The
// meshObject API returns a superset of the tags a caller sent — an entry for every schema property
// (empty list for unset ones) plus injected restricted-tag defaults — which the caller may be unable
// to manage. Keeping only the previously tracked keys prevents those server-side additions from
// entering the user-managed `tags` attribute and producing spurious drift on the next plan.
//
// On import there is no prior state (tags is null), so apiTags is returned unchanged and the full set
// round-trips. Reading state can fail, so diagnostics are appended to diags; check diags.HasError()
// at the call site as usual.
func reconcileTrackedTags(ctx context.Context, state tfsdk.State, tagsPath path.Path, apiTags map[string][]string, diags *diag.Diagnostics) map[string][]string {
	var priorTags types.Map
	diags.Append(state.GetAttribute(ctx, tagsPath, &priorTags)...)
	if diags.HasError() || priorTags.IsNull() {
		return apiTags
	}

	var tracked map[string][]string
	diags.Append(priorTags.ElementsAs(ctx, &tracked, false)...)
	if diags.HasError() {
		return apiTags
	}

	return reconcileTags(tracked, apiTags)
}

// reconcileTags is the pure core of reconcileTrackedTags: it restricts apiTags to the keys present in
// tracked, dropping the server-injected superset entries (empty lists for undeclared properties and
// restricted-tag defaults) that are not tracked in state.
func reconcileTags(tracked, apiTags map[string][]string) map[string][]string {
	reconciled := make(map[string][]string, len(tracked))
	for key := range tracked {
		if value, ok := apiTags[key]; ok {
			reconciled[key] = value
		}
	}
	return reconciled
}

// meshRefByUuid builds a {kind, uuid} reference for the given meshObject kind.
func meshRefByUuid(opts meshRefOptions) schema.SingleNestedAttribute {
	return meshRef("uuid", "UUID (`metadata.uuid`) of `"+opts.Kind+"`.", opts)
}

// meshRefByName builds a {kind, name} reference for the given meshObject kind.
func meshRefByName(opts meshRefOptions) schema.SingleNestedAttribute {
	return meshRef("name", "Named identifier (`metadata.name`) of `"+opts.Kind+"`.", opts)
}

// meshRef builds the whole {kind, <uuid|name>} reference attribute — block scaffolding, identifier
// and kind discriminator — for a meshObject reference. Callers pick the identifier flavour via
// meshRefByUuid / meshRefByName rather than calling this directly.
//
// It intentionally does not fit refs with bespoke schemas: the discriminated uuid-or-name target_ref
// and the building_block_definition_version_ref (which carries an extra content_hash on the v3 path).
func meshRef(idName, idDesc string, opts meshRefOptions) schema.SingleNestedAttribute {
	kindAttr := schema.StringAttribute{
		MarkdownDescription: "meshObject type, always `" + opts.Kind + "`.",
		Default:             stringdefault.StaticString(opts.Kind),
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	idAttr := schema.StringAttribute{MarkdownDescription: idDesc}
	block := schema.SingleNestedAttribute{MarkdownDescription: opts.Description}

	// `kind` is always lenient for inputs: optional, defaulted to the single valid value, and
	// OneOf-validated, so the user never has to spell it out. (Output overrides this below.)
	lenientKind := func() {
		kindAttr.Optional = true
		kindAttr.Computed = true
		kindAttr.Validators = []validator.String{stringvalidator.OneOf(opts.Kind)}
	}
	// A computed identifier keeps its last-known value while the plan can't resolve it yet.
	computedId := func() {
		idAttr.Optional = true
		idAttr.Computed = true
		idAttr.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	}

	switch {
	case opts.Output:
		block.Computed = true
		kindAttr.Computed = true
		idAttr.Computed = true
		idAttr.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
		// Keep `kind` known at plan time; only the identifier is genuinely computed.
		block.PlanModifiers = []planmodifier.Object{refOutputKind{kind: opts.Kind, idName: idName}}

	case opts.OptionalComputed:
		// meshStack defaults this input, so the user may omit it: block and identifier are Optional+Computed.
		lenientKind()
		computedId()
		block.Optional = true
		block.Computed = true

	case opts.InSet:
		// The ref is hashed as an opaque set element, so its identifier can't be plain Required —
		// a set element whose identifier is still unknown at plan collapses to a wholly-unknown
		// element (sets hash by whole value), and a Required identifier then fails with "Missing
		// Configuration for Required Attribute". Keep the identifier Optional+Computed with an
		// AlsoRequires guard that enforces presence while tolerating the unknown.
		lenientKind()
		computedId()
		block.Required = true
		block.Validators = []validator.Object{
			objectvalidator.AlsoRequires(path.MatchRelative().AtName(idName)),
		}
		// Note goes on the identifier, not the block: set-element-typed refs
		// (dependency_refs, mandatory_building_block_refs) are spread into a NestedAttributeObject
		// that keeps only .Attributes, dropping the block description — but the identifier renders
		// under "Optional:" in every case, which is exactly the line that needs the caveat.
		idAttr.MarkdownDescription += " Required; optional here only so a computed reference can be used inside a set, and enforced at plan time."

	default:
		// Plain required input: both block and identifier are Required.
		lenientKind()
		block.Required = true
		idAttr.Required = true
	}

	// An omittable-computed input keeps its last-known value while the plan can't resolve it yet.
	if block.Computed && !opts.Output {
		block.PlanModifiers = append(block.PlanModifiers, objectplanmodifier.UseStateForUnknown())
	}
	if opts.RequiresReplace {
		block.PlanModifiers = append(block.PlanModifiers, objectplanmodifier.RequiresReplace())
	}

	block.Attributes = map[string]schema.Attribute{"kind": kindAttr, idName: idAttr}
	return block
}

// refOutputKind keeps an output reference's `kind` known at plan time. An output block is Computed,
// so on create the framework plans the whole object as unknown ("known after apply") — including
// `kind`, which is always the single constant value. This modifier fills a still-unknown block with
// its known kind and an unknown identifier; when prior state exists it carries that state instead
// (as UseStateForUnknown would), so updates keep showing the already-resolved identifier.
//
// The resulting partial object does not survive set membership: Terraform hashes set elements by
// whole value, so a ref feeding a set is unknown there regardless — which is why set-nested input
// identifiers still can't be plain Required (see meshRef). Plan modifiers run for resources only, so
// this is inert for data-source refs, which instead resolve `kind` at their plan-time read.
type refOutputKind struct {
	kind   string
	idName string
}

func (m refOutputKind) Description(context.Context) string {
	return "Keeps the reference `kind` known at plan time; the identifier stays computed."
}

func (m refOutputKind) MarkdownDescription(ctx context.Context) string { return m.Description(ctx) }

func (m refOutputKind) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if !req.PlanValue.IsUnknown() {
		return
	}
	if !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}

	obj, diags := types.ObjectValue(req.PlanValue.AttributeTypes(ctx), map[string]attr.Value{
		"kind":   types.StringValue(m.kind),
		m.idName: types.StringUnknown(),
	})
	resp.Diagnostics.Append(diags...)
	resp.PlanValue = obj
}

func tenantTagsAttribute() schema.SingleNestedAttribute {
	tagMappersNested := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				MarkdownDescription: "Key for the tag mapper",
				Required:            true,
			},
			"value_pattern": schema.StringAttribute{
				MarkdownDescription: "Value pattern for the tag mapper",
				Required:            true,
			},
		},
	}

	return schema.SingleNestedAttribute{
		MarkdownDescription: "Tenant tags configuration",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"namespace_prefix": schema.StringAttribute{
				MarkdownDescription: "This is the prefix for all labels created by meshStack. It helps to keep track of which labels are managed by meshStack. It is recommended to let this prefix end with a delimiter like an underscore.",
				Required:            true,
			},
			"tag_mappers": schema.SetNestedAttribute{
				MarkdownDescription: "Set of tag mappers for tenant tags",
				Optional:            true,
				Computed:            true,
				Default:             emptySetDefault(tagMappersNested),
				NestedObject:        tagMappersNested,
			},
		},
	}
}

// previewDisclaimer returns a standard MarkdownDescription note for resources and data sources
// that use a preview API. Append this to the MarkdownDescription of any preview resource.
func previewDisclaimer() string {
	return "\n\n~> **Preview:** This resource is in preview. " +
		"Breaking changes are possible without prior notice due to changes in the underlying [meshStack preview API](https://docs.meshcloud.io/api/technical-specifications#preview-endpoints) or due to changes in this provider. " +
		"Please ensure you are running the latest version of the provider and report any bugs via [GitHub issues](https://github.com/meshcloud/terraform-provider-meshstack/issues) " +
		"or via support@meshcloud.io."
}
