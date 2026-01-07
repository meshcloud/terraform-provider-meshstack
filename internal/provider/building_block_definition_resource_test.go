package provider

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

func TestAccBuildingBlockDefinition(t *testing.T) {
	const resourceAddress = "meshstack_building_block_definition.example"
	const workspaceIdentifier = "my-workspace"

	const (
		// Note: whenever you need to change this value to fix the test, carefully review why that must be done!
		version1ContentHash = "v1:11ec8cd6ece960391afef439dab121641964a81489af9d63cd5329e0861c0c93"
	)

	var (
		versionStateDraft    = client.MeshBuildingBlockDefinitionVersionStateDraft.String()
		versionStateReleased = client.MeshBuildingBlockDefinitionVersionStateReleased.String()
	)

	// Define expected version objects
	version1 := func(state string) knownvalue.Check {
		return knownvalue.MapExact(map[string]knownvalue.Check{
			"uuid":         knownvalue.StringExact("dummy-version-uuid-1"),
			"number":       knownvalue.Int64Exact(1),
			"state":        knownvalue.StringExact(state),
			"content_hash": knownvalue.StringExact(version1ContentHash),
		})
	}
	version2 := knownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":         knownvalue.StringExact("dummy-version-uuid-2"),
		"number":       knownvalue.Int64Exact(2),
		"state":        knownvalue.StringExact(versionStateDraft),
		"content_hash": knownvalue.StringExact(version1ContentHash),
	})

	mock := newMockBuildingBlockDefinitionClients()

	config := examples.Resource{Name: "building_block_definition"}.String()
	configSpecChange := strings.ReplaceAll(config, "An example building block definition", "An updated building block definition")
	configDraftFalse := strings.ReplaceAll(configSpecChange, "draft = true", "draft = false")
	configDraftTrueAgain := strings.ReplaceAll(configSpecChange, "draft = false", "draft = true")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(withMockClient(client.Client{
			BuildingBlockDefinition:        &mock.bbd,
			BuildingBlockDefinitionVersion: &mock.bbdVersion,
		})),
		IsUnitTest: true,
		Steps: []resource.TestStep{
			// Step 1: Create resource and validate state thoroughly!
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// Metadata checks
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), knownvalue.MapExact(map[string]knownvalue.Check{
						"uuid":                   knownvalue.StringExact("dummy-uuid-12345"),
						"owned_by_workspace":     knownvalue.StringExact(workspaceIdentifier),
						"created_on":             knownValueNotEmptyString(),
						"marked_for_deletion_on": knownvalue.Null(),
						"marked_for_deletion_by": knownvalue.Null(),
						"tags": knownvalue.MapExact(map[string]knownvalue.Check{
							"environment": knownvalue.ListExact([]knownvalue.Check{
								knownvalue.StringExact("production"),
								knownvalue.StringExact("staging"),
							}),
							"team": knownvalue.ListExact([]knownvalue.Check{
								knownvalue.StringExact("platform-team"),
							}),
							"cost-center": knownvalue.ListExact([]knownvalue.Check{
								knownvalue.StringExact("cc-123"),
							}),
						}),
					})),

					// Spec checks
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), knownvalue.MapExact(map[string]knownvalue.Check{
						"display_name":      knownvalue.StringExact("Example Building Block"),
						"symbol":            knownvalue.StringExact("🏗️"),
						"description":       knownvalue.StringExact("An example building block definition"),
						"readme":            knownValueNotEmptyString(),
						"support_url":       knownvalue.StringExact("https://support.example.com/building-blocks"),
						"documentation_url": knownvalue.StringExact("https://docs.example.com/building-blocks"),
						"target_type":       knownvalue.StringExact("TENANT_LEVEL"),
						"supported_platforms": knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("azure.platform"),
							knownvalue.StringExact("aws.platform"),
						}),
						"run_transparency":          knownvalue.Bool(true),
						"use_in_landing_zones_only": knownvalue.Bool(true),
						"notification_subscriber_usernames": knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("admin@example.com"),
							knownvalue.StringExact("ops@example.com"),
						}),
					})),

					// version_spec checks
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("version_spec"), knownvalue.MapExact(map[string]knownvalue.Check{
						"state":                      knownvalue.StringExact("DRAFT"),
						"version_number":             knownvalue.Int64Exact(1),
						"draft":                      knownvalue.Bool(true),
						"only_apply_once_per_tenant": knownvalue.Bool(false),
						"deletion_mode":              knownvalue.StringExact("DELETE"),
						"runner_ref": knownvalue.MapExact(map[string]knownvalue.Check{
							"uuid": knownvalue.StringExact(""),
							"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
						}),
						"dependency_refs": knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("dep-1"),
							knownvalue.StringExact("dep-2"),
						}),
						"inputs": knownvalue.MapExact(map[string]knownvalue.Check{
							"environment": knownvalue.MapExact(map[string]knownvalue.Check{
								"display_name":           knownvalue.StringExact("Environment"),
								"type":                   knownvalue.StringExact("SINGLE_SELECT"),
								"assignment_type":        knownvalue.StringExact("USER_INPUT"),
								"is_environment":         knownvalue.Bool(false),
								"updateable_by_consumer": knownvalue.Bool(true),
								"description":            knownvalue.StringExact("The target environment"),
								"selectable_values": knownvalue.ListExact([]knownvalue.Check{
									knownvalue.StringExact("dev"),
									knownvalue.StringExact("staging"),
									knownvalue.StringExact("prod"),
								}),
								"value_validation_regex":         knownvalue.Null(),
								"validation_regex_error_message": knownvalue.Null(),
								"argument":                       knownvalue.Null(),
								"default_value":                  knownvalue.Null(),
								"sensitive":                      knownvalue.Null(),
							}),
							"resource_name": knownvalue.MapExact(map[string]knownvalue.Check{
								"display_name":                   knownvalue.StringExact("Resource Name"),
								"type":                           knownvalue.StringExact("BOOLEAN"),
								"assignment_type":                knownvalue.StringExact("STATIC"),
								"is_environment":                 knownvalue.Bool(false),
								"updateable_by_consumer":         knownvalue.Bool(true),
								"description":                    knownvalue.StringExact("Name of the resource to create"),
								"argument":                       knownvalue.StringExact("true"),
								"default_value":                  knownvalue.StringExact("true"),
								"value_validation_regex":         knownvalue.StringExact("^[a-z0-9-]+$"),
								"validation_regex_error_message": knownvalue.StringExact("Resource name must contain only lowercase letters, numbers, and hyphens"),
								"selectable_values":              knownvalue.Null(),
								"sensitive":                      knownvalue.Null(),
							}),
							"something_very_secret": knownvalue.MapExact(map[string]knownvalue.Check{
								"display_name":           knownvalue.StringExact(""),
								"type":                   knownvalue.StringExact("STRING"),
								"assignment_type":        knownvalue.StringExact("STATIC"),
								"is_environment":         knownvalue.Bool(false),
								"updateable_by_consumer": knownvalue.Bool(true),
								"description":            knownvalue.StringExact("Name of the resource to create"),
								"sensitive": knownvalue.MapExact(map[string]knownvalue.Check{
									"argument": knownvalue.MapExact(map[string]knownvalue.Check{
										"value":       knownvalue.Null(),
										"hash":        knownvalue.StringExact("sha256:write-only-plaintext-value-should-be-ephemeral"),
										"fingerprint": knownvalue.StringExact("sha256:write-only-plaintext-value-should-be-ephemeral"),
									}),
									"default_value": knownvalue.MapExact(map[string]knownvalue.Check{
										"value":       knownvalue.Null(),
										"hash":        knownvalue.StringExact("sha256:write-only-plaintext-value-should-be-ephemeral"),
										"fingerprint": knownvalue.StringExact("sha256:write-only-plaintext-value-should-be-ephemeral"),
									}),
								}),
								"value_validation_regex":         knownvalue.StringExact("^[a-z0-9-]+$"),
								"validation_regex_error_message": knownvalue.StringExact("Resource name must contain only lowercase letters, numbers, and hyphens"),
								"selectable_values":              knownvalue.Null(),
								"argument":                       knownvalue.Null(),
								"default_value":                  knownvalue.Null(),
							}),
						}),
						"implementation": knownvalue.MapExact(map[string]knownvalue.Check{
							"manual":                knownvalue.Null(),
							"github_workflows":      knownvalue.Null(),
							"gitlab_pipeline":       knownvalue.Null(),
							"azure_devops_pipeline": knownvalue.Null(),
							"terraform": knownvalue.MapExact(map[string]knownvalue.Check{
								"terraform_version":              knownvalue.StringExact("1.9.0"),
								"repository_url":                 knownvalue.StringExact("https://github.com/example/building-block.git"),
								"async":                          knownvalue.Bool(false),
								"repository_path":                knownvalue.StringExact("terraform/modules/example"),
								"ref_name":                       knownvalue.StringExact("v1.0.0"),
								"use_mesh_http_backend_fallback": knownvalue.Bool(false),
								"ssh_known_host": knownvalue.MapExact(map[string]knownvalue.Check{
									"host":      knownvalue.StringExact("github.com"),
									"key_type":  knownvalue.StringExact("ssh-rsa"),
									"key_value": knownValueNotEmptyString(),
								}),
								"ssh_private_key": knownvalue.MapExact(map[string]knownvalue.Check{
									"value":       knownvalue.Null(),                                                         // Value is write-only, not stored in state
									"hash":        knownvalue.StringExact("sha256:-----BEGIN OPENSSH PRIVATE KEY-----\n..."), // Hash should be computed from plaintext by backend
									"fingerprint": knownvalue.StringExact("sha256:-----BEGIN OPENSSH PRIVATE KEY-----\n..."), // Fingerprint is computed (hash is used as initial value)
								}),
							}),
						}),
						"outputs": knownvalue.MapExact(map[string]knownvalue.Check{
							"tenant_id": knownvalue.MapExact(map[string]knownvalue.Check{
								"display_name":    knownvalue.StringExact("Tenant ID"),
								"type":            knownvalue.StringExact("STRING"),
								"assignment_type": knownvalue.StringExact("PLATFORM_TENANT_ID"),
							}),
							"sign_in_url": knownvalue.MapExact(map[string]knownvalue.Check{
								"display_name":    knownvalue.StringExact("Sign-in URL"),
								"type":            knownvalue.StringExact("STRING"),
								"assignment_type": knownvalue.StringExact("SIGN_IN_URL"),
							}),
						}),
					})),

					// Version checks - only one draft version exists, so version_latest_release is not set
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("version_latest"), version1(versionStateDraft)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{version1(versionStateDraft)})),
				},
				PostApplyFunc: func() {
					assert.Len(t, mock.store.definitions, 1)
					assert.Len(t, mock.store.versions, 1)
				},
			},
			// Step 2: Update BBD Spec, which will not trigger a new BBD version
			{
				Config: configSpecChange,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("description"), knownvalue.StringExact("An updated building block definition")),
				},
				PostApplyFunc: func() {
					assert.Len(t, mock.store.definitions, 1)
					assert.Len(t, mock.store.versions, 1)
				},
			},
			// Step 3: Update BBD Version Spec with draft=false, which will release the existing BBD version
			{
				Config: configDraftFalse,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// Version checks - draft is now released, so 'version_latest_release' becomes set (content hash does not change though)
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("version_latest_release"), version1(versionStateReleased)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("version_latest"), version1(versionStateReleased)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{version1(versionStateReleased)})),
				},
				PostApplyFunc: func() {
					assert.Len(t, mock.store.definitions, 1)
					assert.Len(t, mock.store.versions, 1)
				},
			},
			// Step 4: Update BBD Version Spec with draft=true again, which will create a new draft version
			{
				Config: configDraftTrueAgain,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// Version checks - draft is now released, so 'version_latest_release' becomes set (content hash does not change though)
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("version_latest_release"), version1(versionStateReleased)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("version_latest"), version2),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
						version1(versionStateReleased),
						version2,
					})),
				},
				PostApplyFunc: func() {
					assert.Len(t, mock.store.definitions, 1)
					assert.Len(t, mock.store.versions, 2)
				},
			},
		},
	})
}

