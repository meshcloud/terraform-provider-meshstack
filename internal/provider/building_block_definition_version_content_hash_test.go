package provider

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

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

func Test_contentHash_compareToStored(t *testing.T) {
	hash1 := BuildingBlockDefinitionVersionContentHash{hashVersion: 2, hashValue: "aaa"}
	hash2 := BuildingBlockDefinitionVersionContentHash{hashVersion: 2, hashValue: "bbb"}

	tests := []struct {
		name    string
		current BuildingBlockDefinitionVersionContentHash
		stored  string
		want    hashComparison
	}{
		{"same version, same value -> same", hash1, hash1.toBase64(), hashSame},
		{"same version, different value -> different", hash1, hash2.toBase64(), hashDifferent},
		{"stored v1 raw, current v2 -> incomparable", hash1, "v1:someOldHash", hashIncomparable},
		{"stored unparsable -> incomparable", hash1, "not base64 !!!", hashIncomparable},
		{"stored empty -> incomparable", hash1, "", hashIncomparable},
		{"unparsable receiver -> incomparable", BuildingBlockDefinitionVersionContentHash{}, hash1.toBase64(), hashIncomparable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.current.compareToStored(tt.stored))
		})
	}
}

func Test_contentHash_compareToStored_v1(t *testing.T) {
	currentV1 := BuildingBlockDefinitionVersionContentHash{hashVersion: 1, hashValue: "v1:sameHash"}

	assert.Equal(t, hashSame, currentV1.compareToStored("v1:sameHash"))
	assert.Equal(t, hashDifferent, currentV1.compareToStored("v1:otherHash"))
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

	mutated.BuildingBlockDefinitionRef = &client.BuildingBlockDefinitionRef{
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
