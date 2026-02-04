package clientmock

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
	reflectwalk "github.com/meshcloud/terraform-provider-meshstack/internal/util/reflect"
)

type Client struct {
	BuildingBlockDefinition        MeshBuildingBlockDefinitionClient
	BuildingBlockDefinitionVersion MeshBuildingBlockDefinitionVersionClient
	TagDefinition                  MeshTagDefinitionClient
	Platform                       MeshPlatformClient
	Location                       MeshLocationClient
}

func (c Client) AsClient() client.Client {
	return client.Client{
		BuildingBlockDefinition:        c.BuildingBlockDefinition,
		BuildingBlockDefinitionVersion: c.BuildingBlockDefinitionVersion,
		TagDefinition:                  c.TagDefinition,
		Platform:                       c.Platform,
		Location:                       c.Location,
	}
}

func NewMock() Client {
	bbdVersionStore := make(Store[client.MeshBuildingBlockDefinitionVersion])
	return Client{
		BuildingBlockDefinition:        MeshBuildingBlockDefinitionClient{make(Store[client.MeshBuildingBlockDefinition]), bbdVersionStore},
		BuildingBlockDefinitionVersion: MeshBuildingBlockDefinitionVersionClient{bbdVersionStore},
		TagDefinition:                  MeshTagDefinitionClient{make(Store[client.MeshTagDefinition])},
		Platform:                       MeshPlatformClient{make(Store[client.MeshPlatform])},
		Location:                       MeshLocationClient{make(Store[client.MeshLocation])},
	}
}

type Store[M any] map[string]*M

func (s Store[M]) SortedKeys() []string {
	return slices.SortedFunc(maps.Keys(s), strings.Compare)
}

// backendSecretBehavior mocks backend behavior in the sense that it consumes the plaintext secret and returns a hash of the secret only.
func backendSecretBehavior[T any](allowSecretHashOnlyOnCreate bool, dto, existingDto *T) {
	handleSecret := func(secret, existingSecret *clientTypes.Secret) {
		if secret != nil && secret.Plaintext != nil && *secret.Plaintext != "" {
			secret.Hash = ptr.To(fmt.Sprintf("sha256:%s", *secret.Plaintext))
			secret.Plaintext = nil
		} else if existingSecret != nil {
			switch {
			case existingSecret.Plaintext != nil:
				panic("found plaintext in existing secret, only hash should be known")
			case existingSecret.Hash == nil:
				panic("no hash found in existing secret")
			case secret == nil || secret.Hash == nil:
				panic("existing secret present, but no known hash provided for check")
			case *existingSecret.Hash != *secret.Hash:
				panic("mismatching hash for existing secret")
			}
		} else if !allowSecretHashOnlyOnCreate || secret == nil || secret.Hash == nil || *secret.Hash == "" {
			panic("inconsistent create or update of secret in mock client (empty plaintext provided?)")
		}
	}

	secretType := reflect.TypeFor[clientTypes.Secret]()
	secretOrAnyType := reflect.TypeFor[clientTypes.SecretOrAny]()
	if err := reflectwalk.Walk(reflect.ValueOf(dto), func(path reflectwalk.WalkPath, v reflect.Value) error {
		switch {
		case !v.CanAddr():
			return nil
		case v.Type().ConvertibleTo(secretType):
			secret, _ := v.Addr().Interface().(*clientTypes.Secret)
			var existingSecret *clientTypes.Secret
			if existingDto != nil {
				if vExisting, err := path.TryTraverse(existingDto); err == nil {
					existingSecret, _ = vExisting.Addr().Interface().(*clientTypes.Secret)
				}
			}
			handleSecret(secret, existingSecret)
			return path.Stop()
		case v.Type().ConvertibleTo(secretOrAnyType):
			secretOrAny, _ := v.Addr().Interface().(*clientTypes.SecretOrAny)
			if secretOrAny.HasX() {
				secret := &secretOrAny.X
				var existingSecret *clientTypes.Secret
				if existingDto != nil {
					if vExisting, err := path.TryTraverse(existingDto); err == nil {
						existingSecretOrAny, _ := vExisting.Addr().Interface().(*clientTypes.SecretOrAny)
						if existingSecretOrAny.HasX() {
							existingSecret = &existingSecretOrAny.X
						}
					}
				}
				handleSecret(secret, existingSecret)
			}
			return path.Stop()
		}
		return nil
	}); err != nil {
		panic(err)
	}
}
