package provider

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	//go:embed testdata/bbd/version-spec.json
	versionSpecJson []byte
	//go:embed testdata/bbd/version-spec-irrelevant-change.json
	versionSpecIrrelevantChangeJson []byte
	//go:embed testdata/bbd/version-spec-with-reordered-inputs.json
	versionSpecReorderedInputsJson []byte
	//go:embed testdata/bbd/version-spec-with-displayOrder.json
	versionSpecWithDisplayOrderJson []byte
	//go:embed testdata/bbd/version-spec-relevant-change.json
	versionSpecRelevantChangeJson []byte
	//go:embed testdata/bbd/version-spec-with-plaintext-secret.json
	versionSpecPlaintextSecretJson []byte
	//go:embed testdata/bbd/version-spec-null-outputs.json
	versionSpecNullOutputsChangeJson []byte
	//go:embed testdata/bbd/version-spec-empty-outputs.json
	versionSpecEmptyOutputsJson []byte
)

// Test_versionContentHash pins the raw digest (hashValue) each fixture produces under the *current* hashing
// algorithm. It asserts the digest alone, not the versioned toBase64 encoding, so the two independent concerns
// stay separate:
//   - The digest is a property of the algorithm. It is allowed to change when the algorithm changes (e.g.
//     folding display_order back into the hash). But any such change is a breaking change to already-stored
//     hashes, so it MUST be paired with a currentHashVersion bump — otherwise a hash stored by an older
//     provider compares same-version against the new digest and is reported as spuriously "changed", rerunning
//     every already-released building block. A bump instead makes the old hash incomparable, so it is
//     recomputed at the new version (Test_contentHash_compareToStored_acrossVersions).
//   - The version prefix is pinned separately in Test_contentHash_currentVersion.
//
// So: change a digest below only alongside a deliberate hash-version bump, never on its own.
func Test_versionContentHash(t *testing.T) {
	const (
		digestExample      = "814c7b4eb579d555f4f1589bd34c4f913d250fe1d5fda7b2fdffe05e75fcd910"
		digestRelevant     = "6ddbadbe5eb3baa76e7a2488f22ce62bd0e94eb380cbfbd55724231f704264dc"
		digestNullOutputs  = "1a24f70de617e64fc258ceca4a4b7159ecebd9a95acd4ea40851e443c080038e"
		digestDisplayOrder = "a7cba4239e3d448387d188d1457bd3e00aa66694244dd6e26e8d7a055c9c4075"
	)
	require.NotEqual(t, digestExample, digestRelevant)

	tests := []struct {
		name string
		json []byte
		want string
	}{
		{"example", versionSpecJson, digestExample},
		{"reordered inputs hash the same as example", versionSpecReorderedInputsJson, digestExample},
		{"irrelevant change hashes the same as example", versionSpecIrrelevantChangeJson, digestExample},
		{"relevant change hashes differently", versionSpecRelevantChangeJson, digestRelevant},
		{"null outputs", versionSpecNullOutputsChangeJson, digestNullOutputs},
		{"empty outputs hash the same as null outputs", versionSpecEmptyOutputsJson, digestNullOutputs},
		{"display_order affects the hash", versionSpecWithDisplayOrderJson, digestDisplayOrder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, forTestCalculateContentHash(t, tt.json).hashValue)
		})
	}
}

// Test_contentHash_currentVersion is the single guard for the hash-version prefix. Bumping currentHashVersion
// is a deliberate act (it makes every previously stored hash incomparable, forcing a recompute at the new
// version instead of a spurious "changed" — see Test_contentHash_compareToStored_acrossVersions), so it must
// be paired with an intentional edit here.
func Test_contentHash_currentVersion(t *testing.T) {
	require.Equal(t, 4, currentHashVersion)

	h := forTestCalculateContentHash(t, versionSpecJson)
	require.Equal(t, currentHashVersion, h.hashVersion)

	decoded, err := base64.StdEncoding.DecodeString(h.toBase64())
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(decoded), fmt.Sprintf("v%d:", currentHashVersion)),
		"toBase64 must carry the current version prefix, got %q", string(decoded))
}

func Test_contentHash_base64Roundtrip(t *testing.T) {
	original := BuildingBlockDefinitionVersionContentHash{hashVersion: currentHashVersion, hashValue: "foobar"}

	var decoded BuildingBlockDefinitionVersionContentHash
	require.NoError(t, decoded.loadFromBase64(original.toBase64()))

	assert.Equal(t, original, decoded)
}

