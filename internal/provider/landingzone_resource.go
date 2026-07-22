package provider

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/modifiers/platformtypemodifier"
	"github.com/meshcloud/terraform-provider-meshstack/internal/validators"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &landingZoneResource{}
	_ resource.ResourceWithConfigure    = &landingZoneResource{}
	_ resource.ResourceWithImportState  = &landingZoneResource{}
	_ resource.ResourceWithUpgradeState = &landingZoneResource{}
)

// NewLandingZoneResource is a helper function to simplify the provider implementation.
func NewLandingZoneResource() resource.Resource {
	return &landingZoneResource{}
}

// landingZoneResource is the resource implementation.
type landingZoneResource struct {
	meshLandingZoneClient client.MeshLandingZoneClient
}

// landingZoneRefOutput is the computed self-`ref` of a landing zone (name-based).
type landingZoneRefOutput struct {
	Name string `tfsdk:"name"`
	Kind string `tfsdk:"kind"`
}

// landingZoneModel is the state model: the API's landing zone fields plus the computed self-`ref`.
// The client struct has no `ref` field, so we wrap it here rather than mutating client types.
type landingZoneModel struct {
	Ref      landingZoneRefOutput           `tfsdk:"ref"`
	Metadata client.MeshLandingZoneMetadata `tfsdk:"metadata"`
	Spec     client.MeshLandingZoneSpec     `tfsdk:"spec"`
	Status   client.MeshLandingZoneStatus   `tfsdk:"status"`
}

func landingZoneModelFrom(lz *client.MeshLandingZone) landingZoneModel {
	return landingZoneModel{
		Ref:      landingZoneRefOutput{Name: lz.Metadata.Name, Kind: client.MeshObjectKind.LandingZone},
		Metadata: lz.Metadata,
		Spec:     lz.Spec,
		Status:   lz.Status,
	}
}

// Metadata returns the resource type name.
func (r *landingZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_landingzone"
}

// Configure adds the provider configured client to the resource.
func (r *landingZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshLandingZoneClient = client.LandingZone
	})...)
}

