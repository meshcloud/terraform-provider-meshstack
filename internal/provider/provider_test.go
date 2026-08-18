package provider

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/pkg/login"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

// ProviderFactoriesForTest are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
func ProviderFactoriesForTest(opts ...providerOption) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"meshstack":       providerserver.NewProtocol6WithError(New("test", opts...)()),
		"meshstack-other": providerserver.NewProtocol6WithError(New("test-other", opts...)()),
	}
}

// IsMockClientTest reports whether tests run against the in-memory mock client (TF_ACC unset)
// instead of a real backend. Mock and acceptance runs must stay in lock-step; see the
// acceptance-testing skill for when gating a step or assertion on this is warranted.
func IsMockClientTest() bool {
	return os.Getenv("TF_ACC") == ""
}

// AdminWorkspaceIdentifier is the admin (partner) workspace seeded by the dev dump that the
// acceptance tests run against — both locally and in CI (where the dev-dump-seeded meshStack is
// brought up as an ephemeral service). The dev dump sets meshfed's
// web.register.default-partner-identifier to "demo-partner", which is the partner/admin workspace.
// Some resources (e.g. Entra ID integrations) can only be owned by the admin workspace — meshfed
// rejects any other owner — so tests for them hardcode this identifier. It is specific to the dev
// dump and does not exist on other meshStack instances, which is fine: the acceptance suite only
// ever runs against the dev dump.
const AdminWorkspaceIdentifier = "demo-partner"

// envKeyScratchDump, when non-empty, makes ApplyAndTest dump each step's HCL config to disk
// (as a standalone, re-runnable config) instead of running the test. Set it to "1"/"true" to
// dump into the repo-root scratch/ dir, or to a directory path to dump there. See the
// scratch-config skill.
const envKeyScratchDump = "MESHSTACK_SCRATCH_DUMP"

// scratchProviderTf is written alongside each dumped main.tf so the config resolves the
// dev-built provider via a dev_overrides CLI config (TF_CLI_CONFIG_FILE). Credentials and
// endpoint come from the MESHSTACK_* environment variables.
const scratchProviderTf = `terraform {
  required_providers {
    meshstack = {
      source = "meshcloud/meshstack"
    }
  }
}

provider "meshstack" {
  # endpoint/credentials read from MESHSTACK_ENDPOINT, MESHSTACK_API_KEY, MESHSTACK_API_SECRET
}
`

// restrictedTagLocks serialize the tests that create a restricted tag definition with a default
// value. Such a definition is global to its meshObject kind: the backend adds the tag to every
// resource of that kind, so a test running at the same time fails its empty-plan and import checks.
// The random name suffix isolates the tag's key, not its effect.
var restrictedTagLocks = map[string]*sync.RWMutex{
	client.MeshObjectKind.BuildingBlockDefinition: {},
	client.MeshObjectKind.LandingZone:             {},
	client.MeshObjectKind.Project:                 {},
}

type ApplyAndTestOption func(*applyAndTestOptions)

type applyAndTestOptions struct {
	LockExclusiveKinds []string
}

// TouchesExclusively marks a test that creates a restricted tag definition with a default value for
// the given kind. See restrictedTagLocks for why such a test cannot run next to others.
func TouchesExclusively(kind string) ApplyAndTestOption {
	return func(options *applyAndTestOptions) {
		options.LockExclusiveKinds = append(options.LockExclusiveKinds, kind)
	}
}

// acquireRestrictedTagLocks must run after t.Parallel() has returned. Until then the test is
// paused, and a read lock taken earlier would make a writer wait for a test that has not started.
func acquireRestrictedTagLocks(t *testing.T, options applyAndTestOptions) func() {
	t.Helper()

	slices.Sort(options.LockExclusiveKinds)

	for _, kind := range options.LockExclusiveKinds {
		if _, ok := restrictedTagLocks[kind]; !ok {
			t.Fatalf("TouchesExclusively(%q): unknown meshObject kind, add it to restrictedTagLocks", kind)
		}
	}

	releases := make([]func(), 0, len(restrictedTagLocks))
	// Sorted, so every caller takes the locks in the same order. Ranging over the map directly
	// would let writers of different kinds deadlock.
	for _, kind := range slices.Sorted(maps.Keys(restrictedTagLocks)) {
		lock := restrictedTagLocks[kind]
		if slices.Contains(options.LockExclusiveKinds, kind) {
			lock.Lock()
			releases = append(releases, lock.Unlock)
		} else {
			lock.RLock()
			releases = append(releases, lock.RUnlock)
		}
	}

	return func() {
		for _, release := range releases {
			release()
		}
	}
}