// Shared state for mock clients.
type mockBuildingBlockDefinitionStore struct {
	definitions    map[string]*client.MeshBuildingBlockDefinition
	versions       map[string]*client.MeshBuildingBlockDefinitionVersion
	versionCounter int64
}

type mockBuildingBlockDefinitionClients struct {
	store      *mockBuildingBlockDefinitionStore
	bbd        mockBBDClient
	bbdVersion mockBBDVersionClient
}

func newMockBuildingBlockDefinitionClients() *mockBuildingBlockDefinitionClients {
	store := &mockBuildingBlockDefinitionStore{
		definitions: make(map[string]*client.MeshBuildingBlockDefinition),
		versions:    make(map[string]*client.MeshBuildingBlockDefinitionVersion),
	}
	return &mockBuildingBlockDefinitionClients{
		store:      store,
		bbd:        mockBBDClient{store: store},
		bbdVersion: mockBBDVersionClient{store: store},
	}
}

// Mock BBD Client.
type mockBBDClient struct {
	store *mockBuildingBlockDefinitionStore
}

func (m mockBBDClient) List(_ context.Context, workspaceIdentifier *string) ([]client.MeshBuildingBlockDefinition, error) {
	var result []client.MeshBuildingBlockDefinition
	for _, def := range m.store.definitions {
		if workspaceIdentifier == nil || def.Metadata.OwnedByWorkspace == *workspaceIdentifier {
			result = append(result, *def)
		}
	}
	return result, nil
}

