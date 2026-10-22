package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ datasource.DataSource              = &buildingBlockDefinitionDataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlockDefinitionDataSource{}
)

func NewBuildingBlockDefinitionDataSource() datasource.DataSource {
	return &buildingBlockDefinitionDataSource{}
}

type buildingBlockDefinitionDataSource struct {
	meshBuildingBlockDefinitionClient        client.MeshBuildingBlockDefinitionClient
	meshBuildingBlockDefinitionVersionClient client.MeshBuildingBlockDefinitionVersionClient
}

type buildingBlockDefinitionFilter struct {
	Uuid             *string
	OwnedByWorkspace *string
	DisplayName      *string
	TargetType       *string
}

func (d *buildingBlockDefinitionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_definition"
}

func (d *buildingBlockDefinitionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshBuildingBlockDefinitionClient = client.BuildingBlockDefinition
		d.meshBuildingBlockDefinitionVersionClient = client.BuildingBlockDefinitionVersion
	})...)
}

func (d *buildingBlockDefinitionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := buildingBlockDefinitionDataSourceAttributes()
	attributes["metadata"] = asFilter(attributes["metadata"], "uuid", "owned_by_workspace")
	attributes["spec"] = asFilter(attributes["spec"], "display_name", "target_type")

	resp.Schema = schema.Schema{
		MarkdownDescription: "Read the one building block definition that matches what you set in `metadata` and `spec`, " +
			"such as `spec.display_name`, the name the marketplace in meshPanel shows. " +
			"It searches your workspace's own and all published definitions, and fails unless exactly one of them matches. " +
			"Assign `version_latest`, `version_latest_release` once a version is released, or a version such as `versions[\"v2\"]` " +
			"to `building_block_definition_version_ref` of a building block. " +
			"This data source performs an additional API call to load all versions of the definition." +
			"\n\n" +
			"**Cross-Workspace Access**: When accessing building block definitions from workspaces other than your own " +
			"using a workspace-scoped API key, the `content_hash` attribute will be null as it requires detailed version " +
			"information that is not accessible without workspace permissions. The `versions`, `version_latest`, and " +
			"`version_latest_release` attributes will still be populated with uuid, number, state, and workload_identity_federation." +
			"\n\n" + previewDisclaimer(),
		Attributes: attributes,
	}
}

func (d *buildingBlockDefinitionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var filter buildingBlockDefinitionFilter
	for attributePath, target := range map[string]**string{
		"metadata.uuid":               &filter.Uuid,
		"metadata.owned_by_workspace": &filter.OwnedByWorkspace,
		"spec.display_name":           &filter.DisplayName,
		"spec.target_type":            &filter.TargetType,
	} {
		block, attribute, _ := strings.Cut(attributePath, ".")
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(block).AtName(attribute), target)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	candidates := d.candidates(ctx, filter, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	definition := filter.only(candidates, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	model := buildingBlockDefinitionDataSourceModelFrom(ctx, d.meshBuildingBlockDefinitionVersionClient, definition, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (d *buildingBlockDefinitionDataSource) candidates(ctx context.Context, filter buildingBlockDefinitionFilter, diags *diag.Diagnostics) []client.MeshBuildingBlockDefinition {
	if filter.Uuid == nil {
		definitions, err := d.meshBuildingBlockDefinitionClient.List(ctx, filter.OwnedByWorkspace)
		if err != nil {
			diags.AddError("Unable to list building block definitions", err.Error())
			return nil
		}
		for _, definition := range definitions {
			if definition.Metadata.Uuid == nil {
				diags.AddError(
					"Building block definition UUID missing",
					"API returned a building block definition without metadata.uuid, which cannot be represented in this data source.",
				)
				return nil
			}
		}
		return definitions
	}
	definition, err := d.meshBuildingBlockDefinitionClient.Read(ctx, *filter.Uuid)
	if httpErr, isHttpErr := errors.AsType[client.HttpError](err); isHttpErr && httpErr.IsForbidden() {
		return nil
	}
	if err != nil {
		diags.AddError("Unable to read building block definition", err.Error())
	}
	if definition == nil {
		return nil
	}
	return []client.MeshBuildingBlockDefinition{*definition}
}

// only checks owned_by_workspace on listed definitions too, because the list always holds all
// published definitions next to the workspace's own.
func (filter buildingBlockDefinitionFilter) only(candidates []client.MeshBuildingBlockDefinition, diags *diag.Diagnostics) (definition client.MeshBuildingBlockDefinition) {
	var matches []client.MeshBuildingBlockDefinition
	for _, candidate := range candidates {
		if filter.matches(candidate) {
			matches = append(matches, candidate)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0]
	case 0:
		diags.AddError("Building block definition not found",
			fmt.Sprintf("No building block definition that you can see matches %s.", filter))
	default:
		found := make([]string, len(matches))
		for i, match := range matches {
			found[i] = fmt.Sprintf("%s (%q in workspace %s)", *match.Metadata.Uuid, match.Spec.DisplayName, match.Metadata.OwnedByWorkspace)
		}
		diags.AddError("Building block definition is ambiguous",
			fmt.Sprintf("%d building block definitions match %s: %s. Set `metadata.uuid` to pick exactly one.",
				len(matches), filter, strings.Join(found, ", ")))
	}
	return
}

func (filter buildingBlockDefinitionFilter) matches(definition client.MeshBuildingBlockDefinition) bool {
	return (filter.Uuid == nil || definition.Metadata.Uuid != nil && *definition.Metadata.Uuid == *filter.Uuid) &&
		(filter.OwnedByWorkspace == nil || definition.Metadata.OwnedByWorkspace == *filter.OwnedByWorkspace) &&
		(filter.DisplayName == nil || definition.Spec.DisplayName == *filter.DisplayName) &&
		(filter.TargetType == nil || string(definition.Spec.TargetType) == *filter.TargetType)
}

func (filter buildingBlockDefinitionFilter) String() string {
	var set []string
	for _, attribute := range []struct {
		name  string
		value *string
	}{
		{"metadata.uuid", filter.Uuid},
		{"metadata.owned_by_workspace", filter.OwnedByWorkspace},
		{"spec.display_name", filter.DisplayName},
		{"spec.target_type", filter.TargetType},
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