// ApplyAndTest runs a TF test case. When TF_ACC is not set, it uses a mock
// client (unit test mode). When TF_ACC is set, it runs against a real meshStack.
// All tests using ApplyAndTest run in parallel, except that a test marked with
// TouchesExclusively runs alone.
//
// When MESHSTACK_SCRATCH_DUMP is set, it instead dumps each step's config to disk and
// returns without running the test (see dumpStepConfigs).
func ApplyAndTest(t *testing.T, testCase resource.TestCase, opts ...ApplyAndTestOption) {
	t.Helper()

	var options applyAndTestOptions
	for _, opt := range opts {
		opt(&options)
	}

	if target := os.Getenv(envKeyScratchDump); target != "" {
		dumpStepConfigs(t, target, testCase.Steps)
		return
	}

	if IsMockClientTest() {
		mockClient := clientmock.NewMock()
		testCase.IsUnitTest = true
		testCase.ProtoV6ProviderFactories = ProviderFactoriesForTest(func(provider *MeshStackProvider) {
			provider.clientFactory = func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics) {
				return mockClient.AsClient(), nil
			}
		})
	} else {
		// os.Setenv (not t.Setenv) because t.Setenv is incompatible with the t.Parallel() call below.
		require.NoError(t, os.Setenv("MESHSTACK_SKIP_VERSION_CHECK", "true")) //nolint:usetesting // see comment above
		t.Parallel()
		releaseRestrictedTagLocks := acquireRestrictedTagLocks(t, options)
		defer releaseRestrictedTagLocks()
		testCase.PreCheck = func() { DefaultTestPreCheck(t) }
		testCase.ProtoV6ProviderFactories = ProviderFactoriesForTest()
	}

	resource.Test(t, testCase)
}

func DefaultTestPreCheck(t *testing.T) {
	t.Helper()
	endpoint := os.Getenv(login.EnvKeyEndpoint)
	require.Truef(t, strings.HasPrefix(endpoint, "http://localhost"),
		"Env %s='%s' does not start with http://localhost, only locally running meshStacks should be used for tests", login.EnvKeyEndpoint, endpoint)
	require.NotEmptyf(t, os.Getenv(login.EnvKeyApiKey), "Env %s empty, please set before running", login.EnvKeyApiKey)
	require.NotEmptyf(t, os.Getenv(login.EnvKeyApiSecret), "Env %s empty, please set before running", login.EnvKeyApiSecret)
}

// dumpStepConfigs writes each test step's HCL config to
// <base>/<sanitized test name>/stepNN/{main.tf,provider.tf} so it can be run standalone
// against a local meshStack via the dev-built provider. target is either "1"/"true"
// (dump into the repo-root scratch/ dir) or a directory path. Steps without a Config
// (e.g. import-only steps) are skipped.
func dumpStepConfigs(t *testing.T, target string, steps []resource.TestStep) {
	t.Helper()

	base := target
	if base == "1" || base == "true" {
		base = "scratch"
	}
	if !filepath.IsAbs(base) {
		base = filepath.Join(moduleRoot(t), base)
	}

	testDir := filepath.Join(base, sanitizeTestName(t.Name()))

	written := 0
	for i, step := range steps {
		if step.Config == "" {
			continue
		}
		stepDir := filepath.Join(testDir, fmt.Sprintf("step%02d", i+1))
		require.NoError(t, os.MkdirAll(stepDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(stepDir, "main.tf"), []byte(step.Config), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(stepDir, "provider.tf"), []byte(scratchProviderTf), 0o644))
		written++
	}

	t.Logf("dumped %d step config(s) to %s", written, testDir)
}

// moduleRoot returns the repository root by walking up from the working directory until it
// finds go.mod (go test runs in the package dir, e.g. internal/provider/).
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqualf(t, parent, dir, "could not locate go.mod above %s", dir)
		dir = parent
	}
}

