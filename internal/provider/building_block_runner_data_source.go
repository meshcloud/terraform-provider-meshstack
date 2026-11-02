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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ datasource.DataSource              = &buildingBlockRunnerDataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlockRunnerDataSource{}
)

func NewBuildingBlockRunnerDataSource() datasource.DataSource {
	return &buildingBlockRunnerDataSource{}
}

type buildingBlockRunnerDataSource struct {
	client client.MeshBuildingBlockRunnerClient
}

type buildingBlockRunnerFilter struct {
	Uuid             *string
	OwnedByWorkspace *string
	DisplayName      *string
}

func (d *buildingBlockRunnerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_runner"
}

func (d *buildingBlockRunnerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(providerClient client.Client) {
		d.client = providerClient.BuildingBlockRunner
	})...)
}

func (d *buildingBlockRunnerDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	var runnerResource resource.SchemaResponse
	NewBuildingBlockRunnerResource().Schema(ctx, resource.SchemaRequest{}, &runnerResource)
	resourceAttributes := runnerResource.Schema.Attributes

	resp.Schema = schema.Schema{
		MarkdownDescription: "Read the one building block runner that matches what you set in `metadata` and `spec`, " +
			"such as `spec.display_name`. Without any of `metadata.uuid`, `metadata.owned_by_workspace` and `spec.display_name`, " +
			"it reads the runner meshStack hosts. " +
			"It searches your workspace's own and all shared runners, and fails unless exactly one of them matches. " +
			"It is read-only and mirrors the `meshstack_building_block_runner` resource." +
			previewDisclaimer(),
		Attributes: map[string]schema.Attribute{
			"metadata": asFilter(computedDataSourceAttribute(resourceAttributes["metadata"]), "uuid", "owned_by_workspace"),
			"spec":     asFilter(computedDataSourceAttribute(resourceAttributes["spec"]), "display_name"),
			"status":   computedDataSourceAttribute(resourceAttributes["status"]),
			"ref":      computedDataSourceAttribute(resourceAttributes["ref"]),
		},
	}
}

func (d *buildingBlockRunnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var filter buildingBlockRunnerFilter
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &filter.Uuid)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &filter.OwnedByWorkspace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("display_name"), &filter.DisplayName)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if filter == (buildingBlockRunnerFilter{}) {
		filter.Uuid = new(SharedBuildingBlockRunnerUuid)
	}

	candidates := d.candidates(ctx, filter, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	runner := filter.only(candidates, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var state buildingBlockRunnerModel
	state.setFromClientDto(&runner)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, state)...)
}

func (d *buildingBlockRunnerDataSource) candidates(ctx context.Context, filter buildingBlockRunnerFilter, diags *diag.Diagnostics) []client.MeshBuildingBlockRunner {
	if filter.Uuid == nil {
		runners, err := d.client.List(ctx, client.MeshBuildingBlockRunnerFilter{WorkspaceIdentifier: filter.OwnedByWorkspace})
		if err != nil {
			diags.AddError("Unable to list building block runners", err.Error())
			return nil
		}
		for _, runner := range runners {
			if runner.Metadata.Uuid == nil {
				diags.AddError(
					"Building block runner UUID missing",
					"API returned a building block runner without metadata.uuid, which cannot be represented in this data source.",
				)
				return nil
			}
		}
		return runners
	}
	runner, err := d.client.Read(ctx, *filter.Uuid)
	if httpErr, isHttpErr := errors.AsType[client.HttpError](err); isHttpErr && httpErr.IsForbidden() {
		return nil
	}
	if err != nil {
		diags.AddError("Unable to read building block runner", err.Error())
	}
	if runner == nil {
		return nil
	}
	return []client.MeshBuildingBlockRunner{*runner}
}

func (filter buildingBlockRunnerFilter) only(candidates []client.MeshBuildingBlockRunner, diags *diag.Diagnostics) client.MeshBuildingBlockRunner {
	return onlyMatch("building block runner", filter, candidates, filter.matches, func(runner client.MeshBuildingBlockRunner) string {
		return fmt.Sprintf("%s (%q in workspace %s)", *runner.Metadata.Uuid, runner.Spec.DisplayName, runner.Metadata.OwnedByWorkspace)
	}, diags)
}

func (filter buildingBlockRunnerFilter) matches(runner client.MeshBuildingBlockRunner) bool {
	return (filter.Uuid == nil || runner.Metadata.Uuid != nil && *runner.Metadata.Uuid == *filter.Uuid) &&
		(filter.OwnedByWorkspace == nil || runner.Metadata.OwnedByWorkspace == *filter.OwnedByWorkspace) &&
		(filter.DisplayName == nil || runner.Spec.DisplayName == *filter.DisplayName)
}

func (filter buildingBlockRunnerFilter) String() string {
	var set []string
	for _, attribute := range []struct {
		name  string
		value *string
	}{
		{"metadata.uuid", filter.Uuid},
		{"metadata.owned_by_workspace", filter.OwnedByWorkspace},
		{"spec.display_name", filter.DisplayName},
	} {
		if attribute.value != nil {
			set = append(set, fmt.Sprintf("%s = %q", attribute.name, *attribute.value))
		}
	}
	return strings.Join(set, " and ")
}
