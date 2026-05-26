package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ resource.Resource                   = &buildingBlockRunnerResource{}
	_ resource.ResourceWithConfigure      = &buildingBlockRunnerResource{}
	_ resource.ResourceWithValidateConfig = &buildingBlockRunnerResource{}
	_ resource.ResourceWithImportState    = &buildingBlockRunnerResource{}
)

func NewBuildingBlockRunnerResource() resource.Resource {
	return &buildingBlockRunnerResource{}
}

type buildingBlockRunnerResource struct {
	client client.MeshBuildingBlockRunnerClient
}

func (r *buildingBlockRunnerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_runner"
}

func (r *buildingBlockRunnerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(providerClient client.Client) {
		r.client = providerClient.BuildingBlockRunner
	})...)
}

type buildingBlockRunnerModel struct {
	client.MeshBuildingBlockRunner
	Ref struct {
		Kind string `tfsdk:"kind"`
		Uuid string `tfsdk:"uuid"`
	} `tfsdk:"ref"`
}

func (model *buildingBlockRunnerModel) setFromClientDto(dto *client.MeshBuildingBlockRunner) {
	model.MeshBuildingBlockRunner = *dto
	model.Ref.Kind = client.MeshObjectKind.BuildingBlockRunner
	model.Ref.Uuid = *dto.Metadata.Uuid
}

func (r *buildingBlockRunnerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	nonEmptyStringValidator := stringvalidator.RegexMatches(
		regexp.MustCompile(`\S`),
		"Value must not be empty or whitespace.",
	)

	wifProviderAttributes := map[string]schema.Attribute{
		"audience": schema.StringAttribute{
			MarkdownDescription: "Audience value for the federated identity token.",
			Required:            true,
			Validators: []validator.String{
				nonEmptyStringValidator,
			},
		},
		"token_path": schema.StringAttribute{
			MarkdownDescription: "Path to the federated identity token file on the runner.",
			Required:            true,
			Validators: []validator.String{
				nonEmptyStringValidator,
			},
		},
	}

	wifAttributes := map[string]schema.Attribute{
		"subject": schema.StringAttribute{
			MarkdownDescription: "The subject claim of the OIDC token issued to this runner, e.g., `system:serviceaccount:namespace:my-runner`.",
			Required:            true,
			Validators: []validator.String{
				nonEmptyStringValidator,
			},
		},
		"issuer": schema.StringAttribute{
			MarkdownDescription: "The OIDC issuer URL of the identity provider that issues tokens for this runner. This is used to configure trust with the target cloud provider.",
			Required:            true,
			Validators: []validator.String{
				nonEmptyStringValidator,
			},
		},
		"gcp": schema.SingleNestedAttribute{
			MarkdownDescription: "GCP workload identity federation configuration.",
			Optional:            true,
			Attributes:          wifProviderAttributes,
		},
		"aws": schema.SingleNestedAttribute{
			MarkdownDescription: "AWS workload identity federation configuration.",
			Optional:            true,
			Attributes:          wifProviderAttributes,
		},
		"azure": schema.SingleNestedAttribute{
			MarkdownDescription: "Azure workload identity federation configuration.",
			Optional:            true,
			Attributes:          wifProviderAttributes,
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a meshBuildingBlockRunner in meshStack. " +
			"Building block runners are agents that execute building block runs. " +
			previewDisclaimer(),

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata of the building block runner.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "Unique identifier of the runner. Assigned by meshStack on creation.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace that owns this runner.",
						Required:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp when the runner was created (ISO 8601).",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"last_seen": schema.StringAttribute{
						MarkdownDescription: "Timestamp when the runner last connected to meshStack (ISO 8601).",
						Computed:            true,
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Specification of the building block runner.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Human-readable display name of the runner.",
						Required:            true,
					},
					"public_key": schema.StringAttribute{
						MarkdownDescription: "RSA public key in PEM format (`BEGIN PUBLIC KEY`) or an X.509 certificate (`BEGIN CERTIFICATE`). meshStack uses this key to encrypt secrets sent to the runner.",
						Required:            true,
					},
					"implementation_type": schema.StringAttribute{
						MarkdownDescription: "Type of building block implementation this runner handles. One of: `TERRAFORM`, `GITHUB_WORKFLOW`, `GITLAB_PIPELINE`, `AZURE_DEVOPS_PIPELINE`, `MANUAL`.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							// TODO: Switch to in-place update once meshStack supports updating implementation type.
							// Keep replacement semantics until then to prevent plan/apply drift.
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.OneOf(client.MeshBuildingBlockRunnerImplementationTypes...),
						},
					},
					"restriction": schema.StringAttribute{
						MarkdownDescription: "Visibility restriction of the runner. `PUBLIC` makes the runner available to all workspaces. `PRIVATE` (default) restricts it to the owning workspace. Only administrators can set `PUBLIC`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("PRIVATE"),
						PlanModifiers: []planmodifier.String{
							// TODO: Switch to in-place update once meshStack supports updating restriction.
							// Keep replacement semantics until then to prevent plan/apply drift.
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.OneOf("PUBLIC", "PRIVATE"),
						},
					},
					"is_self_hosted": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether the runner is self-hosted. This field is read-only and set by meshStack.",
						Computed:            true,
					},
					"workload_identity_federation": schema.SingleNestedAttribute{
						MarkdownDescription: "Optional workload identity federation configuration. When provided, at least one of `gcp`, `aws`, or `azure` must be set.",
						Optional:            true,
						Attributes:          wifAttributes,
					},
				},
			},
			"ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Reference to this runner, can be used in building block definitions.",
				Computed:            true,
				Attributes:          meshUuidRefOutputAttribute(client.MeshObjectKind.BuildingBlockRunner),
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *buildingBlockRunnerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var wif types.Object

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("workload_identity_federation"), &wif)...)
	if resp.Diagnostics.HasError() || wif.IsNull() || wif.IsUnknown() {
		return
	}

	attributes := wif.Attributes()
	hasKnownProvider := false
	hasUnknownProvider := false
	for _, providerName := range []string{"gcp", "aws", "azure"} {
		providerValue, ok := attributes[providerName]
		if !ok {
			continue
		}
		if providerValue.IsUnknown() {
			hasUnknownProvider = true
			continue
		}
		if !providerValue.IsNull() {
			hasKnownProvider = true
		}
	}

	if hasKnownProvider || hasUnknownProvider {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("spec").AtName("workload_identity_federation"),
		"Invalid workload identity federation configuration",
		"At least one provider configuration must be set: `gcp`, `aws`, or `azure`.",
	)
}

