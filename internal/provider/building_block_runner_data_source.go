package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
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

// buildingBlockRunnerDataSourceModel drops the runner's public key: it is an input of the resource,
// not something a consumer of the runner needs.
type buildingBlockRunnerDataSourceModel struct {
	Metadata buildingBlockRunnerDataSourceMetadata `tfsdk:"metadata"`
	Spec     buildingBlockRunnerDataSourceSpec     `tfsdk:"spec"`
	Ref      client.UuidRef                        `tfsdk:"ref"`
}

type buildingBlockRunnerDataSourceMetadata struct {
	Uuid             string  `tfsdk:"uuid"`
	OwnedByWorkspace string  `tfsdk:"owned_by_workspace"`
	CreatedOn        *string `tfsdk:"created_on"`
	LastSeen         *string `tfsdk:"last_seen"`
}

type buildingBlockRunnerDataSourceSpec struct {
	DisplayName                string                            `tfsdk:"display_name"`
	ImplementationType         string                            `tfsdk:"implementation_type"`
	Restriction                *string                           `tfsdk:"restriction"`
	IsSelfHosted               *bool                             `tfsdk:"is_self_hosted"`
	WorkloadIdentityFederation *buildingBlockRunnerDataSourceWif `tfsdk:"workload_identity_federation"`
}

// buildingBlockRunnerDataSourceWif reports the identity scheme as a template: the resolved subject
// belongs to a definition, not to the runner.
type buildingBlockRunnerDataSourceWif struct {
	Issuer          *string                             `tfsdk:"issuer"`
	SubjectTemplate *string                             `tfsdk:"subject_template"`
	Gcp             *client.MeshRunnerWifProviderConfig `tfsdk:"gcp"`
	Aws             *client.MeshRunnerWifProviderConfig `tfsdk:"aws"`
	Azure           *client.MeshRunnerWifProviderConfig `tfsdk:"azure"`
}

func buildingBlockRunnerDataSourceModelFromDto(dto *client.MeshBuildingBlockRunner) buildingBlockRunnerDataSourceModel {
	model := buildingBlockRunnerDataSourceModel{
		Metadata: buildingBlockRunnerDataSourceMetadata{
			Uuid:             *dto.Metadata.Uuid,
			OwnedByWorkspace: dto.Metadata.OwnedByWorkspace,
			CreatedOn:        dto.Metadata.CreatedOn,
			LastSeen:         dto.Metadata.LastSeen,
		},
		Spec: buildingBlockRunnerDataSourceSpec{
			DisplayName:        dto.Spec.DisplayName,
			ImplementationType: dto.Spec.ImplementationType,
			Restriction:        dto.Spec.Restriction,
			IsSelfHosted:       dto.Spec.IsSelfHosted,
		},
		Ref: client.UuidRef{Kind: client.MeshObjectKind.BuildingBlockRunner, Uuid: *dto.Metadata.Uuid},
	}
	if wif := dto.Spec.WorkloadIdentityFederation; wif != nil {
		model.Spec.WorkloadIdentityFederation = &buildingBlockRunnerDataSourceWif{
			Issuer:          wif.Issuer,
			SubjectTemplate: wif.SubjectTemplate,
			Gcp:             wif.Gcp,
			Aws:             wif.Aws,
			Azure:           wif.Azure,
		}
	}
	return model
}

func (d *buildingBlockRunnerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_runner"
}

func (d *buildingBlockRunnerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(providerClient client.Client) {
		d.client = providerClient.BuildingBlockRunner
	})...)
}