// ResourceSchemaForTest returns a resource's current schema, so tests can assert on its attributes
// without importing the framework themselves.
func ResourceSchemaForTest(t *testing.T, r fwresource.Resource) fwschema.Schema {
	t.Helper()

	var resp fwresource.SchemaResponse
	r.Schema(context.Background(), fwresource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "building schema: %s", resp.Diagnostics)

	return resp.Schema
}

// HasNestedAttributeForTest reports whether a schema's single-nested attribute declares the given
// child, e.g. spec.quotas.
func HasNestedAttributeForTest(t *testing.T, s fwschema.Schema, parent, child string) bool {
	t.Helper()

	nested, ok := s.Attributes[parent].(fwschema.SingleNestedAttribute)
	require.Truef(t, ok, "%s attribute is %T, want SingleNestedAttribute", parent, s.Attributes[parent])

	_, found := nested.Attributes[child]
	return found
}

// UpgradeResourceStateFromJSON drives a resource's state upgrader over raw prior-state JSON the way
// the framework does — JSON keys absent from the prior schema are skipped, and attributes the prior
// schema declares but the JSON omits decode to null — then reads the upgraded state into target.
// Returns the upgrade's diagnostics so a test can assert on a rejected upgrade too.
//
// Driving the upgrader directly is what makes a pre-release state shape testable at all: reproducing
// it through Terraform would mean installing an older published provider and applying against a real
// backend.
func UpgradeResourceStateFromJSON(t *testing.T, r fwresource.ResourceWithUpgradeState, version int64, rawJSON string, target any) diag.Diagnostics {
	t.Helper()

	upgraded, diags := UpgradeResourceState(t, r, version, rawJSON)
	if diags.HasError() {
		return diags
	}
	diags.Append(upgraded.state.Get(context.Background(), target)...)

	return diags
}

// UpgradedState wraps the state an upgrader produced so a test can read it one attribute at a time
// without importing the framework.
type UpgradedState struct {
	t     *testing.T
	state tfsdk.State
}

// Attribute reads the attribute at the given dotted path (e.g. "spec.display_name") into target.
func (u UpgradedState) Attribute(attributePath string, target any) {
	u.t.Helper()

	p := path.Empty()
	for _, step := range strings.Split(attributePath, ".") {
		p = p.AtName(step)
	}
	diags := u.state.GetAttribute(context.Background(), p, target)
	require.Falsef(u.t, diags.HasError(), "reading %s from the upgraded state: %s", attributePath, diags)
}

// UpgradeResourceState is UpgradeResourceStateFromJSON without the final whole-model read, for a
// resource whose model cannot be read back through plain framework reflection.
func UpgradeResourceState(t *testing.T, r fwresource.ResourceWithUpgradeState, version int64, rawJSON string) (UpgradedState, diag.Diagnostics) {
	t.Helper()

	ctx := context.Background()

	upgrader, ok := r.UpgradeState(ctx)[version]
	require.Truef(t, ok, "resource declares no state upgrader for version %d", version)
	require.NotNilf(t, upgrader.PriorSchema, "state upgrader for version %d has no PriorSchema", version)

	priorValue, err := tfprotov6.RawState{JSON: []byte(rawJSON)}.UnmarshalWithOpts(
		upgrader.PriorSchema.Type().TerraformType(ctx),
		tfprotov6.UnmarshalOpts{ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true}},
	)
	require.NoError(t, err, "decoding prior state against the upgrader's PriorSchema")

	req := fwresource.UpgradeStateRequest{State: &tfsdk.State{Raw: priorValue, Schema: *upgrader.PriorSchema}}
	// Raw is left unset, as the framework does when it calls the upgrader.
	resp := fwresource.UpgradeStateResponse{State: tfsdk.State{Schema: ResourceSchemaForTest(t, r)}}

	upgrader.StateUpgrader(ctx, req, &resp)

	return UpgradedState{t: t, state: resp.State}, resp.Diagnostics
}

// sanitizeTestName turns a *testing.T name into a relative path. Subtest separators ("/")
// are kept so subtests nest into directories; any other unsafe character becomes "_".
func sanitizeTestName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '/' || r == '.' || r == '_' || r == '-':
			return r
		default:
			return '_'
		}
	}, name)
}