func (r *landingZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	quotas := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				MarkdownDescription: "Quota key identifier. Must match a quota key that has been defined on the platform.",
				Required:            true,
			},
			"value": schema.Int64Attribute{
				MarkdownDescription: "Quota value.",
				Required:            true,
			},
		},
	}

	buildingBlockRef := meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.BuildingBlockDefinition, InSet: true})
	buildingBlockRefs := schema.NestedAttributeObject{
		Attributes: buildingBlockRef.Attributes,
		Validators: buildingBlockRef.Validators,
	}

	resp.Schema = schema.Schema{
		// v1 corrected metadata.tags from set(string) to list(string) — see UpgradeState.
		Version:             1,
		MarkdownDescription: "Represents a meshStack landing zone.",
		Attributes: map[string]schema.Attribute{
			"ref": meshRefByName(meshRefOptions{
				Kind:        client.MeshObjectKind.LandingZone,
				Description: "Reference to this landing zone, can be used as `landing_zone_ref` in tenant resources. The landing zone name is only unique together with its platform, so a `meshstack_tenant` references both `platform_ref` and `landing_zone_ref`.",
				Output:      true,
			}),

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Landing zone identifier.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`),
								"must be alphanumeric with dashes, must be lowercase, and have no leading, trailing or consecutive dashes",
							),
						},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace that owns this landing zone.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"tags": tagsAttribute(tagsOptions{Kind: client.MeshObjectKind.LandingZone, Restricted: true}),
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the landing zone.",
						Required:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "description of the landing zone.",
						Required:            true,
					},
					"automate_deletion_approval": schema.BoolAttribute{
						MarkdownDescription: "Whether deletion approval is automated for this landing zone.",
						Required:            true,
					},
					"automate_deletion_replication": schema.BoolAttribute{
						MarkdownDescription: "Whether deletion replication is automated for this landing zone.",
						Required:            true,
					},
					"info_link": schema.StringAttribute{
						MarkdownDescription: "Link to additional information about the landing zone.",
						Optional:            true,
					},
					"platform_ref": meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.Platform, Description: "Reference to the platform this landing zone belongs to.", RequiresReplace: true}),
					"platform_properties": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform-specific configuration options.",
						Required:            true,
						Sensitive:           false,
						Validators: []validator.Object{
							validators.ExactlyOneAttributeValidator{},
						},
						Attributes: map[string]schema.Attribute{
							"aws":        awsPlatformConfigSchema(),
							"aks":        aksPlatformConfigSchema(),
							"azure":      azurePlatformConfigSchema(),
							"azurerg":    azureRgPlatformConfigSchema(),
							"custom":     customPlatformConfigSchema(),
							"gcp":        gcpPlatformConfigSchema(),
							"kubernetes": kubernetesPlatformConfigSchema(),
							"openshift":  openShiftPlatformConfigSchema(),
							"type": schema.StringAttribute{
								MarkdownDescription: "Type of the platform. This field is automatically inferred from which platform configuration is provided and cannot be set manually.",
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.RequiresReplace(),
									platformtypemodifier.SetTypeFromPlatform(),
								},
							},
						},
					},
					"quotas": schema.SetNestedAttribute{
						MarkdownDescription: "Quota definitions for this landing zone.",
						Optional:            true,
						Computed:            true,
						Default:             emptySetDefault(quotas),
						NestedObject:        quotas,
					},
					"mandatory_building_block_refs": schema.SetNestedAttribute{
						MarkdownDescription: "List of mandatory building block references for this landing zone.",
						Optional:            true,
						Computed:            true,
						Default:             emptySetDefault(buildingBlockRefs),
						NestedObject:        buildingBlockRefs,
					},
					"recommended_building_block_refs": schema.SetNestedAttribute{
						MarkdownDescription: "List of recommended building block references for this landing zone.",
						Optional:            true,
						Computed:            true,
						Default:             emptySetDefault(buildingBlockRefs),
						NestedObject:        buildingBlockRefs,
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Current Landing Zone status.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"disabled": schema.BoolAttribute{
						MarkdownDescription: "True if the landing zone is disabled.",
						Computed:            true,
					},
					"restricted": schema.BoolAttribute{
						MarkdownDescription: "If true, users will be unable to select this landing zone in meshPanel. " +
							"Only Platform teams can create tenants using restricted landing zones with the meshObject API.",
						Computed: true,
					},
				},
			},
		},
	}
}

func awsPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "AWS platform properties.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"aws_target_org_unit_id": schema.StringAttribute{
				MarkdownDescription: "The created AWS account for this Landing Zone will be put under the given Organizational Unit. You can also input a Root ID (starting with 'r-') then the account will be put directly under this root without assigning it to an OU (this is not recommended).",
				Required:            true,
			},
			"aws_enroll_account": schema.BoolAttribute{
				MarkdownDescription: "If true, accounts will be enrolled to AWS control tower. In case an enrollment configuration is provided for the AWS platform AND this value is set to true, created AWS accounts will automatically be enrolled with AWS Control Tower. Automatic account enrollment does also require the Target Organizational Unit to already be enrolled with AWS Control Tower and the corresponding meshfed-service role needs to be in the \"IAM Principal\" list for the Portfolio access of the Account Factory Product ID you defined in platform settings. Click [here](https://docs.meshcloud.io/integrations/aws/how-to-integrate/#7-integrate-aws-control-tower) to learn more about the Control Tower setup.",
				Required:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"aws_lambda_arn": schema.StringAttribute{
				MarkdownDescription: "If provided, it is invoked after each project replication. You can use it to trigger a custom Account Vending Machine to perform several additional provisioning steps.",
				Optional:            true,
			},
			"aws_role_mappings": schema.SetNestedAttribute{
				MarkdownDescription: "Roles can be mapped from the meshRole to the AWS Role. The AWS role will be part of the role or group name within AWS. If empty, the default that is configured on platform level will be used.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.ProjectRole, Description: "Reference to the meshProjectRole.", InSet: true}),
						"platform_role": schema.StringAttribute{
							MarkdownDescription: "The AWS platform role",
							Required:            true,
						},
						"policies": schema.SetAttribute{
							MarkdownDescription: "List of policies associated with this role mapping",
							ElementType:         types.StringType,
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func aksPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "AKS platform properties.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"kubernetes_role_mappings": schema.SetNestedAttribute{
				MarkdownDescription: "Roles need to be mapped from the meshRole to the Cluster Role. You can use " +
					"both built in roles like 'editor' or custom roles that you setup in the Kubernetes Cluster " +
					"before. For more information see [the Landing Zone documentation](https://docs.meshcloud.io/meshstack.kubernetes.landing-zones/).",
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.ProjectRole, Description: "Reference to the meshProjectRole.", InSet: true}),
						"platform_roles": schema.SetAttribute{
							MarkdownDescription: "List of AKS platform roles to assign to the meshProject role.",
							ElementType:         types.StringType,
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func azurePlatformConfigSchema() schema.Attribute {
	azureRoleMappings := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"project_role_ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.ProjectRole, Description: "Reference to the meshProjectRole.", InSet: true}),
			"azure_group_suffix": schema.StringAttribute{
				MarkdownDescription: "The given role name will be injected into the" +
					" group name via the group naming pattern configured on the" +
					" platform instance.",
				Required: true,
			},
			"azure_role_definitions": schema.SetNestedAttribute{
				MarkdownDescription: "List of Azure role definitions",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"azure_role_definition_id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Azure role definition ID",
						},
						"abac_condition": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "an ABAC condition for the role assignment in form of a string",
						},
					},
				},
			},
		},
	}

	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure platform properties.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"azure_management_group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Azure Management Group ID where projects will be created.",
			},
			"azure_role_mappings": schema.SetNestedAttribute{
				MarkdownDescription: "An array of mappings between the meshRole and the Azure" +
					" specific access role. " +
					"For more information see [the Landing Zone documentation](https://docs.meshcloud.io/meshstack.azure.landing-zones#meshrole-to-platform-role-mapping). " +
					"If empty, the default that is configured on platform level will be used.",
				Optional:     true,
				Computed:     true,
				Default:      emptySetDefault(azureRoleMappings),
				NestedObject: azureRoleMappings,
			},
		},
	}
}

func gcpPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "GCP platform properties.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"gcp_cloud_function_url": schema.StringAttribute{
				MarkdownDescription: "If a GCP Cloud Function URL is provided it is getting called at the end of the replication process.",
				Optional:            true,
			},
			"gcp_folder_id": schema.StringAttribute{
				MarkdownDescription: "Google Cloud Projects will be added to this Google Cloud Folder. This allows applying Organization Policies to all projects managed under this Landing Zone.",
				Optional:            true,
			},
			"gcp_role_mappings": schema.SetNestedAttribute{
				MarkdownDescription: "You can use both built-in roles like 'roles/editor' or" +
					" custom roles like 'organizations/123123123123/roles/meshstack." +
					"project_developer'. For more information see " +
					"[the Landing Zone documentation](https://docs.meshcloud.io/meshstack.gcp.landing-zones/#meshrole-to-platform-role-mapping). Multiple GCP Roles can be assigned to one meshRole. If empty, " +
					"the default that is configured on platform level will be used.",
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.ProjectRole, Description: "Reference to the meshProjectRole.", InSet: true}),
						"platform_roles": schema.SetAttribute{
							MarkdownDescription: "Can be empty. List of GCP IAM roles to assign to the meshProject role.",
							ElementType:         types.StringType,
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func azureRgPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Resource Group platform properties.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"azure_rg_location": schema.StringAttribute{
				MarkdownDescription: "The newly created Resource Group for the meshProjects will get assigned to this location. It must be all lower case and without spaces (e.g. `eastus2` for East US 2). In order to list the available locations you can use `az account list-locations --query \"[*].name\" --out tsv | sort`",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"azure_rg_role_mappings": schema.SetNestedAttribute{
				MarkdownDescription: "An array of mappings between the meshRole and the Azure" +
					" specific access role. " +
					"For more information see [the Landing Zone documentation](https://docs.meshcloud.io/meshstack.azure.landing-zones#meshrole-to-platform-role-mapping). " +
					"If empty, the default that is configured on platform level will be used.",
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.ProjectRole, Description: "Reference to the meshProjectRole.", InSet: true}),
						"azure_group_suffix": schema.StringAttribute{
							MarkdownDescription: "The given role name will be injected into the" +
								" group name via the group naming pattern configured on the" +
								" platform instance.",
							Required: true,
						},
						"azure_role_definition_ids": schema.SetAttribute{
							MarkdownDescription: "Role Definitions with the given IDs will be attached to this Azure Role.",
							ElementType:         types.StringType,
							Required:            true,
						},
					},
				},
			},
			"azure_function": schema.SingleNestedAttribute{
				MarkdownDescription: "Assign an Azure function to the Landing Zone configuration to trigger a small piece of code in the cloud.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"azure_function_url": schema.StringAttribute{
						MarkdownDescription: "The URL of your Azure Function. This is typically a value like https://my-function-app.azurewebsites.net/myfunc",
						Required:            true,
					},
					"azure_function_scope": schema.StringAttribute{
						MarkdownDescription: "The unique ID of the Azure Enterprise Application your function belongs to. More details are described [here](https://docs.meshcloud.io/docs/meshstack.azure.landing-zones.html#azure-function-invocation).",
						Required:            true,
					},
				},
			},
		},
	}
}

func kubernetesPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Kubernetes platform properties.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"kubernetes_role_mappings": schema.SetNestedAttribute{
				MarkdownDescription: "Kubernetes role mappings configuration.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.ProjectRole, Description: "Reference to the meshProjectRole.", InSet: true}),
						"platform_roles": schema.SetAttribute{
							MarkdownDescription: "Roles need to be mapped from the meshRole to" +
								" the Cluster Role. You can use both built in roles like 'editor' or custom roles that you setup in the Kubernetes Cluster" +
								" before. For more information see [the Landing Zone documentation](https://docs.meshcloud.io/meshstack.kubernetes.landing-zones/).",
							ElementType: types.StringType,
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func openShiftPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "OpenShift platform properties.",
		Optional:            true,
		Attributes:          map[string]schema.Attribute{},
	}
}

func customPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Custom platform properties. Custom platforms do not require any platform-specific configuration properties, so this is intentionally an empty object (`{}`). Simply set `custom = {}` in your Terraform configuration.",
		Optional:            true,
		Attributes:          map[string]schema.Attribute{},
	}
}

func (r *landingZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	landingZone := client.MeshLandingZoneCreate{
		Metadata: client.MeshLandingZoneMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &landingZone.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &landingZone.Metadata)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createdLandingZone, err := r.meshLandingZoneClient.Create(ctx, &landingZone)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Landing Zone",
			"Could not create landing zone, unexpected error: "+err.Error(),
		)
		return
	}

	// Keep the tags the user declared rather than the superset the API returns (every schema property
	// plus injected restricted-tag defaults), which would break plan/apply consistency.
	createdLandingZone.Metadata.Tags = landingZone.Metadata.Tags

	resp.Diagnostics.Append(resp.State.Set(ctx, landingZoneModelFrom(createdLandingZone))...)
}