func (d *buildingBlockRunnerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	wifProviderAttributes := func(cloud string) schema.SingleNestedAttribute {
		return schema.SingleNestedAttribute{
			MarkdownDescription: "Workload identity federation values for " + cloud + ". Null when the runner does not federate with " + cloud + ".",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"audience": schema.StringAttribute{
					MarkdownDescription: "Audience the federated identity token is issued for. Trust this value in your " + cloud + " backplane.",
					Computed:            true,
				},
				"token_path": schema.StringAttribute{
					MarkdownDescription: "Path the runner writes that token to.",
					Computed:            true,
				},
			},
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single building block runner by UUID, defaulting to the shared runner meshStack hosts. " +
			"Use it to configure the trust of a cloud backplane that exists before any building block definition — the shared AWS OIDC provider, " +
			"a GCP workload identity pool provider — from `spec.workload_identity_federation.issuer` and the per-cloud `audience`.\n\n" +
			"Read the **subject** from the definition instead: `spec.workload_identity_federation.subject_template` is a template, and " +
			"meshStack resolves it per building block definition into " +
			"`meshstack_building_block_definition.<name>.status.workload_identity_federation.subject`. " +
			"A module that grants a definition access to a cloud takes the subject from there and only issuer and audience from here, " +
			"so trust and execution cannot drift apart when the definition moves to another runner.\n\n" +
			"The template understands two placeholders, `{{ workspaceIdentifier }}` and `{{ buildingBlockDefinitionUuid }}`; " +
			"a template without placeholders is a fixed subject, which is what a runner minting one identity for all of its runs declares." +
			previewDisclaimer(),

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata of the building block runner. Omit it to read the shared runner.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the runner to read. Defaults to `" + client.SharedBuildingBlockRunnerUuid + "`, the shared runner meshStack hosts, " +
							"which is also the runner a building block definition uses when its `version_spec.runner_ref` is omitted.",
						Optional: true,
						Computed: true,
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace that owns this runner.",
						Computed:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp when the runner was created (ISO 8601).",
						Computed:            true,
					},
					"last_seen": schema.StringAttribute{
						MarkdownDescription: "Timestamp when the runner last connected to meshStack (ISO 8601).",
						Computed:            true,
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Specification of the building block runner.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Human-readable display name of the runner.",
						Computed:            true,
					},
					"implementation_type": schema.StringAttribute{
						MarkdownDescription: "Type of building block implementation this runner handles. One of: `" +
							"TERRAFORM`, `GITHUB_WORKFLOW`, `GITLAB_PIPELINE`, `AZURE_DEVOPS_PIPELINE`, `MANUAL`, `ALL`.",
						Computed: true,
					},
					"restriction": schema.StringAttribute{
						MarkdownDescription: "Visibility restriction of the runner. `PUBLIC` makes it available to all workspaces, `PRIVATE` to the owning workspace only.",
						Computed:            true,
					},
					"is_self_hosted": schema.BoolAttribute{
						MarkdownDescription: "Whether the runner is hosted by you rather than by meshStack.",
						Computed:            true,
					},
					"workload_identity_federation": schema.SingleNestedAttribute{
						MarkdownDescription: "The identity scheme this runner declares. Null when the runner presents no identity of its own.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"issuer": schema.StringAttribute{
								MarkdownDescription: "OIDC issuer URL of the identity provider that issues tokens for this runner's building block runs. Trust it in your cloud backplane.",
								Computed:            true,
							},
							"subject_template": schema.StringAttribute{
								MarkdownDescription: subjectTemplateDescription("The template meshStack resolves into the subject claim of the tokens this runner presents"),
								Computed:            true,
							},
							"gcp":   wifProviderAttributes("GCP"),
							"aws":   wifProviderAttributes("AWS"),
							"azure": wifProviderAttributes("Azure"),
						},
					},
				},
			},
			"ref": meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.BuildingBlockRunner, Description: "Reference to this runner, can be used as `version_spec.runner_ref` of a building block definition.", Output: true}),
		},
	}
}

func (d *buildingBlockRunnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var configuredUuid types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &configuredUuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := client.SharedBuildingBlockRunnerUuid
	if !configuredUuid.IsNull() && !configuredUuid.IsUnknown() && configuredUuid.ValueString() != "" {
		uuid = configuredUuid.ValueString()
	}

	runner, err := d.client.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block runner", fmt.Sprintf("Reading runner '%s' failed: %s", uuid, err.Error()))
		return
	}
	if runner == nil {
		resp.Diagnostics.AddError("Building block runner not found", fmt.Sprintf("No building block runner with UUID '%s' exists, or your API key may not read it.", uuid))
		return
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, buildingBlockRunnerDataSourceModelFromDto(runner))...)
}