func Test_contentHash_loadFromBase64_malformed(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
	}{
		{"not base64", "this is not base64 !!!"},
		{"base64 without colon", base64.StdEncoding.EncodeToString([]byte("noColonHere"))},
		{"base64 with too many colons", base64.StdEncoding.EncodeToString([]byte("v2:v3:bbb"))},
		{"base64 with non-integer version", base64.StdEncoding.EncodeToString([]byte("vX:bbb"))},
		{"empty string", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := BuildingBlockDefinitionVersionContentHash{hashVersion: currentHashVersion, hashValue: "seed"}
			assert.Error(t, h.loadFromBase64(tt.encoded))
		})
	}
}

// storedHash renders a (version, hashValue) into the string compareToStored later reads back. The only
// difference between versions is whether the string is base64-encoded: v1 predates base64 and is persisted
// raw, every later version goes through toBase64. Callers must understand the raw v1 form is exactly its
// hashValue, which itself carries the "v1:" prefix — so pass the full "v1:..." string as value for v1
// (a bare digest for later versions).
func storedHash(version int, value string) string {
	h := BuildingBlockDefinitionVersionContentHash{hashVersion: version, hashValue: value}
	if version == 1 {
		return h.hashValue
	}
	return h.toBase64()
}

// Test_contentHash_compareToStored_acrossVersions is the heart of the version-bump contract: two hashes are
// only comparable when their versions match. A stored hash from an older provider (v1, v2 or v3) is therefore
// incomparable to a current v4 hash — the case that must NOT be reported as "changed", because that would
// rerun every already-released building block after a provider upgrade (issue behind the v3->v4 bump).
func Test_contentHash_compareToStored_acrossVersions(t *testing.T) {
	tests := []struct {
		name         string
		currentVer   int
		currentValue string
		storedVer    int
		storedValue  string
		want         hashComparison
	}{
		{"same version, same value -> same", 4, "aaa", 4, "aaa", hashSame},
		{"same version, different value -> different", 4, "aaa", 4, "bbb", hashDifferent},
		{"legacy v1 self-consistent -> same", 1, "v1:h", 1, "v1:h", hashSame},
		{"legacy v1 different value -> different", 1, "v1:h", 1, "v1:other", hashDifferent},
		{"stored v1, current v2 -> incomparable", 2, "aaa", 1, "v1:h", hashIncomparable},
		{"stored v1, current v4 -> incomparable", 4, "aaa", 1, "v1:h", hashIncomparable},
		{"stored v2, current v4 -> incomparable", 4, "aaa", 2, "aaa", hashIncomparable},
		{"stored v3, current v4 -> incomparable", 4, "aaa", 3, "aaa", hashIncomparable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := BuildingBlockDefinitionVersionContentHash{hashVersion: tt.currentVer, hashValue: tt.currentValue}
			assert.Equal(t, tt.want, current.compareToStored(storedHash(tt.storedVer, tt.storedValue)))
		})
	}
}

func Test_contentHash_compareToStored_unparsable(t *testing.T) {
	current := BuildingBlockDefinitionVersionContentHash{hashVersion: currentHashVersion, hashValue: "aaa"}

	assert.Equal(t, hashIncomparable, current.compareToStored("not base64 !!!"))
	assert.Equal(t, hashIncomparable, current.compareToStored(""))
	// A receiver that never loaded (zero version) cannot be compared to anything.
	assert.Equal(t, hashIncomparable, BuildingBlockDefinitionVersionContentHash{}.compareToStored(current.toBase64()))
}

func Test_contentHash_changeDetectionEndToEnd(t *testing.T) {
	stored := forTestCalculateContentHash(t, versionSpecJson).toBase64()

	sameAgain := forTestCalculateContentHash(t, versionSpecJson)
	irrelevant := forTestCalculateContentHash(t, versionSpecIrrelevantChangeJson)
	relevant := forTestCalculateContentHash(t, versionSpecRelevantChangeJson)

	assert.Equal(t, hashSame, sameAgain.compareToStored(stored))
	assert.Equal(t, hashSame, irrelevant.compareToStored(stored))
	assert.Equal(t, hashDifferent, relevant.compareToStored(stored))
}

