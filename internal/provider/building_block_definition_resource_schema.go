package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
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
		MarkdownDescription: "Building block definition inputs. Map from input name to input configuration.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"display_name": schema.StringAttribute{
					MarkdownDescription: "Display name for the input shown to users.",
					Required:            true,
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "Description of the input parameter.",
					Optional:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Input type. One of %s.", client.MeshBuildingBlockIOTypes.Markdown()),
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf(client.MeshBuildingBlockIOTypes.Strings()...),
					},
				},
				"assignment_type": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("How the input value is assigned. One of %s.", client.MeshBuildingBlockInputAssignmentTypes.Markdown()),
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf(client.MeshBuildingBlockInputAssignmentTypes.Strings()...),
					},
				},
				"argument": generic.AttributeSchema[any](
					generic.WithMarkdownDescription("Argument value for static assignment types. "+
						"Must be provided if `assignment_type` is "+client.MeshBuildingBlockInputAssignmentTypeStatic.Markdown()+", "+client.MeshBuildingBlockInputAssignmentTypeBuildingBlockOutput.Markdown()+". Otherwise it must not be provided. "+
						"The value must be passed through `jsonencode` to support dynamic typing as given by `type` attribute. "+
						"In case of "+client.MeshBuildingBlockInputAssignmentTypeBuildingBlockOutput.Markdown()+", must have the format `"+`jsonencode("<BuildingBlockDefinitionUuid>.<outputName>")`+"`."),
					generic.WithFlags(generic.AttributeOptional),
				),
				"default_value": generic.AttributeSchema[any](
					generic.WithMarkdownDescription("Default value for the input. Can be provided if `assignment_type` is "),
					generic.WithFlags(generic.AttributeOptional),
				),
				"sensitive": schema.SingleNestedAttribute{
					MarkdownDescription: "Sensitive input values, mutually exclusive with non-sensitive `argument` and `default_value` attributes. " +
						"You can provide an empty attribute `sensitive = {}` to mark this input sensitive without providing `argument` or `default_value`. " +
						"Sensitive is only supported for `argument_type` of " + client.MeshBuildingBlockInputAssignmentTypeUserInput.Markdown() + ", " + client.MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput.Markdown() + ", " + client.MeshBuildingBlockInputAssignmentTypeStatic.Markdown() + ".",
					Optional: true,
					Attributes: map[string]schema.Attribute{
						"argument": secret.AttributeSchema(secret.AttributeSchemaOptions{
							MarkdownDescription: "Sensitive variant of `argument` attribute. See there for further explanation.",
							Optional:            true,
						}),
						"default_value": secret.AttributeSchema(secret.AttributeSchemaOptions{
							MarkdownDescription: "Sensitive variant of `default_value` attribute. See there for further explanation.",
							Optional:            true,
						}),
					},
				},
				"is_environment": schema.BoolAttribute{
					MarkdownDescription: "Whether this input is exposed as an environment variable.",
					Optional:            true,
					Computed:            true,
					Default:             booldefault.StaticBool(false),
				},
				"updateable_by_consumer": schema.BoolAttribute{
					MarkdownDescription: "Whether consumers can update this input value.",
					Optional:            true,
					Computed:            true,
					Default:             booldefault.StaticBool(false),
				},
				"selectable_values": schema.ListAttribute{
					MarkdownDescription: "List of allowed values for " + client.MeshBuildingBlockIOTypeSingleSelect.Markdown() + " or " + client.MeshBuildingBlockIOTypeMultiSelect.Markdown() + " types.",
					ElementType:         types.StringType,
					Optional:            true,
				},
				"value_validation_regex": schema.StringAttribute{
					MarkdownDescription: "Regular expression to validate input values.",
					Optional:            true,
				},
				"validation_regex_error_message": schema.StringAttribute{
					MarkdownDescription: "Error message shown when validation regex fails.",
					Optional:            true,
				},
			},
		},
	}

	outputsAttribute := schema.MapNestedAttribute{
		MarkdownDescription: "Building block definition outputs. Map from output name to output configuration.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"display_name": schema.StringAttribute{
					MarkdownDescription: "Display name for the output shown to users.",
					Required:            true,
				},
				"assignment_type": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("How the output is assigned. One of %s.", client.MeshBuildingBlockDefinitionOutputAssignmentTypes.Markdown()),
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf(client.MeshBuildingBlockDefinitionOutputAssignmentTypes.Strings()...),
					},
				},
				"type": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Output type. One of %s.", client.MeshBuildingBlockIOTypes.Markdown()),
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf(client.MeshBuildingBlockIOTypes.Strings()...),
					},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack building block definition with version information merged into a single resource.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"uuid": generic.AttributeSchema[clientTypes.String](
						generic.WithMarkdownDescription("Unique identifier of the building block definition (server-generated)."),
						generic.WithFlags(generic.AttributeComputed),
					),
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "The workspace that owns this building block definition.",
						Required:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags associated with this building block definition.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Optional:            true,
					},
					"created_on": generic.AttributeSchema[clientTypes.String](
						generic.WithMarkdownDescription("Timestamp when the building block definition was created."),
						generic.WithFlags(generic.AttributeComputed),
					),
					"marked_for_deletion_on": generic.AttributeSchema[clientTypes.String](
						generic.WithMarkdownDescription("Timestamp when the building block definition was marked for deletion."),
						generic.WithFlags(generic.AttributeOptional|generic.AttributeComputed),
					),
					"marked_for_deletion_by": generic.AttributeSchema[clientTypes.String](
						generic.WithMarkdownDescription("User who marked the building block definition for deletion."),
						generic.WithFlags(generic.AttributeOptional|generic.AttributeComputed),
					),
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name for the building block definition.",
						Required:            true,
					},
					"symbol": schema.StringAttribute{
						MarkdownDescription: "Icon symbol for the building block definition.",
						Optional:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description of the building block definition.",
						Required:            true,
					},
					"readme": schema.StringAttribute{
						MarkdownDescription: "Detailed readme/documentation in markdown format.",
						Optional:            true,
					},
					"support_url": schema.StringAttribute{
						MarkdownDescription: "URL for support resources.",
						Optional:            true,
					},
					"documentation_url": schema.StringAttribute{
						MarkdownDescription: "URL for additional documentation.",
						Optional:            true,
					},
					"supported_platforms": schema.ListAttribute{
						MarkdownDescription: fmt.Sprintf("List of platform identifiers that this building block supports. Required and must be non-empty if target_type is `%s`", client.MeshBuildingBlockTypeTenantLevel),
						ElementType:         types.StringType,
						Optional:            true,
						Validators: []validator.List{
							supportedPlatformsValidator{},
						},
					},
					"run_transparency": schema.BoolAttribute{
						MarkdownDescription: "Whether to enable run transparency for this building block.",
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
						MarkdownDescription: fmt.Sprintf("Target type for building blocks using this definition. One of %s.", client.MeshBuildingBlockTypes.Markdown()),
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(client.MeshBuildingBlockTypeWorkspaceLevel.String()),
						Validators: []validator.String{
							stringvalidator.OneOf(client.MeshBuildingBlockTypes.Strings()...),
						},
					},
					"notification_subscriber_usernames": schema.ListAttribute{
						MarkdownDescription: "List of usernames to notify about events related to this building block.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},

			"version_spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Version specification for the building block definition.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"version_number": generic.AttributeSchema[clientTypes.Number](
						generic.WithMarkdownDescription("The current version number, see also dedicated `version_latest` and `version_latest_release` outputs."),
						generic.WithFlags(generic.AttributeComputed),
					),
					"state": generic.AttributeSchema[client.MeshBuildingBlockDefinitionVersionState](
						generic.WithMarkdownDescription("State of the current version."),
						generic.WithFlags(generic.AttributeComputed),
						generic.WithStringValidators(stringvalidator.OneOf(client.MeshBuildingBlockDefinitionVersionStates.Strings()...)),
					),
					"draft": schema.BoolAttribute{
						MarkdownDescription: "Whether the current version is a draft. Set to false to release the version.",
						Required:            true,
					},
					"runner_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the runner to run the implementation.",
						Required:            true,
						Attributes:          meshUuidRefAttribute("meshBuildingBlockRunner"),
					},
					"only_apply_once_per_tenant": schema.BoolAttribute{
						MarkdownDescription: "Whether this building block can only be applied once per tenant.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
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
					"dependency_refs": schema.ListAttribute{
						MarkdownDescription: "List of UUIDs of building block definitions this definition depends on.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"inputs":  inputsAttribute,
					"outputs": outputsAttribute,

					"implementation": schema.SingleNestedAttribute{
						MarkdownDescription: "Implementation configuration for the building block. Must contain exactly one of `manual`, `terraform`, `github_workflows`, `gitlab_pipeline`, or `azure_devops_pipeline`.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"manual": schema.SingleNestedAttribute{
								MarkdownDescription: "Manual implementation (no automation).",
								Optional:            true,
								Validators: []validator.Object{
									objectvalidator.ExactlyOneOf(
										path.MatchRelative().AtParent().AtName("terraform"),
										path.MatchRelative().AtParent().AtName("github_workflows"),
										path.MatchRelative().AtParent().AtName("gitlab_pipeline"),
										path.MatchRelative().AtParent().AtName("azure_devops_pipeline"),
									),
								},
								Attributes: map[string]schema.Attribute{},
							},
							"terraform": schema.SingleNestedAttribute{
								MarkdownDescription: "Terraform implementation configuration.",
								Optional:            true,
								Validators: []validator.Object{
									objectvalidator.ExactlyOneOf(
										path.MatchRelative().AtParent().AtName("manual"),
										path.MatchRelative().AtParent().AtName("github_workflows"),
										path.MatchRelative().AtParent().AtName("gitlab_pipeline"),
										path.MatchRelative().AtParent().AtName("azure_devops_pipeline"),
									),
								},
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
									"ssh_private_key": secret.AttributeSchema(secret.AttributeSchemaOptions{
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
								Validators: []validator.Object{
									objectvalidator.ExactlyOneOf(
										path.MatchRelative().AtParent().AtName("manual"),
										path.MatchRelative().AtParent().AtName("terraform"),
										path.MatchRelative().AtParent().AtName("gitlab_pipeline"),
										path.MatchRelative().AtParent().AtName("azure_devops_pipeline"),
									),
								},
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
								Validators: []validator.Object{
									objectvalidator.ExactlyOneOf(
										path.MatchRelative().AtParent().AtName("manual"),
										path.MatchRelative().AtParent().AtName("terraform"),
										path.MatchRelative().AtParent().AtName("github_workflows"),
										path.MatchRelative().AtParent().AtName("azure_devops_pipeline"),
									),
								},
								Attributes: map[string]schema.Attribute{
									"project_id": schema.StringAttribute{
										MarkdownDescription: "GitLab project ID.",
										Required:            true,
									},
									"ref_name": schema.StringAttribute{
										MarkdownDescription: "Git reference (branch, tag) to use.",
										Required:            true,
									},
									"pipeline_trigger_token": secret.AttributeSchema(secret.AttributeSchemaOptions{
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
								Validators: []validator.Object{
									objectvalidator.ExactlyOneOf(
										path.MatchRelative().AtParent().AtName("manual"),
										path.MatchRelative().AtParent().AtName("terraform"),
										path.MatchRelative().AtParent().AtName("github_workflows"),
										path.MatchRelative().AtParent().AtName("gitlab_pipeline"),
									),
								},
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
		},
	}
}