func (r *landingZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	landingZone, err := r.meshLandingZoneClient.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read landing zone '%s'", name),
			err.Error(),
		)
		return
	}

	if landingZone == nil {
		// The landing zone was deleted outside of Terraform, so we remove it from the state
		resp.State.RemoveResource(ctx)
		return
	}

	// Keep only the tags we already track. The API returns a superset (every schema property plus
	// injected restricted-tag defaults) that the caller may be unable to manage, so mirroring it
	// verbatim would surface as drift. On import there is no prior state (tags is null); we keep the
	// full set so a normal import round-trips, accepting that a restricted default would then show up.
	landingZone.Metadata.Tags = reconcileTrackedTags(ctx, req.State, path.Root("metadata").AtName("tags"), landingZone.Metadata.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, landingZoneModelFrom(landingZone))...)
}

func (r *landingZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	landingZone := client.MeshLandingZoneCreate{
		Metadata: client.MeshLandingZoneMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &landingZone.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &landingZone.Metadata)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updatedLandingZone, err := r.meshLandingZoneClient.Update(ctx, landingZone.Metadata.Name, &landingZone)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Landing Zone",
			"Could not update landing zone, unexpected error: "+err.Error(),
		)
		return
	}

	// Keep the tags the user declared rather than the superset the API returns, mirroring Create.
	updatedLandingZone.Metadata.Tags = landingZone.Metadata.Tags

	resp.Diagnostics.Append(resp.State.Set(ctx, landingZoneModelFrom(updatedLandingZone))...)
}

func (r *landingZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.meshLandingZoneClient.Delete(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete landing zone '%s'", name),
			err.Error(),
		)
		return
	}
}

func (r *landingZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("name"), req, resp)
}

// landingZoneSchemaV0Once builds the v0 schema for the state upgrader: the current schema with
// metadata.tags as its old set(string) type. v1 corrected tags to list(string) (tag values are
// lists, not sets), so this exists only to read v0 state.
var landingZoneSchemaV0Once = sync.OnceValue(func() schema.Schema {
	var schemaResp resource.SchemaResponse
	(&landingZoneResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	s.Version = 0

	metadata, ok := s.Attributes["metadata"].(schema.SingleNestedAttribute)
	if !ok {
		panic("landing zone metadata attribute is not a SingleNestedAttribute")
	}
	metadata.Attributes = maps.Clone(metadata.Attributes)
	metadata.Attributes["tags"] = schema.MapAttribute{
		ElementType: types.SetType{ElemType: types.StringType},
		Optional:    true,
		Computed:    true,
		Default:     mapdefault.StaticValue(types.MapValueMust(types.SetType{ElemType: types.StringType}, map[string]attr.Value{})),
	}
	s.Attributes = maps.Clone(s.Attributes)
	s.Attributes["metadata"] = metadata
	return s
})

func (r *landingZoneResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	prior := landingZoneSchemaV0Once()
	return map[int64]resource.StateUpgrader{
		0: {PriorSchema: &prior, StateUpgrader: r.upgradeTagsSetToListV0},
	}
}

// upgradeTagsSetToListV0 migrates v0 (tags as set) state to v1 (list). The Go model holds tags as
// map[string][]string, so a read-then-write under the current schema does the conversion losslessly.
func (r *landingZoneResource) upgradeTagsSetToListV0(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var model landingZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