func (m mockBBDClient) Read(_ context.Context, uuid string) (*client.MeshBuildingBlockDefinition, error) {
	if def, ok := m.store.definitions[uuid]; ok {
		return def, nil
	}
	return nil, nil
}

func (m mockBBDClient) Create(_ context.Context, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	created := &client.MeshBuildingBlockDefinition{
		ApiVersion: "v1",
		Kind:       "meshBuildingBlockDefinition",
		Metadata: client.MeshBuildingBlockDefinitionMetadata{
			MeshBuildingBlockDefinitionMetadataBase: client.MeshBuildingBlockDefinitionMetadataBase{
				OwnedByWorkspace: definition.Metadata.OwnedByWorkspace,
				Tags:             definition.Metadata.Tags,
			},
			Uuid:      clientTypes.PtrTo("dummy-uuid-12345"),
			CreatedOn: clientTypes.PtrTo("2024-01-01T00:00:00Z"),
		},
		Spec: definition.Spec,
	}

	m.store.definitions[*created.Metadata.Uuid] = created

	// Create initial empty version (as the backend does)
	m.store.versionCounter++
	versionUuid := fmt.Sprintf("dummy-version-uuid-%d", m.store.versionCounter)
	initialVersionSpec := client.MeshBuildingBlockDefinitionVersionSpec{
		MeshBuildingBlockDefinitionVersionSpecBase: client.MeshBuildingBlockDefinitionVersionSpecBase{
			BuildingBlockDefinitionRef: client.BuildingBlockDefinitionRef{
				Uuid: *created.Metadata.Uuid,
				Kind: "meshBuildingBlockDefinition",
			},
			DeletionMode: client.BuildingBlockDeletionModeDelete.Unwrap(),
		},
		VersionNumber:  clientTypes.PtrTo(int64(1)),
		State:          client.MeshBuildingBlockDefinitionVersionStateDraft.Ptr(),
		Implementation: client.MeshBuildingBlockDefinitionImplementation[*clientTypes.Secret]{},
	}
	m.store.versions[versionUuid] = &client.MeshBuildingBlockDefinitionVersion{
		ApiVersion: "v1",
		Kind:       "meshBuildingBlockDefinitionVersion",
		Metadata: &client.MeshBuildingBlockDefinitionVersionMetadata{
			Uuid:             versionUuid,
			OwnedByWorkspace: definition.Metadata.OwnedByWorkspace,
			CreatedOn:        "2024-01-01T00:00:00Z",
		},
		Spec: initialVersionSpec,
	}
	return created, nil
}