func (r *buildingBlockRunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := generic.Get[buildingBlockRunnerModel](ctx, req.Plan, &resp.Diagnostics, generic.WithSetUnknownValueToZero())
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.Create(ctx, plan.MeshBuildingBlockRunner)
	if err != nil {
		resp.Diagnostics.AddError("Error creating meshBuildingBlockRunner", err.Error())
		return
	}
	plan.setFromClientDto(created)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan)...)
}

func (r *buildingBlockRunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	uuid := generic.GetAttribute[string](ctx, req.State, path.Root("metadata").AtName("uuid"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	runner, err := r.client.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading meshBuildingBlockRunner", fmt.Sprintf("Reading runner '%s' failed: %s", uuid, err.Error()))
		return
	} else if runner == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	var state buildingBlockRunnerModel
	state.setFromClientDto(runner)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, state)...)
}

func (r *buildingBlockRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := generic.Get[buildingBlockRunnerModel](ctx, req.Plan, &resp.Diagnostics, generic.WithSetUnknownValueToZero())
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.Update(ctx, plan.MeshBuildingBlockRunner)
	if err != nil {
		resp.Diagnostics.AddError("Error updating meshBuildingBlockRunner", fmt.Sprintf("Updating runner '%s' failed: %s", *plan.Metadata.Uuid, err.Error()))
		return
	}
	plan.setFromClientDto(updated)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan)...)
}

func (r *buildingBlockRunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	uuid := generic.GetAttribute[string](ctx, req.State, path.Root("metadata").AtName("uuid"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, uuid); err != nil {
		resp.Diagnostics.AddError("Error deleting meshBuildingBlockRunner", fmt.Sprintf("Deleting runner '%s' failed: %s", uuid, err.Error()))
	}
}

func (r *buildingBlockRunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
}
