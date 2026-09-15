package clientmock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
)

// A version's secrets are stored encrypted for its runner, so meshStack rejects a secret that carries only
// a hash once the version moves to another runner. The mock has to reject it too, or a unit run passes a
// configuration the acceptance run gets a 400 for.
func TestMockBuildingBlockDefinitionVersionRejectsSecretHashOnRunnerChange(t *testing.T) {
	t.Parallel()

	const (
		runnerUuid      = "11111111-1111-1111-1111-111111111111"
		otherRunnerUuid = "22222222-2222-2222-2222-222222222222"
	)

	versionSpec := func(runner string, sshPrivateKey clientTypes.Secret) client.MeshBuildingBlockDefinitionVersionSpec {
		return client.MeshBuildingBlockDefinitionVersionSpec{
			BuildingBlockDefinitionRef: &client.UuidRef{Uuid: "33333333-3333-3333-3333-333333333333", Kind: client.MeshObjectKind.BuildingBlockDefinition},
			RunnerRef:                  &client.UuidRef{Uuid: runner, Kind: client.MeshObjectKind.BuildingBlockRunner},
			Implementation: client.MeshBuildingBlockDefinitionImplementation{
				Terraform: &client.MeshBuildingBlockDefinitionTerraformImplementation{
					TerraformVersion: "1.9.0",
					RepositoryURL:    "https://github.com/example/building-block.git",
					SSHPrivateKey:    &sshPrivateKey,
				},
			},
		}
	}

	create := func(t *testing.T) (versionClient meshBuildingBlockDefinitionVersionClient, uuid, hash string) {
		t.Helper()
		versionClient = meshBuildingBlockDefinitionVersionClient{Store: NewStore[client.MeshBuildingBlockDefinitionVersion]()}
		created, err := versionClient.Create(context.Background(), "my-workspace",
			versionSpec(runnerUuid, clientTypes.Secret{Plaintext: new("private-key")}))
		require.NoError(t, err)
		return versionClient, created.Metadata.Uuid, *created.Spec.Implementation.Terraform.SSHPrivateKey.Hash
	}

	t.Run("hash kept on the same runner", func(t *testing.T) {
		versionClient, uuid, hash := create(t)

		updated, err := versionClient.Update(context.Background(), uuid, "my-workspace",
			versionSpec(runnerUuid, clientTypes.Secret{Hash: new(hash)}))

		require.NoError(t, err)
		assert.Equal(t, hash, *updated.Spec.Implementation.Terraform.SSHPrivateKey.Hash)
	})

	t.Run("hash rejected on another runner", func(t *testing.T) {
		versionClient, uuid, hash := create(t)

		_, err := versionClient.Update(context.Background(), uuid, "my-workspace",
			versionSpec(otherRunnerUuid, clientTypes.Secret{Hash: new(hash)}))

		assert.ErrorContains(t, err, `secret at "*.Implementation.*Terraform.*SSHPrivateKey" must not contain a hash value`)
	})

	t.Run("plaintext accepted on another runner", func(t *testing.T) {
		versionClient, uuid, _ := create(t)

		updated, err := versionClient.Update(context.Background(), uuid, "my-workspace",
			versionSpec(otherRunnerUuid, clientTypes.Secret{Plaintext: new("rotated-private-key")}))

		require.NoError(t, err)
		assert.Equal(t, "sha256:rotated-private-key", *updated.Spec.Implementation.Terraform.SSHPrivateKey.Hash)
	})
}
