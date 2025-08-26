package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &platformResource{}
	_ resource.ResourceWithConfigure   = &platformResource{}
	_ resource.ResourceWithImportState = &platformResource{}
)

// NewPlatformResource is a helper function to simplify the provider implementation.
func NewPlatformResource() resource.Resource {
	return &platformResource{}
}

// platformResource is the resource implementation.
type platformResource struct {
	client *client.MeshStackProviderClient
}

// Metadata returns the resource type name.
func (r *platformResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform"
}

// Configure adds the provider configured client to the resource.
func (r *platformResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *platformResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack platform.\n\n~> **Note:** Managing platforms requires an API key with sufficient admin permissions.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Platform datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v1"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshPlatform`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshPlatform"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Platform identifier.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[a-zA-Z0-9]+([._-][a-zA-Z0-9]+)*$`),
								"must be alphanumeric with dots, dashes or underscores",
							),
						},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Workspace identifier that owns this platform.",
						Required:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation date of the platform.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Deletion date of the platform.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the platform.",
						Required:            true,
					},
					"location_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the location where this platform is deployed.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"kind": schema.StringAttribute{
								MarkdownDescription: "Kind of the referenced object, always 'meshLocation'.",
								Computed:            true,
								Default:             stringdefault.StaticString("meshLocation"),
							},
							"identifier": schema.StringAttribute{
								MarkdownDescription: "Identifier of the location.",
								Required:            true,
							},
						},
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description of the platform.",
						Optional:            true,
					},
					"endpoint": schema.StringAttribute{
						MarkdownDescription: "Platform endpoint URL.",
						Required:            true,
					},
					"support_url": schema.StringAttribute{
						MarkdownDescription: "Support URL for the platform.",
						Optional:            true,
					},
					"documentation_url": schema.StringAttribute{
						MarkdownDescription: "Documentation URL for the platform.",
						Optional:            true,
					},
					"availability": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform availability configuration.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"restriction": schema.StringAttribute{
								MarkdownDescription: "Platform access restriction ('PUBLIC', 'PRIVATE', 'RESTRICTED').",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("PUBLIC", "PRIVATE", "RESTRICTED"),
								},
							},
							"restricted_to_workspaces": schema.ListAttribute{
								MarkdownDescription: "List of workspace identifiers that have access to this platform (only relevant for RESTRICTED platforms).",
								ElementType:         types.StringType,
								Optional:            true,
							},
							"marketplace_status": schema.StringAttribute{
								MarkdownDescription: "Platform marketplace publication status ('UNPUBLISHED', 'PUBLISHED', 'REQUESTED', 'REJECTED').",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("UNPUBLISHED", "PUBLISHED", "REQUESTED", "REJECTED"),
								},
							},
						},
					},
					"contributing_workspaces": schema.ListAttribute{
						MarkdownDescription: "List of workspace identifiers that contribute to this platform.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"config": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform-specific configuration options.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"aws":        awsPlatformConfigSchema(),
							"aks":        aksPlatformConfigSchema(),
							"azure":      azurePlatformConfigSchema(),
							"azurerg":    azureRGPlatformConfigSchema(),
							"gcp":        gcpPlatformConfigSchema(),
							"kubernetes": kubernetesPlatformConfigSchema(),
							"openshift":  openshiftPlatformConfigSchema(),
							"type": schema.StringAttribute{
								MarkdownDescription: "Type of the platform (e.g., 'aks', 'aws', 'azure', 'azurerg', 'gcp', 'kubernetes', 'openshift').",
								Required:            true,
							},
						},
					},
				},
			},
		},
	}
}

func (r *platformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	platform := client.PlatformCreate{
		Metadata: client.PlatformCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &platform.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &platform.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &platform.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &platform.Metadata.OwnedByWorkspace)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createdPlatform, err := r.client.CreatePlatform(&platform)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Platform",
			"Could not create platform, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, createdPlatform)...)
}

func (r *platformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var identifier string

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &identifier)...)

	if resp.Diagnostics.HasError() {
		return
	}

	platform, err := r.client.ReadPlatform(identifier)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read platform '%s'", identifier),
			err.Error(),
		)
		return
	}

	if platform == nil {
		// The platform was deleted outside of Terraform, so we remove it from the state
		resp.State.RemoveResource(ctx)
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, platform)...)
}

func (r *platformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	platform := client.PlatformCreate{
		Metadata: client.PlatformCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &platform.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &platform.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &platform.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &platform.Metadata.OwnedByWorkspace)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updatedPlatform, err := r.client.UpdatePlatform(platform.Metadata.Name, &platform)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Platform",
			"Could not update platform, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, updatedPlatform)...)
}

func (r *platformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var identifier string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &identifier)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePlatform(identifier)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete platform '%s'", identifier),
			err.Error(),
		)
		return
	}
}

func (r *platformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("name"), req, resp)
}

// Schema helper functions for platform configurations

func awsPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "AWS platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region",
				Required:            true,
			},
			"replication": schema.SingleNestedAttribute{
				MarkdownDescription: "AWS replication configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"access_config": schema.SingleNestedAttribute{
						MarkdownDescription: "meshStack currently supports 2 types of authentication. Workload Identity Federation (using OIDC) is the one that we recommend as it enables secure access to your AWS account without using long lived credentials. Alternatively, you can use credential based authentication by providing access and secret keys. Either the serviceUserConfig or workloadIdentityConfig must be provided.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"organization_root_account_role": schema.StringAttribute{
								MarkdownDescription: "ARN of the Management Account Role. The Management Account contains your AWS organization. E.g. arn:aws:iam::123456789:role/MeshfedServiceRole.",
								Required:            true,
							},
							"organization_root_account_external_id": schema.StringAttribute{
								MarkdownDescription: "External ID for organization root account role.",
								Optional:            true,
							},
							"service_user_config": schema.SingleNestedAttribute{
								MarkdownDescription: "Service user configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"access_key": schema.StringAttribute{
										MarkdownDescription: "AWS access key for service user",
										Required:            true,
									},
									"secret_key": schema.StringAttribute{
										MarkdownDescription: "AWS secret key for service user",
										Required:            true,
										Sensitive:           true,
									},
								},
							},
							"workload_identity_config": schema.SingleNestedAttribute{
								MarkdownDescription: "Workload identity configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"role_arn": schema.StringAttribute{
										MarkdownDescription: "IAM role ARN for workload identity.",
										Required:            true,
									},
								},
							},
						},
					},
					"wait_for_external_avm": schema.BoolAttribute{
						MarkdownDescription: "Flag to wait for external AVM.Please use this setting with care! It is currently very specific to certain tags being present on the account! In general, we recommend not to activate this waitForExternalAvm functionality! In a meshLandingZone an AVM can be triggered via an AWS StackSet or via a Lambda Function. If meshStack shall wait for the AVM to complete when creating a new platform tenant, this flag must be checked. meshStack will identify completion of the AVM by checking the presence of the following tags on the AWS account: 'ProductName' is set to workspace identifier and 'Stage' is set to project identifier.",
						Required:            true,
					},
					"automation_account_role": schema.StringAttribute{
						MarkdownDescription: "ARN of the Automation Account Role. The Automation Account contains all AWS StackSets and Lambda Functions that shall be executed via meshLandingZones. E.g. arn:aws:iam::123456789:role/MeshfedAutomationRole.",
						Required:            true,
					},
					"automation_account_external_id": schema.StringAttribute{
						MarkdownDescription: "External ID for automation account role.",
						Optional:            true,
					},
					"account_access_role": schema.StringAttribute{
						MarkdownDescription: "The name for the Account Access Role that will be rolled out to all managed accounts. Only a name, not an ARN must be set here, as the ARN must be built dynamically for every managed AWS Account. The replicator service user needs to assume this role in all accounts to manage them.",
						Required:            true,
					},
					"account_alias_pattern": schema.StringAttribute{
						MarkdownDescription: "Pattern for account aliases.",
						Required:            true,
					},
					"enforce_account_alias": schema.BoolAttribute{
						MarkdownDescription: "Flag to enforce account alias. If set, meshStack will guarantee on every replication that the configured Account Alias is applied. Otherwise it will only set the Account Alias once during tenant creation.",
						Required:            true,
					},
					"account_email_pattern": schema.StringAttribute{
						MarkdownDescription: "With a String Pattern you can define how the account email address of the created AWS account will be set. E.g. 'aws+#{workspaceIdentifier}.#{projectIdentifier}@yourcompany.com'. Please consider that this email address is limited to 64 characters! Also have a look at our docs for more information.",
						Required:            true,
					},
					"tenant_tags": meshTagConfigSchema(),
					"aws_sso": schema.SingleNestedAttribute{
						MarkdownDescription: "AWS SSO configuration.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"scim_endpoint": schema.StringAttribute{
								MarkdownDescription: "The SCIM endpoint you can find in your AWS IAM Identity Center Automatic provisioning config.",
								Required:            true,
							},
							"arn": schema.StringAttribute{
								MarkdownDescription: "The ARN of your AWS IAM Identity Center Instance. E.g. arn:aws:sso:::instance/ssoins-123456789abc.",
								Required:            true,
							},
							"group_name_pattern": schema.StringAttribute{
								MarkdownDescription: "Configures the pattern that defines the desired name of AWS IAM Identity Center groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix, which is configurable via Role Mappings in this platform config or via a meshLandingZone. Operators must ensure the group names will be unique within the same AWS IAM Identity Center Instance with that configuration. meshStack will additionally prefix the group name with 'mst-' to be able to identify the groups that are managed by meshStack.",
								Required:            true,
							},
							"sso_access_token": schema.StringAttribute{
								MarkdownDescription: "The AWS IAM Identity Center SCIM Access Token that was generated via the Automatic provisioning config in AWS IAM Identity Center.",
								Required:            true,
								Sensitive:           true,
							},
							"role_mappings": schema.MapAttribute{
								MarkdownDescription: "Role mappings.",
								ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
									"aws_role_name":       types.StringType,
									"permission_set_arns": types.ListType{ElemType: types.StringType},
								}},
								Required: true,
							},
							"sign_in_url": schema.StringAttribute{
								MarkdownDescription: " The AWS IAM Identity Center sign in Url, that must be used by end-users to log in via AWS IAM Identity Center to AWS Management Console.",
								Required:            true,
							},
						},
					},
					"enrollment_configuration": schema.SingleNestedAttribute{
						MarkdownDescription: "AWS enrollment configuration.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"management_account_id": schema.StringAttribute{
								MarkdownDescription: "The Account ID of the management account configured for the platform instance.",
								Required:            true,
							},
							"account_factory_product_id": schema.StringAttribute{
								MarkdownDescription: "Account factory product ID.",
								Required:            true,
							},
						},
					},
					"self_downgrade_access_role": schema.BoolAttribute{
						MarkdownDescription: "Flag for self downgrade access role. If set, meshStack will revoke its rights on the managed account that were only needed for initial account creation.",
						Required:            true,
					},
					"skip_user_group_permission_cleanup": schema.BoolAttribute{
						MarkdownDescription: "Skip user group permission cleanup.",
						Required:            true,
					},
				},
			},
		},
	}
}

func aksPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "AKS platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the AKS cluster",
				Required:            true,
			},
			"disable_ssl_validation": schema.BoolAttribute{
				MarkdownDescription: "Flag to disable SSL validation for the AKS cluster. (SSL Validation should at best never be disabled, but for integration of some private cloud platforms in an early state, they might not yet be using valid SSL certificates. In that case it can make sense to disable SSL validation here to already test integration of these platforms.)",
				Optional:            true,
			},
			"replication": schema.SingleNestedAttribute{
				MarkdownDescription: "AKS replication configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"access_token": schema.StringAttribute{
						MarkdownDescription: "The Access Token of the service account for replicator access.",
						Required:            true,
						Sensitive:           true,
					},
					"namespace_name_pattern": schema.StringAttribute{
						MarkdownDescription: "Pattern for naming namespaces in AKS",
						Required:            true,
					},
					"group_name_pattern": schema.StringAttribute{
						MarkdownDescription: "Pattern for naming groups in AKS",
						Required:            true,
					},
					"service_principal": schema.SingleNestedAttribute{
						MarkdownDescription: "Service principal configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"client_id": schema.StringAttribute{
								MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID'.",
								Required:            true,
							},
							"auth_type": schema.StringAttribute{
								MarkdownDescription: "Authentication type for the service principal (CREDENTIALS or WORKLOAD_IDENTITY)",
								Required:            true,
							},
							"credentials_auth_client_secret": schema.StringAttribute{
								MarkdownDescription: "Client secret for the service principal (if authType is CREDENTIALS)",
								Optional:            true,
								Sensitive:           true,
							},
							"entra_tenant": schema.StringAttribute{
								MarkdownDescription: "Domain name or ID of the Entra Tenant that holds the Service Principal.",
								Required:            true,
							},
							"object_id": schema.StringAttribute{
								MarkdownDescription: "he Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
								Required:            true,
							},
						},
					},
					"aks_subscription_id": schema.StringAttribute{
						MarkdownDescription: "AKS subscription ID.",
						Required:            true,
					},
					"aks_cluster_name": schema.StringAttribute{
						MarkdownDescription: "AKS cluster name.",
						Required:            true,
					},
					"aks_resource_group": schema.StringAttribute{
						MarkdownDescription: "AKS resource group.",
						Required:            true,
					},
					"redirect_url": schema.StringAttribute{
						MarkdownDescription: "This is the URL that Azure's consent experience redirects users to after they accept their invitation.",
						Optional:            true,
					},
					"send_azure_invitation_mail": schema.BoolAttribute{
						MarkdownDescription: "Flag to send Azure invitation emails. When true, meshStack instructs Azure to send out Invitation mails to invited users.",
						Required:            true,
					},
					"user_look_up_strategy": schema.StringAttribute{
						MarkdownDescription: "User lookup strategy",
						Required:            true,
					},
					"administrative_unit_id": schema.StringAttribute{
						MarkdownDescription: "Administrative unit ID.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func azurePlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"entra_tenant": schema.StringAttribute{
				MarkdownDescription: "Azure Active Directory (Entra ID) tenant",
				Required:            true,
			},
			"replication": azureReplicationConfigSchema(),
		},
	}
}

func azureRGPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Resource Group platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"entra_tenant": schema.StringAttribute{
				MarkdownDescription: "Azure Active Directory (Entra ID) tenant",
				Required:            true,
			},
			"replication": azureRGReplicationConfigSchema(),
		},
	}
}

func gcpPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "GCP platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"replication": schema.SingleNestedAttribute{
				MarkdownDescription: "GCP replication configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"service_account_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Service account configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"service_account_credentials_config": schema.SingleNestedAttribute{
								MarkdownDescription: "Service account credentials configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"service_account_credentials_b64": schema.StringAttribute{
										MarkdownDescription: "Base64 encoded credentials.json file for a GCP ServiceAccount. The replicator uses this Service Account to automate GCP API operations (IAM, ResourceManager etc.).",
										Required:            true,
										Sensitive:           true,
									},
								},
							},
							"service_account_workload_identity_config": schema.SingleNestedAttribute{
								MarkdownDescription: "Service account workload identity configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"audience": schema.StringAttribute{
										MarkdownDescription: "Workload identity audience.",
										Required:            true,
									},
									"service_account_email": schema.StringAttribute{
										MarkdownDescription: "Service account email.",
										Required:            true,
									},
								},
							},
						},
					},
					"domain": schema.StringAttribute{
						MarkdownDescription: "The domain used for cloud identity directory-groups created and managed by meshStack. meshStack maintains separate groups for each meshProject role on each managed GCP project.",
						Required:            true,
					},
					"customer_id": schema.StringAttribute{
						MarkdownDescription: "A Google Customer ID. It typically starts with a 'C'.",
						Required:            true,
					},
					"group_name_pattern": schema.StringAttribute{
						MarkdownDescription: "Pattern for group names.",
						Required:            true,
					},
					"project_name_pattern": schema.StringAttribute{
						MarkdownDescription: "Pattern for project names.",
						Required:            true,
					},
					"project_id_pattern": schema.StringAttribute{
						MarkdownDescription: "Pattern for project IDs.",
						Required:            true,
					},
					"billing_account_id": schema.StringAttribute{
						MarkdownDescription: "Billing account ID.",
						Required:            true,
					},
					"user_lookup_strategy": schema.StringAttribute{
						MarkdownDescription: "User lookup strategy.",
						Required:            true,
					},
					"used_external_id_type": schema.StringAttribute{
						MarkdownDescription: "Used external ID type.",
						Optional:            true,
					},
					"role_mappings": schema.MapAttribute{
						MarkdownDescription: "Role mappings.",
						ElementType:         types.StringType,
						Required:            true,
					},
					"allow_hierarchical_folder_assignment": schema.BoolAttribute{
						MarkdownDescription: "Allow hierarchical folder assignment.",
						Required:            true,
					},
					"tenant_tags": meshTagConfigSchema(),
					"skip_user_group_permission_cleanup": schema.BoolAttribute{
						MarkdownDescription: "Skip user group permission cleanup.",
						Required:            true,
					},
				},
			},
		},
	}
}

func kubernetesPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Kubernetes platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL for Kubernetes API.",
				Required:            true,
			},
			"disable_ssl_validation": schema.BoolAttribute{
				MarkdownDescription: "Disable SSL validation.",
				Optional:            true,
			},
			"replication": schema.SingleNestedAttribute{
				MarkdownDescription: "Kubernetes replication configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"client_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Kubernetes client configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"access_token": schema.StringAttribute{
								MarkdownDescription: "Access token for Kubernetes.",
								Required:            true,
								Sensitive:           true,
							},
						},
					},
					"namespace_name_pattern": schema.StringAttribute{
						MarkdownDescription: "Pattern for namespace names.",
						Required:            true,
					},
				},
			},
		},
	}
}

func openshiftPlatformConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "OpenShift platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL for OpenShift API.",
				Required:            true,
			},
			"disable_ssl_validation": schema.BoolAttribute{
				MarkdownDescription: "Disable SSL validation.",
				Optional:            true,
			},
			"replication": schema.SingleNestedAttribute{
				MarkdownDescription: "OpenShift replication configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"client_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Client configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"access_token": schema.StringAttribute{
								MarkdownDescription: "Access token for OpenShift.",
								Required:            true,
								Sensitive:           true,
							},
						},
					},
					"web_console_url": schema.StringAttribute{
						MarkdownDescription: "Web console URL.",
						Optional:            true,
					},
					"project_name_pattern": schema.StringAttribute{
						MarkdownDescription: "Pattern for project names.",
						Required:            true,
					},
					"enable_template_instantiation": schema.BoolAttribute{
						MarkdownDescription: "Enable template instantiation.",
						Required:            true,
					},
					"role_mappings": schema.MapAttribute{
						MarkdownDescription: "Role mappings.",
						ElementType:         types.StringType,
						Required:            true,
					},
					"identity_provider_name": schema.StringAttribute{
						MarkdownDescription: "Identity provider name.",
						Required:            true,
					},
					"tenant_tags": meshTagConfigSchema(),
				},
			},
		},
	}
}

// Helper schemas for reusable components

func meshTagConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Mesh tag configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"namespace_prefix": schema.StringAttribute{
				MarkdownDescription: " This is the prefix for all labels created by meshStack. It helps to keep track of which labels are managed by meshStack. It is recommended to let this prefix end with a delimiter like an underscore.",
				Required:            true,
			},
			"tag_mappers": schema.ListAttribute{
				MarkdownDescription: "Tag mappers.",
				ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
					"key":           types.StringType,
					"value_pattern": types.StringType,
				}},
				Required: true,
			},
		},
	}
}

func azureReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure replication configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"service_principal": azureServicePrincipalSchema(),
			"provisioning": schema.SingleNestedAttribute{
				MarkdownDescription: "To provide Azure Subscription for your organization's meshProjects, meshcloud supports using Enterprise Enrollment or allocating from a pool of pre-provisioned subscriptions. One of the subFields enterpriseEnrollment, customerAgreement or preProvisioned must be provided!",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"subscription_owner_object_ids": schema.ListAttribute{
						MarkdownDescription: "Subscription owner object IDs.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"enterprise_enrollment": schema.SingleNestedAttribute{
						MarkdownDescription: "Enterprise enrollment configuration.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"enrollment_account_id": schema.StringAttribute{
								MarkdownDescription: "Enrollment account ID.",
								Required:            true,
							},
							"subscription_offer_type": schema.StringAttribute{
								MarkdownDescription: "Subscription offer type.",
								Required:            true,
							},
							"use_legacy_subscription_enrollment": schema.BoolAttribute{
								MarkdownDescription: "Use legacy subscription enrollment.",
								Required:            true,
							},
							"subscription_creation_error_cooldown_sec": schema.Int64Attribute{
								MarkdownDescription: "Subscription creation error cooldown in seconds.",
								Optional:            true,
							},
						},
					},
					"customer_agreement": schema.SingleNestedAttribute{
						MarkdownDescription: "Customer agreement configuration.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"source_service_principal": azureGraphApiCredentialsSchema(),
							"destination_entra_id": schema.StringAttribute{
								MarkdownDescription: "Destination Entra ID.",
								Required:            true,
							},
							"source_entra_tenant": schema.StringAttribute{
								MarkdownDescription: "Source Entra tenant.",
								Required:            true,
							},
							"billing_scope": schema.StringAttribute{
								MarkdownDescription: "Billing scope.",
								Required:            true,
							},
							"subscription_creation_error_cooldown_sec": schema.Int64Attribute{
								MarkdownDescription: "Subscription creation error cooldown in seconds.",
								Optional:            true,
							},
						},
					},
					"pre_provisioned": schema.SingleNestedAttribute{
						MarkdownDescription: "Pre-provisioned subscription configuration.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"unused_subscription_name_prefix": schema.StringAttribute{
								MarkdownDescription: "Unused subscription name prefix.",
								Required:            true,
							},
						},
					},
				},
			},
			"b2b_user_invitation": schema.SingleNestedAttribute{
				MarkdownDescription: "B2B user invitation configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_url": schema.StringAttribute{
						MarkdownDescription: "This is the URL that Azure's consent experience redirects users to after they accept their invitation.",
						Required:            true,
					},
					"send_azure_invitation_mail": schema.BoolAttribute{
						MarkdownDescription: "When true, meshStack instructs Azure to send out Invitation mails to invited users. These mails allow users to redeem their invitation to the AAD tenant only using email and Azure Portal.",
						Required:            true,
					},
				},
			},
			"subscription_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of Azure Subscriptions managed by meshStack.",
				Required:            true,
			},
			"group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of AAD groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix, which is configurable via Role Mappings in this platform config or via a meshLandingZone. Operators must ensure the group names are unique in the managed AAD Tenant.",
				Required:            true,
			},
			"blueprint_service_principal": schema.StringAttribute{
				MarkdownDescription: "Blueprint service principal.",
				Required:            true,
			},
			"blueprint_location": schema.StringAttribute{
				MarkdownDescription: "Blueprint location.",
				Required:            true,
			},
			"role_mappings": schema.MapAttribute{
				MarkdownDescription: "Role mappings.",
				ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
					"alias": types.StringType,
					"id":    types.StringType,
				}},
				Required: true,
			},
			"tenant_tags": meshTagConfigSchema(),
			"user_look_up_strategy": schema.StringAttribute{
				MarkdownDescription: "User lookup strategy ('userPrincipalName' or 'email'). Users can either be looked up in cloud platforms by email or UPN (User Principal Name). In most cases email is the matching way as it is the only identifier that is consistently used throughout all cloud platforms and meshStack.",
				Required:            true,
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "Flag to skip user group permission cleanup. For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the Azure platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Required:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "Administrative unit ID.",
				Optional:            true,
			},
		},
	}
}

func azureRGReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Resource Group replication configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"service_principal": azureServicePrincipalSchema(),
			"subscription": schema.StringAttribute{
				MarkdownDescription: "The Subscription that will contain all the created Resource Groups. Once you set the Subscription, you must not change it.",
				Required:            true,
			},
			"resource_group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name Resource Group managed by meshStack. It follows the usual replicator string pattern features. Operators must ensure the group names are unique within the Subscription.",
				Required:            true,
			},
			"user_group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of AAD groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix. This suffix is configurable via Role Mappings in this platform config.",
				Required:            true,
			},
			"b2b_user_invitation": schema.SingleNestedAttribute{
				MarkdownDescription: "B2B user invitation configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_url": schema.StringAttribute{
						MarkdownDescription: "This is the URL that Azure's consent experience redirects users to after they accept their invitation.",
						Required:            true,
					},
					"send_azure_invitation_mail": schema.BoolAttribute{
						MarkdownDescription: "When true, meshStack instructs Azure to send out Invitation mails to invited users. These mails allow users to redeem their invitation to the AAD tenant only using email and Azure Portal.",
						Required:            true,
					},
				},
			},
			"user_look_up_strategy": schema.StringAttribute{
				MarkdownDescription: "User lookup strategy ('userPrincipalName' or 'email'). Users can either be looked up in cloud platforms by email or UPN (User Principal Name). In most cases email is the matching way as it is the only identifier that is consistently used throughout all cloud platforms and meshStack.",
				Required:            true,
			},
			"tenant_tags": meshTagConfigSchema(),
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "Skip user group permission cleanup.",
				Required:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "Administrative unit ID.",
				Optional:            true,
			},
		},
	}
}

func azureServicePrincipalSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure service principal configuration.",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID",
				Required:            true,
			},
			"auth_type": schema.StringAttribute{
				MarkdownDescription: "Authentication type (CREDENTIALS or WORKLOAD_IDENTITY)",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("CREDENTIALS", "WORKLOAD_IDENTITY"),
				},
			},
			"credentials_auth_client_secret": schema.StringAttribute{
				MarkdownDescription: "Client secret (if authType is CREDENTIALS)",
				Optional:            true,
				Sensitive:           true,
			},
			"object_id": schema.StringAttribute{
				MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
				Required:            true,
			},
		},
	}
}

func azureGraphApiCredentialsSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Graph API credentials.",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				MarkdownDescription: "Client ID.",
				Required:            true,
			},
			"auth_type": schema.StringAttribute{
				MarkdownDescription: "Authentication type (CREDENTIALS or WORKLOAD_IDENTITY).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("CREDENTIALS", "WORKLOAD_IDENTITY"),
				},
			},
			"credentials_auth_client_secret": schema.StringAttribute{
				MarkdownDescription: "Client secret for credentials authentication.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}