// Mirrors the released-version immutability guard's fail-safe: when the stored hash is from an older provider,
// it is incomparable to the current-version hash.
// The guard then recomputes the released spec from state AT THE CURRENT version so the comparison becomes
// meaningful again — a genuine change is detected (would reject) and an unchanged spec is recognized (no-op).
func Test_contentHash_incomparableStoredHash_resolvedByRecomputeAtCurrentVersion(t *testing.T) {
	changedPlanHash := forTestCalculateContentHash(t, versionSpecRelevantChangeJson)
	unchangedPlanHash := forTestCalculateContentHash(t, versionSpecJson)

	// A legacy/stale stored hash is incomparable to the current-version plan hash - the case that used to
	// be silently treated as "no change".
	staleStored := "v1:someLegacyHashValue"
	require.Equal(t, hashIncomparable, changedPlanHash.compareToStored(staleStored))

	// Guard's fail-safe: recompute the released spec (the authoritative released version in state) at the
	// current version, then compare like-for-like.
	authoritative := forTestCalculateContentHash(t, versionSpecJson)

	// A genuine change is now detected (guard rejects the mutation of an immutable version) ...
	assert.Equal(t, hashDifferent, changedPlanHash.compareToStored(authoritative.toBase64()))
	// ... and an unchanged spec is a no-op (no spurious rejection).
	assert.Equal(t, hashSame, unchangedPlanHash.compareToStored(authoritative.toBase64()))
}

func Test_versionContentHash_plaintextSecret(t *testing.T) {
	var versionSpec client.MeshBuildingBlockDefinitionVersionSpec
	require.NoError(t, json.Unmarshal(versionSpecPlaintextSecretJson, &versionSpec))
	var diags diag.Diagnostics
	calculateBuildingBlockDefinitionVersionContentHash(versionSpec, &diags)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Detail(), "version_spec carries a plaintext secret value, which must not be hashed")
}

// Test_versionContentHash_userDataNamedPlaintext guards against the false positive from issue #196: an
// input named "plaintext" is a user-chosen map key, not a secret, so it must hash successfully rather than
// trip the plaintext safeguard (the old JSON-key-based safeguard rejected it).
func Test_versionContentHash_userDataNamedPlaintext(t *testing.T) {
	const userDataJson = `{
		"inputs": {
			"plaintext": {
				"displayName": "Plaintext",
				"type": "STRING",
				"assignmentType": "STATIC",
				"argument": "hello"
			}
		},
		"implementation": {"terraform": {}}
	}`
	var versionSpec client.MeshBuildingBlockDefinitionVersionSpec
	require.NoError(t, json.Unmarshal([]byte(userDataJson), &versionSpec))
	var diags diag.Diagnostics
	actualHashStr := calculateBuildingBlockDefinitionVersionContentHash(versionSpec, &diags).toBase64()
	require.Empty(t, diags)
	assert.NotEmpty(t, actualHashStr)
}

// Test_versionContentHash_ignoresPerVersionFields pins the invariant that the content hash is independent
// of the per-BBD buildingBlockDefinitionRef and the per-version versionNumber/state (all stripped before
// hashing). Were they hashed, the same version_spec would hash differently across BBDs/versions, breaking
// the released-version immutability and change-detection comparisons. This is the focused, behavioral
// replacement for the old DisallowMapKeys("buildingBlockDefinitionRef") safeguard, which - like the
// "plaintext" one - would have misfired on a user-chosen input/output named "buildingBlockDefinitionRef".
func Test_versionContentHash_ignoresPerVersionFields(t *testing.T) {
	var base, mutated client.MeshBuildingBlockDefinitionVersionSpec
	require.NoError(t, json.Unmarshal(versionSpecJson, &base))
	require.NoError(t, json.Unmarshal(versionSpecJson, &mutated))

	mutated.BuildingBlockDefinitionRef = &client.UuidRef{
		Kind: client.MeshObjectKind.BuildingBlockDefinition,
		Uuid: "a-different-bbd-uuid",
	}
	mutated.VersionNumber = new(int64(99))
	mutated.State = client.MeshBuildingBlockDefinitionVersionStateReleased.Ptr()

	var diags diag.Diagnostics
	baseHash := calculateBuildingBlockDefinitionVersionContentHash(base, &diags).toBase64()
	mutatedHash := calculateBuildingBlockDefinitionVersionContentHash(mutated, &diags).toBase64()
	require.Empty(t, diags)
	assert.Equal(t, baseHash, mutatedHash, "buildingBlockDefinitionRef/versionNumber/state must not affect the content hash")
}

func forTestCalculateContentHash(t *testing.T, raw []byte) BuildingBlockDefinitionVersionContentHash {
	t.Helper()

	var spec client.MeshBuildingBlockDefinitionVersionSpec
	require.NoError(t, json.Unmarshal(raw, &spec))

	var diags diag.Diagnostics
	h := calculateBuildingBlockDefinitionVersionContentHash(spec, &diags)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	require.Empty(t, diags)
	require.NotNil(t, h)

	return h
}
