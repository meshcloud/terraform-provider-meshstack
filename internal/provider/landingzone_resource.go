package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &landingZoneResource{}
	_ resource.ResourceWithConfigure   = &landingZoneResource{}
	_ resource.ResourceWithImportState = &landingZoneResource{}
)

// NewLandingZoneResource is a helper function to simplify the provider implementation.
func NewLandingZoneResource() resource.Resource {
	return &landingZoneResource{}
}

// landingZoneResource is the resource implementation.
type landingZoneResource struct {
	client *client.MeshStackProviderClient
}

// Metadata returns the resource type name.
func (r *landingZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_landingzone"
}

// Configure adds the provider configured client to the resource.
func (r *landingZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *landingZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack landing zone." +
			"\n\n~> **Note:** Managing landing zones requires an API key with sufficient admin permissions.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Landing zone datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v1-preview"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshLandingZone`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshLandingZone"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshLandingZone"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

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
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags of the landing zone.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Optional:            true,
						Computed:            true,
						Default:             mapdefault.StaticValue(types.MapValueMust(types.ListType{ElemType: types.StringType}, map[string]attr.Value{})),
					},
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
					"platform_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the platform this landing zone belongs to.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"uuid": schema.StringAttribute{
								PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
								MarkdownDescription: "UUID of the platform.",
								Required:            true,
							},
							"kind": schema.StringAttribute{
								MarkdownDescription: "Must always be set to meshPlatform",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("meshPlatform"),
								},
							},
						},
					},
					"platform_properties": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform-specific configuration options.",
						Required:            true,
						Sensitive:           false,
						Attributes: map[string]schema.Attribute{
							"aws":        awsPlatformConfigSchema(),
							"aks":        aksPlatformConfigSchema(),
							"azure":      azurePlatformConfigSchema(),
							"azurerg":    azureRgPlatformConfigSchema(),
							"gcp":        gcpPlatformConfigSchema(),
							"kubernetes": kubernetesPlatformConfigSchema(),
							"openshift":  openShiftPlatformConfigSchema(),
							"type": schema.StringAttribute{
								MarkdownDescription: "Type of the platform. Must be one of: `aws`, `aks`, `azure`, `azurerg`, `gcp`, `kubernetes`, `openshift`.",
								Required:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
						},
					},
				},
			},
		},
	}
}

func awsPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "AWS platform properties. Must be present if `type` is `aws`",
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
			"aws_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "Roles can be mapped from the meshRole to the AWS Role. The AWS role will be part of the role or group name within AWS. If empty, the default that is configured on platform level will be used.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"platform_role": schema.StringAttribute{
							MarkdownDescription: "The AWS platform role",
							Required:            true,
						},
						"policies": schema.ListAttribute{
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
		MarkdownDescription: "AKS platform properties. Must be present if `type` is `aks`",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"kubernetes_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "Roles need to be mapped from the meshRole to the Cluster Role. You can use " +
					"both built in roles like 'editor' or custom roles that you setup in the Kubernetes Cluster " +
					"before. For more information see [the Landing Zone documentation](https://docs.meshcloud.io/meshstack.kubernetes.landing-zones/).",
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"platform_roles": schema.ListAttribute{
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
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure platform properties. Must be present if `type` is `azure`",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"azure_management_group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Azure Management Group ID where projects will be created.",
			},
			"azure_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "An array of mappings between the meshRole and the Azure" +
					" specific access role. " +
					"For more information see [the Landing Zone documentation](https://docs.meshcloud.io/meshstack.azure.landing-zones#meshrole-to-platform-role-mapping). " +
					"If empty, the default that is configured on platform level will be used.",
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"azure_group_suffix": schema.StringAttribute{
							MarkdownDescription: "The given role name will be injected into the" +
								" group name via the group naming pattern configured on the" +
								" platform instance.",
							Required: true,
						},
						"azure_role_definitions": schema.ListNestedAttribute{
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
				},
			},
		},
	}
}

func gcpPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "GCP platform properties. Must be present if `type` is `gcp`",
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
			"gcp_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "You can use both built-in roles like 'roles/editor' or" +
					" custom roles like 'organizations/123123123123/roles/meshstack." +
					"project_developer'. For more information see " +
					"[the Landing Zone documentation](https://docs.meshcloud.io/meshstack.gcp.landing-zones/#meshrole-to-platform-role-mapping). Multiple GCP Roles can be assigned to one meshRole. If empty, " +
					"the default that is configured on platform level will be used.",
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"platform_roles": schema.ListAttribute{
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
		MarkdownDescription: "Azure Resource Group platform properties. Must be present if `type` is `azurerg`",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"azure_rg_location": schema.StringAttribute{
				MarkdownDescription: "The newly created Resource Group for the meshProjects will get assigned to this location. It must be all lower case and without spaces (e.g. `eastus2` for East US 2). In order to list the available locations you can use `az account list-locations --query \"[*].name\" --out tsv | sort`",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"azure_rg_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "An array of mappings between the meshRole and the Azure" +
					" specific access role. " +
					"For more information see [the Landing Zone documentation](https://docs.meshcloud.io/meshstack.azure.landing-zones#meshrole-to-platform-role-mapping). " +
					"If empty, the default that is configured on platform level will be used.",
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"azure_group_suffix": schema.StringAttribute{
							MarkdownDescription: "The given role name will be injected into the" +
								" group name via the group naming pattern configured on the" +
								" platform instance.",
							Required: true,
						},
						"azure_role_definition_ids": schema.ListAttribute{
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
		MarkdownDescription: "Kubernetes platform properties. Must be present if `type` is `kubernetes`",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"kubernetes_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "Kubernetes role mappings configuration.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"platform_roles": schema.ListAttribute{
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
		MarkdownDescription: "OpenShift platform properties. Must be present if `type` is `openshift`",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"openshift_template": schema.StringAttribute{
				MarkdownDescription: "OpenShift template to use for this landing zone.",
				Optional:            true,
			},
		},
	}
}

func meshProjectRoleAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "the meshProject role",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the meshProjectRole",
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshProjectRole`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshProjectRole"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshProjectRole"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *landingZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	landingZone := client.MeshLandingZoneCreate{
		Metadata: client.MeshLandingZoneCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &landingZone.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &landingZone.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &landingZone.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("tags"), &landingZone.Metadata.Tags)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createdLandingZone, err := r.client.CreateLandingZone(&landingZone)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Landing Zone",
			"Could not create landing zone, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, createdLandingZone)...)
}

func (r *landingZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	landingZone, err := r.client.ReadLandingZone(name)
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

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, landingZone)...)
}

func (r *landingZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	landingZone := client.MeshLandingZoneCreate{
		Metadata: client.MeshLandingZoneCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &landingZone.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &landingZone.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &landingZone.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("tags"), &landingZone.Metadata.Tags)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updatedLandingZone, err := r.client.UpdateLandingZone(landingZone.Metadata.Name, &landingZone)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Landing Zone",
			"Could not update landing zone, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, updatedLandingZone)...)
}

func (r *landingZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteLandingZone(name)
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
