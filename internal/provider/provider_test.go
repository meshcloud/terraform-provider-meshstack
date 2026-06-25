package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
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

// IsMockClientTest returns true when TF_ACC is not set, meaning tests run with a mock client.
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
// scratch-config-testing skill.
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

// ApplyAndTest runs a TF test case. When TF_ACC is not set, it uses a mock
// client (unit test mode). When TF_ACC is set, it runs against a real meshStack.
// All tests using ApplyAndTest run in parallel.
//
// When MESHSTACK_SCRATCH_DUMP is set, it instead dumps each step's config to disk and
// returns without running the test (see dumpStepConfigs).
func ApplyAndTest(t *testing.T, testCase resource.TestCase) {
	t.Helper()

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
		testCase.PreCheck = func() { DefaultTestPreCheck(t) }
		testCase.ProtoV6ProviderFactories = ProviderFactoriesForTest()
	}

	resource.Test(t, testCase)
}

func DefaultTestPreCheck(t *testing.T) {
	t.Helper()
	endpoint := os.Getenv(envKeyMeshstackEndpoint)
	require.Truef(t, strings.HasPrefix(endpoint, "http://localhost"),
		"Env %s='%s' does not start with http://localhost, only locally running meshStacks should be used for tests", envKeyMeshstackEndpoint, endpoint)
	require.NotEmptyf(t, os.Getenv(envKeyMeshstackApiKey), "Env %s empty, please set before running", envKeyMeshstackApiKey)
	require.NotEmptyf(t, os.Getenv(envKeyMeshstackApiSecret), "Env %s empty, please set before running", envKeyMeshstackApiSecret)
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