func (m mockBBDClient) Update(_ context.Context, uuid string, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	if existing, ok := m.store.definitions[uuid]; ok {
		existing.Spec = definition.Spec
		existing.Metadata.Tags = definition.Metadata.Tags
		return existing, nil
	}
	return nil, fmt.Errorf("building block definition not found: %s", uuid)
}

func (m mockBBDClient) Delete(_ context.Context, uuid string) error {
	delete(m.store.definitions, uuid)
	return nil
}

type mockBBDVersionClient struct {
	store *mockBuildingBlockDefinitionStore
}

// mockBackendSecretBehavior mocks backend behavior in the sense that it consumes the plaintext secret and returns a hash of the secret only.
func mockBackendSecretBehavior(versionSpec, existingVersionSpec *client.MeshBuildingBlockDefinitionVersionSpec) {
	type getTargetFunc func(*client.MeshBuildingBlockDefinitionVersionSpec) **clientTypes.Secret
	handleSecret := func(getTarget getTargetFunc) {
		secretTarget := getTarget(versionSpec)
		if secretTarget == nil {
			return
		}
		secret := *secretTarget
		if secret != nil && secret.Plaintext != nil {
			secret.Hash = clientTypes.PtrTo(fmt.Sprintf("sha256:%s", *secret.Plaintext))
			secret.Plaintext = nil
		} else if existingVersionSpec != nil {
			existingSecretTarget := getTarget(existingVersionSpec)
			if existingSecretTarget != nil {
				existingSecret := *existingSecretTarget
				if existingSecret == nil {
					panic("existing secret is nil")
				} else if secret == nil {
					panic("new secret is nil")
				}
				if secret.Hash != nil && existingSecret.Hash != nil && *secret.Hash != *existingSecret.Hash {
					panic("existing secret hash mismatch")
				} else if secret.Plaintext != nil {
					panic("secret plaintext non-empty")
				}
				return
			} else if secret == nil || secret.Plaintext == nil {
				panic("changing to new secret without plaintext")
			}
		} else if secret == nil || secret.Hash == nil {
			panic("inconsistent create or update of secret in mock client")
		}
	}
	handleSecret(func(versionSpec *client.MeshBuildingBlockDefinitionVersionSpec) **clientTypes.Secret {
		if versionSpec.Implementation.Terraform != nil {
			return &versionSpec.Implementation.Terraform.SSHPrivateKey
		}
		return nil
	})
	handleSecret(func(versionSpec *client.MeshBuildingBlockDefinitionVersionSpec) **clientTypes.Secret {
		if versionSpec.Implementation.GitlabPipeline != nil {
			return &versionSpec.Implementation.GitlabPipeline.PipelineTriggerToken
		}
		return nil
	})

	orAny := func(getTarget func(*client.MeshBuildingBlockDefinitionVersionSpec) *clientTypes.SecretOrAny) getTargetFunc {
		return func(spec *client.MeshBuildingBlockDefinitionVersionSpec) **clientTypes.Secret {
			var target *clientTypes.Secret
			secretOrAnyTarget := getTarget(spec)
			if secretOrAnyTarget != nil && secretOrAnyTarget.HasX() {
				target = &secretOrAnyTarget.X
			}
			return &target
		}
	}

	for inputKey, input := range versionSpec.Inputs {
		if input.IsSensitive {
			handleSecret(orAny(func(spec *client.MeshBuildingBlockDefinitionVersionSpec) *clientTypes.SecretOrAny {
				if input := spec.Inputs[inputKey]; input != nil {
					return &input.Argument
				}
				return nil
			}))
			handleSecret(orAny(func(spec *client.MeshBuildingBlockDefinitionVersionSpec) *clientTypes.SecretOrAny {
				if input := spec.Inputs[inputKey]; input != nil {
					return &input.DefaultValue
				}
				return nil
			}))
		}
	}
}

