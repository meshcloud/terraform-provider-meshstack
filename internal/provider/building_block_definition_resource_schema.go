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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
			MarkdownDescription: "State of the version. One of `DRAFT`, `RELEASED`.",
			Computed:            true,
		},
	}

	inputsAttribute := schema.MapNestedAttribute{
		MarkdownDescription: "Building block definition inputs. Map from input key to input configuration.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"display_name": schema.StringAttribute{
					MarkdownDescription: "Display name for the input shown to users.",
					Required:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: "Input type. One of `STRING`, `CODE`, `INTEGER`, `BOOLEAN`, `FILE`, `LIST`, `SINGLE_SELECT`, `MULTI_SELECT`.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("STRING", "CODE", "INTEGER", "BOOLEAN", "FILE", "LIST", "SINGLE_SELECT", "MULTI_SELECT"),
					},
				},
				"assignment_type": schema.StringAttribute{
					MarkdownDescription: "How the input value is assigned. One of `USER_INPUT`, `PLATFORM_OPERATOR_MANUAL_INPUT`, `BUILDING_BLOCK_OUTPUT`, `PLATFORM_TENANT_ID`, `WORKSPACE_IDENTIFIER`, `PROJECT_IDENTIFIER`, `FULL_PLATFORM_IDENTIFIER`, `TENANT_BUILDING_BLOCK_UUID`, `STATIC`, `USER_PERMISSIONS`.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("USER_INPUT", "PLATFORM_OPERATOR_MANUAL_INPUT", "BUILDING_BLOCK_OUTPUT", "PLATFORM_TENANT_ID", "WORKSPACE_IDENTIFIER", "PROJECT_IDENTIFIER", "FULL_PLATFORM_IDENTIFIER", "TENANT_BUILDING_BLOCK_UUID", "STATIC", "USER_PERMISSIONS"),
					},
				},
				"argument": schema.StringAttribute{
					MarkdownDescription: "Argument value for static or template assignment types.",
					Optional:            true,
				},
				"is_environment": schema.BoolAttribute{
					MarkdownDescription: "Whether this input is exposed as an environment variable.",
					Optional:            true,
					Computed:            true,
					Default:             booldefault.StaticBool(false),
				},
				"is_sensitive": schema.BoolAttribute{
					MarkdownDescription: "Whether this input contains sensitive data.",
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
					MarkdownDescription: "List of allowed values for SINGLE_SELECT or MULTI_SELECT types.",
					ElementType:         types.StringType,
					Optional:            true,
				},
				"default_value": schema.StringAttribute{
					MarkdownDescription: "Default value for the input (as string, will be converted based on type).",
					Optional:            true,
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "Description of the input parameter.",
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
		MarkdownDescription: "Building block definition outputs. Map from output key to output configuration.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"display_name": schema.StringAttribute{
					MarkdownDescription: "Display name for the output shown to users.",
					Required:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: "Output type. One of `STRING`, `CODE`, `INTEGER`, `BOOLEAN`, `FILE`, `LIST`, `SINGLE_SELECT`, `MULTI_SELECT`.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("STRING", "CODE", "INTEGER", "BOOLEAN", "FILE", "LIST", "SINGLE_SELECT", "MULTI_SELECT"),
					},
				},
				"assignment_type": schema.StringAttribute{
					MarkdownDescription: "How the output is assigned. One of `NONE`, `PLATFORM_TENANT_ID`, `SIGN_IN_URL`.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("NONE", "PLATFORM_TENANT_ID", "SIGN_IN_URL"),
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
					"uuid": schema.StringAttribute{
						MarkdownDescription: "Unique identifier of the building block definition (server-generated).",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "The workspace that owns this building block definition.",
						Required:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags associated with this building block definition.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Optional:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp when the building block definition was created.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"marked_for_deletion_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp when the building block definition was marked for deletion.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"marked_for_deletion_by": schema.StringAttribute{
						MarkdownDescription: "User who marked the building block definition for deletion.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
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
						MarkdownDescription: fmt.Sprintf("List of platform identifiers that this building block supports. Required and must be non-empty if target_type is `%s`", TenantTargetType),
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
						MarkdownDescription: fmt.Sprintf("Target type for building blocks using this definition. One of `%s`, `%s`.", TenantTargetType, WorkspaceTargetType),
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(WorkspaceTargetType),
						Validators: []validator.String{
							stringvalidator.OneOf(TenantTargetType, WorkspaceTargetType),
						},
					},
					"notification_subscriber_usernames": schema.ListAttribute{
						MarkdownDescription: "List of usernames to notify about events related to this building block.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},

			"draft": schema.BoolAttribute{
				MarkdownDescription: "Whether the current version is a draft. Set to false to release the version.",
				Required:            true,
			},
			"runner_ref": schema.StringAttribute{
				MarkdownDescription: "UUID of the building block runner to use.",
				Required:            true,
			},
			"only_apply_once_per_tenant": schema.BoolAttribute{
				MarkdownDescription: "Whether this building block can only be applied once per tenant.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"deletion_mode": schema.StringAttribute{
				MarkdownDescription: "Deletion behavior. One of `DELETE`, `PURGE`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("DELETE"),
				Validators: []validator.String{
					stringvalidator.OneOf("DELETE", "PURGE"),
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
				MarkdownDescription: "Implementation configuration for the building block. Must contain exactly one of `terraform` or `github_actions`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"terraform": schema.SingleNestedAttribute{
						MarkdownDescription: "Terraform implementation configuration.",
						Optional:            true,
						Validators: []validator.Object{
							objectvalidator.ExactlyOneOf(
								path.MatchRelative().AtParent().AtName("github_actions"),
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
							"ssh_private_key": schema.StringAttribute{
								MarkdownDescription: "SSH private key for accessing private repositories. This value is write-only and will not be stored in state.",
								Optional:            true,
								Sensitive:           true,
								WriteOnly:           true,
								Validators: []validator.String{
									stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("ssh_private_key_version")),
								},
							},
							"ssh_private_key_version": schema.StringAttribute{
								MarkdownDescription: "Version identifier for the SSH private key. Change this value to trigger rotation of the write-only `ssh_private_key` attribute. Required when `ssh_private_key` is set.",
								Optional:            true,
								Validators: []validator.String{
									stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("ssh_private_key")),
								},
							},
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
					"github_actions": schema.SingleNestedAttribute{
						MarkdownDescription: "GitHub Actions implementation configuration.",
						Optional:            true,
						Validators: []validator.Object{
							objectvalidator.ExactlyOneOf(
								path.MatchRelative().AtParent().AtName("terraform"),
							),
						},
						Attributes: map[string]schema.Attribute{
							"repository": schema.StringAttribute{
								MarkdownDescription: "GitHub repository in format `owner/repo`.",
								Optional:            true,
							},
							"branch": schema.StringAttribute{
								MarkdownDescription: "Branch to use for workflows.",
								Optional:            true,
							},
							"apply_workflow": schema.StringAttribute{
								MarkdownDescription: "Workflow file name for apply operations.",
								Optional:            true,
							},
							"destroy_workflow": schema.StringAttribute{
								MarkdownDescription: "Workflow file name for destroy operations.",
								Optional:            true,
							},
							"source_platform_full_identifier": schema.StringAttribute{
								MarkdownDescription: "Full platform identifier.",
								Optional:            true,
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
				MarkdownDescription: "Latest released version (excludes drafts).",
				Computed:            true,
				Attributes:          versionAttributes,
			},
			"versions": schema.ListNestedAttribute{
				MarkdownDescription: "List of all available versions of this building block definition.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: versionAttributes,
				},
			},
		},
	}
}
