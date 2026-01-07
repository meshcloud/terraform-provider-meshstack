package provider

const (
	TenantTargetType    = "TENANT"
	WorkspaceTargetType = "WORKSPACE"
)

type buildingBlockDefinitionResourceModel struct {
	Metadata buildingBlockDefinitionMetadata `tfsdk:"metadata"`
	Spec     buildingBlockDefinitionSpec     `tfsdk:"spec"`

	Draft                  bool     `tfsdk:"draft"`
	OnlyApplyOncePerTenant bool     `tfsdk:"only_apply_once_per_tenant"`
	DeletionMode           string   `tfsdk:"deletion_mode"`
	RunnerRef              string   `tfsdk:"runner_ref"`
	DependencyRefs         []string `tfsdk:"dependency_refs"`

	Inputs         map[string]buildingBlockInput  `tfsdk:"inputs"`
	Outputs        map[string]buildingBlockOutput `tfsdk:"outputs"`
	Implementation buildingBlockImplementation    `tfsdk:"implementation"`

	VersionLatest        buildingBlockDefinitionVersion   `tfsdk:"version_latest"`
	VersionLatestRelease buildingBlockDefinitionVersion   `tfsdk:"version_latest_release"`
	Versions             []buildingBlockDefinitionVersion `tfsdk:"versions"`
}

type buildingBlockImplementation struct {
	Terraform     *buildingBlockTerraformImpl     `tfsdk:"terraform"`
	GithubActions *buildingBlockGithubActionsImpl `tfsdk:"github_actions"`
}

type buildingBlockTerraformImpl struct {
	TerraformVersion           string                     `tfsdk:"terraform_version"`
	RepositoryUrl              string                     `tfsdk:"repository_url"`
	Async                      bool                       `tfsdk:"async"`
	RepositoryPath             *string                    `tfsdk:"repository_path"`
	RefName                    *string                    `tfsdk:"ref_name"`
	SshPrivateKey              *string                    `tfsdk:"ssh_private_key"`
	SshPrivateKeyVersion       *string                    `tfsdk:"ssh_private_key_version"`
	UseMeshHttpBackendFallback bool                       `tfsdk:"use_mesh_http_backend_fallback"`
	SshKnownHost               *buildingBlockSshKnownHost `tfsdk:"ssh_known_host"`
}

type buildingBlockSshKnownHost struct {
	Host     string `tfsdk:"host"`
	KeyType  string `tfsdk:"key_type"`
	KeyValue string `tfsdk:"key_value"`
}

type buildingBlockGithubActionsImpl struct {
	Repository                   *string `tfsdk:"repository"`
	Branch                       *string `tfsdk:"branch"`
	ApplyWorkflow                *string `tfsdk:"apply_workflow"`
	DestroyWorkflow              *string `tfsdk:"destroy_workflow"`
	SourcePlatformFullIdentifier *string `tfsdk:"source_platform_full_identifier"`
}

type buildingBlockDefinitionMetadata struct {
	Uuid                string              `tfsdk:"uuid"`
	OwnedByWorkspace    string              `tfsdk:"owned_by_workspace"`
	Tags                map[string][]string `tfsdk:"tags"`
	CreatedOn           string              `tfsdk:"created_on"`
	MarkedForDeletionOn string              `tfsdk:"marked_for_deletion_on"`
	MarkedForDeletionBy string              `tfsdk:"marked_for_deletion_by"`
}

type buildingBlockDefinitionSpec struct {
	DisplayName                     string   `tfsdk:"display_name"`
	Symbol                          *string  `tfsdk:"symbol"`
	Description                     string   `tfsdk:"description"`
	Readme                          *string  `tfsdk:"readme"`
	SupportUrl                      *string  `tfsdk:"support_url"`
	DocumentationUrl                *string  `tfsdk:"documentation_url"`
	SupportedPlatforms              []string `tfsdk:"supported_platforms"`
	RunTransparency                 bool     `tfsdk:"run_transparency"`
	UseInLandingZonesOnly           bool     `tfsdk:"use_in_landing_zones_only"`
	TargetType                      string   `tfsdk:"target_type"`
	NotificationSubscriberUsernames []string `tfsdk:"notification_subscriber_usernames"`
}

type buildingBlockDefinitionVersion struct {
	Uuid   string `tfsdk:"uuid"`
	Number int64  `tfsdk:"number"`
	State  string `tfsdk:"state"`
}

type buildingBlockInput struct {
	DisplayName                 string   `tfsdk:"display_name"`
	Type                        string   `tfsdk:"type"`
	AssignmentType              string   `tfsdk:"assignment_type"`
	Argument                    *string  `tfsdk:"argument"`
	IsEnvironment               bool     `tfsdk:"is_environment"`
	IsSensitive                 bool     `tfsdk:"is_sensitive"`
	UpdateableByConsumer        bool     `tfsdk:"updateable_by_consumer"`
	SelectableValues            []string `tfsdk:"selectable_values"`
	DefaultValue                *string  `tfsdk:"default_value"`
	Description                 *string  `tfsdk:"description"`
	ValueValidationRegex        *string  `tfsdk:"value_validation_regex"`
	ValidationRegexErrorMessage *string  `tfsdk:"validation_regex_error_message"`
}

type buildingBlockOutput struct {
	DisplayName    string `tfsdk:"display_name"`
	Type           string `tfsdk:"type"`
	AssignmentType string `tfsdk:"assignment_type"`
}