func (m mockBBDVersionClient) List(_ context.Context, buildingBlockDefinitionUuid string) ([]client.MeshBuildingBlockDefinitionVersion, error) {
	var result []client.MeshBuildingBlockDefinitionVersion
	for _, version := range m.store.versions {
		if version.Spec.BuildingBlockDefinitionRef.Uuid == buildingBlockDefinitionUuid {
			result = append(result, *version)
		}
	}
	return result, nil
}

func (m mockBBDVersionClient) Create(_ context.Context, versionSpec client.MeshBuildingBlockDefinitionVersionSpec) (*client.MeshBuildingBlockDefinitionVersion, error) {
	m.store.versionCounter++
	versionUuid := fmt.Sprintf("dummy-version-uuid-%d", m.store.versionCounter)
	// Compute hashes for all secrets in the spec
	mockBackendSecretBehavior(&versionSpec, nil)

	created := &client.MeshBuildingBlockDefinitionVersion{
		ApiVersion: "v1",
		Kind:       "meshBuildingBlockDefinitionVersion",
		Metadata: &client.MeshBuildingBlockDefinitionVersionMetadata{
			Uuid:             versionUuid,
			OwnedByWorkspace: "my-workspace",
			CreatedOn:        "2024-01-01T00:00:00Z",
		},
		Spec: versionSpec,
	}

	m.store.versions[versionUuid] = created
	return created, nil
}

func (m mockBBDVersionClient) Update(_ context.Context, uuid string, versionSpec client.MeshBuildingBlockDefinitionVersionSpec) (*client.MeshBuildingBlockDefinitionVersion, error) {
	if existing, ok := m.store.versions[uuid]; ok {
		// Compute hashes for all secrets in the spec
		mockBackendSecretBehavior(&versionSpec, &existing.Spec)
		existing.Spec = versionSpec
		return existing, nil
	}
	return nil, fmt.Errorf("building block definition version not found: %s", uuid)
}
