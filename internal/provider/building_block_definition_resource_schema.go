package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
	"github.com/meshcloud/terraform-provider-meshstack/internal/validators"
)

func (r *buildingBlockDefinitionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	versionAttributes := map[string]schema.Attribute{
		"uuid": schema.StringAttribute{
			MarkdownDescription: "UUID of the version.",
			Computed:            true,
		},
		"number": schema.Int64Attribute{
			MarkdownDescription: "Version number.",
			Computed:            true,
		},
		"state": schema.StringAttribute{
			MarkdownDescription: fmt.Sprintf("State of the version. One of %s.", client.MeshBuildingBlockDefinitionVersionStates.Markdown()),
			Computed:            true,
		},
		"content_hash": schema.StringAttribute{
			MarkdownDescription: "Content hash of the version. Will only change for draft versions when edited, otherwise constant.",
			Computed:            true,
		},
	}

	inputsAttribute := schema.MapNestedAttribute{
		MarkdownDescription: "Map of input definitions for the building block. Keys are input names, values are input configuration objects. " +
			"Inputs define parameters that building blocks can receive.",
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"display_name": schema.StringAttribute{
					MarkdownDescription: "Human-readable display name for the input.",
					Required:            true,
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "Description explaining the purpose and usage of the input.",
					Optional:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: "Data type of the input. One of " + client.MeshBuildingBlockIOTypes.Markdown() + ".",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf(client.MeshBuildingBlockIOTypes.Strings()...),
					},
				},
				"assignment_type": schema.StringAttribute{
					MarkdownDescription: "How the input value is assigned. One of " + client.MeshBuildingBlockInputAssignmentTypes.Markdown() + ". " +
						"Determines which additional attributes are required or allowed.",
					Required: true,
					Validators: []validator.String{
						stringvalidator.OneOf(client.MeshBuildingBlockInputAssignmentTypes.Strings()...),
					},
				},
				"argument": schema.StringAttribute{
					CustomType: jsontypes.NormalizedType{},
					MarkdownDescription: "Argument value for the input, depending on the assignment type. " +
						"**Required** if `assignment_type` is " + enum.Of(
						client.MeshBuildingBlockInputAssignmentTypeStatic,
						client.MeshBuildingBlockInputAssignmentTypeBuildingBlockOutput,
					).Markdown() + ". " +
						"**Must not be provided** for other assignment types. " +
						"The value must be passed through `jsonencode()` to support dynamic typing as defined by the `type` attribute. " +
						"For " + client.MeshBuildingBlockInputAssignmentTypeBuildingBlockOutput.Markdown() + ", the value must have the format `jsonencode(\"<BuildingBlockDefinitionUuid>.<outputName>\")`.",
					Optional: true,
				},
				"default_value": schema.StringAttribute{
					CustomType: jsontypes.NormalizedType{},
					MarkdownDescription: "Default value for the input. " +
						"**Can only be provided** if `assignment_type` is " + enum.Of(
						client.MeshBuildingBlockInputAssignmentTypeUserInput,
						client.MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput,
					).Markdown() + ". " +
						"Must be passed through `jsonencode()` to match the `type` attribute.",
					Optional: true,
				},
				"sensitive": schema.SingleNestedAttribute{
					MarkdownDescription: "Configuration for sensitive input values. " +
						"**Mutually exclusive** with the non-sensitive `argument` and `default_value` attributes. " +
						"When an input is marked as sensitive, use the nested `sensitive.argument` or `sensitive.default_value` instead of the top-level attributes. " +
						"You can provide an empty attribute `sensitive = {}` to mark this input as sensitive without providing values. " +
						"Sensitive inputs are **only supported** for `assignment_type` of " + enum.Of(
						client.MeshBuildingBlockInputAssignmentTypeUserInput,
						client.MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput,
						client.MeshBuildingBlockInputAssignmentTypeStatic).Markdown() + ".",
					Optional: true,
					Attributes: map[string]schema.Attribute{
						"argument": secret.ResourceSchema(secret.SchemaOptions{
							MarkdownDescription: "Sensitive variant of the `argument` attribute. Contains encrypted secret data.",
							Optional:            true,
						}),
						"default_value": secret.ResourceSchema(secret.SchemaOptions{
							MarkdownDescription: "Sensitive variant of the `default_value` attribute. Contains encrypted secret data.",
							Optional:            true,
						}),
					},
				},
				"is_environment": schema.BoolAttribute{
					MarkdownDescription: "Whether this input is exposed as an environment variable (when `true`) or as a regular variable (when `false`).",
					Optional:            true,
					Computed:            true,
					Default:             booldefault.StaticBool(false),
				},
				"updateable_by_consumer": schema.BoolAttribute{
					MarkdownDescription: "Whether the input value can be updated by consumers without admin or platform operator permissions.",
					Optional:            true,
					Computed:            true,
					Default:             booldefault.StaticBool(false),
				},
				"selectable_values": schema.SetAttribute{
					MarkdownDescription: "List of allowed values for the input. **Required** when `type` is " + client.MeshBuildingBlockIOTypeSingleSelect.Markdown() + " or " + client.MeshBuildingBlockIOTypeMultiSelect.Markdown() + ".",
					ElementType:         types.StringType,
					Optional:            true,
				},
				"value_validation_regex": schema.StringAttribute{
					MarkdownDescription: "Regular expression pattern to validate input values.",
					Optional:            true,
				},
				"validation_regex_error_message": schema.StringAttribute{
					MarkdownDescription: "Error message to display when regex validation fails.",
					Optional:            true,
				},
			},
		},
	}

	outputsAttribute := schema.MapNestedAttribute{
		MarkdownDescription: "Map of output definitions for the building block. Keys are output names, values are output configuration objects. " +
			"Outputs define values that building blocks produce and can be consumed by other building blocks.",
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"display_name": schema.StringAttribute{
					MarkdownDescription: "Human-readable display name for the output.",
					Required:            true,
				},
				"assignment_type": schema.StringAttribute{
					MarkdownDescription: "How the output is used. One of " + client.MeshBuildingBlockDefinitionOutputAssignmentTypes.Markdown() + ".",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf(client.MeshBuildingBlockDefinitionOutputAssignmentTypes.Strings()...),
					},
				},
				"type": schema.StringAttribute{
					MarkdownDescription: "Data type of the output. One of " + client.MeshBuildingBlockIOTypes.Markdown() + ".",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf(client.MeshBuildingBlockIOTypes.Strings()...),
					},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a meshBuildingBlockDefinition in meshStack. " +
			"Building Block Definitions define reusable automation components that can be executed on workspaces or tenants. " +
			"This resource combines the building block definition metadata with version information in a single resource for simplified management.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata of the building block definition. Contains identifiers and ownership details.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID to uniquely identify the building block definition.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace that owns this building block definition.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Key/value pairs of tags set on the building block definition. Values are arrays of strings.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Optional:            true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Specification of the building block definition. Contains configuration settings for the building block.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the building block definition as shown in meshPanel.",
						Required:            true,
					},
					"symbol": schema.StringAttribute{
						MarkdownDescription: "Symbol/icon of the building block definition as shown in meshPanel.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description of the building block definition as shown in meshPanel.",
						Required:            true,
					},
					"readme": schema.StringAttribute{
						MarkdownDescription: "Detailed readme/documentation in markdown format.",
						Optional:            true,
					},
					"support_url": schema.StringAttribute{
						MarkdownDescription: "URL pointing to support resources for the building block definition.",
						Optional:            true,
					},
					"documentation_url": schema.StringAttribute{
						MarkdownDescription: "URL pointing to documentation for the building block definition.",
						Optional:            true,
					},
					"supported_platforms": schema.SetNestedAttribute{
						MarkdownDescription: fmt.Sprintf("List of platform identifiers that this building block supports. Required and must be non-empty if target_type is `%s`", client.MeshBuildingBlockTypeTenantLevel),
						Optional:            true,
						Validators: []validator.Set{
							validators.SupportedPlatforms{},
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"kind": schema.StringAttribute{
									MarkdownDescription: "Kind of the platform ref. Always `meshPlatformType` for now.",
									Optional:            true,
									Computed:            true,
									Default:             stringdefault.StaticString("meshPlatformType"),
									Validators: []validator.String{
										stringvalidator.OneOf(`meshPlatformType`),
									},
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "Name for `meshPlatformType` kind.",
									Optional:            true,
								},
							},
						},
					},
					"run_transparency": schema.BoolAttribute{
						MarkdownDescription: "Specifies the building block run control. When set to `true`, both platform teams and workspace users can view detailed run logs and re-run building blocks. When set to `false` (default), only platform teams have this access.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"use_in_landing_zones_only": schema.BoolAttribute{
						MarkdownDescription: "Whether this building block can only be used in landing zones.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"target_type": schema.StringAttribute{
						MarkdownDescription: fmt.Sprintf("Type of building block definition. Determines where building blocks can be attached. One of %s.", client.MeshBuildingBlockTypes.Markdown()),
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(client.MeshBuildingBlockTypeWorkspaceLevel.String()),
						Validators: []validator.String{
							stringvalidator.OneOf(client.MeshBuildingBlockTypes.Strings()...),
						},
					},
					"notification_subscribers": schema.ListAttribute{
						MarkdownDescription: "List of subscribers to notify about events related to this building block. Prefix usernames with `user:` and emails with `email:`.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},

			"version_spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Version specification for the building block definition.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"draft": schema.BoolAttribute{
						MarkdownDescription: "Whether the current version is a draft. Set to false to release the version.",
						Required:            true,
					},
					"state": schema.StringAttribute{
						MarkdownDescription: "State of the current version. One of " + client.MeshBuildingBlockDefinitionVersionStates.Markdown() + ".",
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf(client.MeshBuildingBlockDefinitionVersionStates.Strings()...),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"version_number": schema.Int64Attribute{
						MarkdownDescription: "The current version number, see also dedicated `version_latest` and `version_latest_release` outputs.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"runner_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the runner to run the implementation. " +
							"If omitted, the pre-defined shared runner is used suitable for the given `implementation` choice",
						Optional:   true,
						Computed:   true,
						Attributes: meshUuidRefAttribute("meshBuildingBlockRunner"),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"only_apply_once_per_tenant": schema.BoolAttribute{
						MarkdownDescription: "Whether this building block can only be applied once per tenant.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
					"deletion_mode": schema.StringAttribute{
						MarkdownDescription: fmt.Sprintf("Deletion behavior. One of %s.", client.BuildingBlockDeletionModes.Markdown()),
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(client.BuildingBlockDeletionModeDelete.String()),
						Validators: []validator.String{
							stringvalidator.OneOf(client.BuildingBlockDeletionModes.Strings()...),
						},
					},
					"permissions": schema.SetAttribute{
						MarkdownDescription: "Set of API permissions required by this building block. " +
							"Will provide building block runs with an ephemeral API token with the specified workspace permissions. " +
							"See [Workspace Permissions](https://docs.meshcloud.io/api/authentication/api-permissions/) for available values and " +
							"[documentation on ephemeral API keys](https://docs.dev.meshcloud.io/concepts/building-block/#ephemeral-api-keys).",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.Set{
							setvalidator.ValueStringsAre(
								stringvalidator.OneOf(client.WorkspacePermissions.Strings()...),
							),
						},
					},
					"dependency_refs": schema.ListNestedAttribute{
						MarkdownDescription: "List of refs to building block definitions this definition depends on.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: meshUuidRefAttribute("meshBuildingBlockDefinition"),
						},
					},
					"inputs":  inputsAttribute,
					"outputs": outputsAttribute,

					"implementation": schema.SingleNestedAttribute{
						MarkdownDescription: "Implementation configuration for the building block. Must contain exactly one of `manual`, `terraform`, `github_workflows`, `gitlab_pipeline`, or `azure_devops_pipeline`.",
						Required:            true,
						Validators: []validator.Object{
							validators.ExactlyOneAttributeValidator{},
						},
						Attributes: map[string]schema.Attribute{
							"manual": schema.SingleNestedAttribute{
								MarkdownDescription: "Manual implementation (no automation).",
								Optional:            true,
								Attributes:          map[string]schema.Attribute{},
							},
							"terraform": schema.SingleNestedAttribute{
								MarkdownDescription: "Terraform implementation configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"terraform_version": schema.StringAttribute{
										MarkdownDescription: "Terraform version to use (e.g., `1.9.0`).",
										Required:            true,
									},
									"repository_url": schema.StringAttribute{
										MarkdownDescription: "Git repository URL containing the Terraform code.",
										Required:            true,
									},
									"async": schema.BoolAttribute{
										MarkdownDescription: "Whether to run Terraform asynchronously.",
										Optional:            true,
										Computed:            true,
										Default:             booldefault.StaticBool(false),
									},
									"repository_path": schema.StringAttribute{
										MarkdownDescription: "Path within the repository to the Terraform module.",
										Optional:            true,
									},
									"ref_name": schema.StringAttribute{
										MarkdownDescription: "Git reference (branch, tag, or commit) to use.",
										Optional:            true,
									},
									"ssh_private_key": secret.ResourceSchema(secret.SchemaOptions{
										MarkdownDescription: "SSH private key for accessing private repositories.",
										Optional:            true,
									}),
									"use_mesh_http_backend_fallback": schema.BoolAttribute{
										MarkdownDescription: "Whether to use meshStack's HTTP backend as fallback.",
										Optional:            true,
										Computed:            true,
										Default:             booldefault.StaticBool(false),
									},
									"ssh_known_host": schema.SingleNestedAttribute{
										MarkdownDescription: "SSH known host configuration.",
										Optional:            true,
										Attributes: map[string]schema.Attribute{
											"host": schema.StringAttribute{
												MarkdownDescription: "Hostname (e.g., `github.com`).",
												Required:            true,
											},
											"key_type": schema.StringAttribute{
												MarkdownDescription: "SSH key type (e.g., `ssh-rsa`).",
												Required:            true,
											},
											"key_value": schema.StringAttribute{
												MarkdownDescription: "SSH key value.",
												Required:            true,
											},
										},
									},
								},
							},
							"github_workflows": schema.SingleNestedAttribute{
								MarkdownDescription: "GitHub Workflows implementation configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"repository": schema.StringAttribute{
										MarkdownDescription: "GitHub repository in format `owner/repo`.",
										Required:            true,
									},
									"branch": schema.StringAttribute{
										MarkdownDescription: "Branch to use for workflows.",
										Required:            true,
									},
									"apply_workflow": schema.StringAttribute{
										MarkdownDescription: "Workflow file name for apply operations.",
										Required:            true,
									},
									"destroy_workflow": schema.StringAttribute{
										MarkdownDescription: "Workflow file name for destroy operations.",
										Optional:            true,
									},
									"async": schema.BoolAttribute{
										MarkdownDescription: "Whether to run workflows asynchronously.",
										Optional:            true,
										Computed:            true,
										Default:             booldefault.StaticBool(false),
									},
									"omit_run_object_input": schema.BoolAttribute{
										MarkdownDescription: "Whether to omit run object input.",
										Optional:            true,
										Computed:            true,
										Default:             booldefault.StaticBool(false),
									},
									"integration_ref": schema.SingleNestedAttribute{
										MarkdownDescription: "Reference to the integration to use.",
										Required:            true,
										Attributes:          meshUuidRefAttribute("meshIntegration"),
									},
								},
							},
							"gitlab_pipeline": schema.SingleNestedAttribute{
								MarkdownDescription: "GitLab Pipeline implementation configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"project_id": schema.StringAttribute{
										MarkdownDescription: "GitLab project ID.",
										Required:            true,
									},
									"ref_name": schema.StringAttribute{
										MarkdownDescription: "Git reference (branch, tag) to use.",
										Required:            true,
									},
									"pipeline_trigger_token": secret.ResourceSchema(secret.SchemaOptions{
										MarkdownDescription: "GitLab pipeline trigger token.",
									}),
									"integration_ref": schema.SingleNestedAttribute{
										MarkdownDescription: "Reference to the integration to use.",
										Required:            true,
										Attributes:          meshUuidRefAttribute("meshIntegration"),
									},
								},
							},
							"azure_devops_pipeline": schema.SingleNestedAttribute{
								MarkdownDescription: "Azure DevOps Pipeline implementation configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"project": schema.StringAttribute{
										MarkdownDescription: "Azure DevOps project name.",
										Required:            true,
									},
									"pipeline_id": schema.StringAttribute{
										MarkdownDescription: "Azure DevOps pipeline ID.",
										Required:            true,
									},
									"async": schema.BoolAttribute{
										MarkdownDescription: "Whether to run pipeline asynchronously.",
										Optional:            true,
										Computed:            true,
										Default:             booldefault.StaticBool(false),
									},
									"integration_ref": schema.SingleNestedAttribute{
										MarkdownDescription: "Reference to the integration to use",
										Required:            true,
										Attributes:          meshUuidRefAttribute("meshIntegration"),
									},
								},
							},
						},
					},
				},
			},

			"version_latest": schema.SingleNestedAttribute{
				MarkdownDescription: "Latest version (including drafts).",
				Computed:            true,
				Attributes:          versionAttributes,
			},
			"version_latest_release": schema.SingleNestedAttribute{
				MarkdownDescription: "Latest released version (excludes drafts) and is null if BBD is initially created in draft mode.",
				Computed:            true,
				Optional:            true,
				Attributes:          versionAttributes,
			},
			"versions": schema.ListNestedAttribute{
				MarkdownDescription: "List of all available versions of this building block definition. Never empty.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: versionAttributes,
				},
			},

			"ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Reference to this building block definition, can be used as dependency ref in other building block definitions.",
				Computed:            true,
				Attributes:          meshUuidRefOutputAttribute("meshBuildingBlockDefinition"),
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}
