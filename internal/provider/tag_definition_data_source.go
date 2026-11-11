package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/util/iter"
)

func NewTagDefinitionDataSource() datasource.DataSource {
	return &tagDefinitionDataSource{}
}

type tagDefinitionDataSource struct {
	meshTagDefinitionClient client.MeshTagDefinitionClient
}

func (d *tagDefinitionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_definition"
}

func (d *tagDefinitionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := tagDefinitionDataSourceAttributes()
	attributes["metadata"] = asFilter(attributes["metadata"], "name")
	attributes["spec"] = asFilter(attributes["spec"], "target_kind", "key")

	resp.Schema = schema.Schema{
		MarkdownDescription: "Read the one tag definition that matches what you set in `metadata` and `spec`, such as `metadata.name`. " +
			"It fails unless exactly one tag definition matches.",
		Attributes: attributes,
	}
}

func (d *tagDefinitionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshTagDefinitionClient = client.TagDefinition
	})...)
}

func (d *tagDefinitionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var filter tagDefinitionFilter
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &filter.Name)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("target_kind"), &filter.TargetKind)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("key"), &filter.Key)...)
	if resp.Diagnostics.HasError() {
		return
	}

	candidates := d.candidates(ctx, filter, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	var matches []client.MeshTagDefinition
	for _, candidate := range candidates {
		if filter.matches(candidate) {
			matches = append(matches, candidate)
		}
	}
	switch len(matches) {
	case 1:
		resp.Diagnostics.Append(resp.State.Set(ctx, &matches[0])...)
	case 0:
		resp.Diagnostics.AddError("Tag definition not found", fmt.Sprintf("No tag definition matches %s.", filter))
	default:
		names := make([]string, len(matches))
		for i, match := range matches {
			names[i] = match.Metadata.Name
		}
		resp.Diagnostics.AddError("Tag definition is ambiguous",
			fmt.Sprintf("%d tag definitions match %s: %s. Set `metadata.name` to pick one.", len(matches), filter, strings.Join(names, ", ")))
	}
}

func (d *tagDefinitionDataSource) candidates(ctx context.Context, filter tagDefinitionFilter, diags *diag.Diagnostics) []client.MeshTagDefinition {
	if filter.Name == nil {
		tags, err := iter.Collect(d.meshTagDefinitionClient.List(ctx, client.NoFilter{}))
		if err != nil {
			diags.AddError("Unable to list meshTagDefinitions", err.Error())
		}
		return tags
	}
	tag, err := d.meshTagDefinitionClient.Read(ctx, *filter.Name)
	if err != nil {
		diags.AddError("Unable to read meshTagDefinition", err.Error())
	}
	if tag == nil {
		return nil
	}
	return []client.MeshTagDefinition{*tag}
}

func tagDefinitionDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"metadata": schema.SingleNestedAttribute{
			MarkdownDescription: "Always contains the 'name' to uniquely identify the meshTagDefinition.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"name": computedString("Must be of the form $targetKind.$key since tag definitions must be non-conflicting."),
			},
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Specification for the meshTagDefinition.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"target_kind": computedString("The kind of meshObject this tag is defined for. Must be one of: meshProject, meshWorkspace, meshLandingZone, meshPaymentMethod or meshBuildingBlockDefinition."),
				"key":         computedString("The key of the tag."),
				"value_type": schema.SingleNestedAttribute{
					MarkdownDescription: "The TagValueType of the tag. Must define exactly one of the available types.",
					Computed:            true,
					Attributes: map[string]schema.Attribute{
						"string": textTagValueTypeDataSourceSchema("string, represented as JSON string"),
						"email":  textTagValueTypeDataSourceSchema("email address, represented as JSON string"),
						"integer": schema.SingleNestedAttribute{
							MarkdownDescription: "an integer, represented as a JSON number",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"default_value": computedInt64("The default value of the tag."),
							},
						},
						"number": schema.SingleNestedAttribute{
							MarkdownDescription: "a decimal number, represented as a JSON number",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"default_value": schema.Float64Attribute{
									MarkdownDescription: "The default value of the tag.",
									Computed:            true,
								},
							},
						},
						"single_select": schema.SingleNestedAttribute{
							MarkdownDescription: "a string from a list of options, represented as a JSON string",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"options": schema.ListAttribute{
									MarkdownDescription: "The allowed options for the tag as a string[]",
									Computed:            true,
									ElementType:         types.StringType,
								},
								"default_value": computedString("The default value of the tag."),
							},
						},
						"multi_select": schema.SingleNestedAttribute{
							MarkdownDescription: "one or multiple strings from a list of options, represented as a JSON array",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"options": schema.ListAttribute{
									MarkdownDescription: "The allowed options for the tag as a string[]",
									Computed:            true,
									ElementType:         types.StringType,
								},
								"default_value": schema.ListAttribute{
									MarkdownDescription: "The default value of the tag.",
									Computed:            true,
									ElementType:         types.StringType,
								},
							},
						},
					},
				},
				"description":     computedString("The detailed description of the tag."),
				"display_name":    computedString("The display name of the tag."),
				"sort_order":      computedInt64("The sort order for this tag when displayed in the UI. meshPanel sorts tags in ascending order."),
				"mandatory":       computedBool("Indicates whether the tag is mandatory."),
				"immutable":       computedBool("Indicates whether the tag value is not editable after initially set."),
				"restricted":      computedBool("Indicates whether only admins can edit this tag."),
				"replication_key": computedString("The replication key of the tag."),
			},
		},
	}
}

func textTagValueTypeDataSourceSchema(description string) schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: description,
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"default_value":    computedString("The default value of the tag."),
			"validation_regex": computedString("The regex pattern that the tag value must match."),
		},
	}
}

type tagDefinitionFilter struct {
	Name       *string
	TargetKind *string
	Key        *string
}

func (filter tagDefinitionFilter) matches(tag client.MeshTagDefinition) bool {
	return (filter.Name == nil || tag.Metadata.Name == *filter.Name) &&
		(filter.TargetKind == nil || tag.Spec.TargetKind == *filter.TargetKind) &&
		(filter.Key == nil || tag.Spec.Key == *filter.Key)
}

func (filter tagDefinitionFilter) String() string {
	var set []string
	for _, attribute := range []struct {
		name  string
		value *string
	}{
		{"metadata.name", filter.Name},
		{"spec.target_kind", filter.TargetKind},
		{"spec.key", filter.Key},
	} {
		if attribute.value != nil {
			set = append(set, fmt.Sprintf("%s = %q", attribute.name, *attribute.value))
		}
	}
	if len(set) == 0 {
		return "an empty filter"
	}
	return strings.Join(set, " and ")
}
