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
	BuildingBlock                  MeshBuildingBlockClient
	BuildingBlockDefinition        MeshBuildingBlockDefinitionClient
	BuildingBlockDefinitionVersion MeshBuildingBlockDefinitionVersionClient
	BuildingBlockV2                MeshBuildingBlockV2Client
	BuildingBlockV3                MeshBuildingBlockV3Client
	Integration                    MeshIntegrationClient
	LandingZone                    MeshLandingZoneClient
	Location                       MeshLocationClient
	PaymentMethod                  MeshPaymentMethodClient
	Platform                       MeshPlatformClient
	PlatformType                   MeshPlatformTypeClient
	Project                        MeshProjectClient
	ProjectGroupBinding            MeshProjectGroupBindingClient
	ProjectUserBinding             MeshProjectUserBindingClient
	ServiceInstance                MeshServiceInstanceClient
	TagDefinition                  MeshTagDefinitionClient
	Tenant                         MeshTenantClient
	TenantV4                       MeshTenantV4Client
	Workspace                      MeshWorkspaceClient
	WorkspaceGroupBinding          MeshWorkspaceGroupBindingClient
	WorkspaceUserBinding           MeshWorkspaceUserBindingClient
}

func (c Client) AsClient() client.Client {
	return client.Client{
		BuildingBlock:                  c.BuildingBlock,
		BuildingBlockDefinition:        c.BuildingBlockDefinition,
		BuildingBlockDefinitionVersion: c.BuildingBlockDefinitionVersion,
		BuildingBlockV2:                c.BuildingBlockV2,
		BuildingBlockV3:                c.BuildingBlockV3,
		Integration:                    c.Integration,
		LandingZone:                    c.LandingZone,
		Location:                       c.Location,
		PaymentMethod:                  c.PaymentMethod,
		Platform:                       c.Platform,
		PlatformType:                   c.PlatformType,
		Project:                        c.Project,
		ProjectGroupBinding:            c.ProjectGroupBinding,
		ProjectUserBinding:             c.ProjectUserBinding,
		ServiceInstance:                c.ServiceInstance,
		TagDefinition:                  c.TagDefinition,
		Tenant:                         c.Tenant,
		TenantV4:                       c.TenantV4,
		Workspace:                      c.Workspace,
		WorkspaceGroupBinding:          c.WorkspaceGroupBinding,
		WorkspaceUserBinding:           c.WorkspaceUserBinding,
	}
}

func NewMock() Client {
	bbdVersionStore := make(Store[client.MeshBuildingBlockDefinitionVersion])
	return Client{
		BuildingBlock:                  MeshBuildingBlockClient{make(Store[client.MeshBuildingBlock])},
		BuildingBlockDefinition:        MeshBuildingBlockDefinitionClient{make(Store[client.MeshBuildingBlockDefinition]), bbdVersionStore},
		BuildingBlockDefinitionVersion: MeshBuildingBlockDefinitionVersionClient{bbdVersionStore},
		BuildingBlockV2:                MeshBuildingBlockV2Client{make(Store[client.MeshBuildingBlockV2])},
		BuildingBlockV3:                MeshBuildingBlockV3Client{make(Store[client.MeshBuildingBlockV3])},
		Integration:                    MeshIntegrationClient{make(Store[client.MeshIntegration])},
		LandingZone:                    MeshLandingZoneClient{make(Store[client.MeshLandingZone])},
		Location:                       MeshLocationClient{make(Store[client.MeshLocation])},
		PaymentMethod:                  MeshPaymentMethodClient{make(Store[client.MeshPaymentMethod])},
		Platform:                       MeshPlatformClient{make(Store[client.MeshPlatform])},
		PlatformType:                   MeshPlatformTypeClient{make(Store[client.MeshPlatformType])},
		Project:                        MeshProjectClient{make(Store[client.MeshProject])},
		ProjectGroupBinding:            MeshProjectGroupBindingClient{make(Store[client.MeshProjectGroupBinding])},
		ProjectUserBinding:             MeshProjectUserBindingClient{make(Store[client.MeshProjectUserBinding])},
		ServiceInstance:                MeshServiceInstanceClient{make(Store[client.MeshServiceInstance])},
		TagDefinition:                  MeshTagDefinitionClient{make(Store[client.MeshTagDefinition])},
		Tenant:                         MeshTenantClient{make(Store[client.MeshTenant])},
		TenantV4:                       MeshTenantV4Client{make(Store[client.MeshTenantV4])},
		Workspace:                      MeshWorkspaceClient{make(Store[client.MeshWorkspace])},
		WorkspaceGroupBinding:          MeshWorkspaceGroupBindingClient{make(Store[client.MeshWorkspaceGroupBinding])},
		WorkspaceUserBinding:           MeshWorkspaceUserBindingClient{make(Store[client.MeshWorkspaceUserBinding])},
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
