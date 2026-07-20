package clientmock

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	reflectwalk "github.com/meshcloud/terraform-provider-meshstack/internal/util/reflect"
)

type Client struct {
	ApiKey                         MeshApiKeyClient
	BuildingBlock                  meshBuildingBlockClient
	BuildingBlockRun               MeshBuildingBlockRunClient
	BuildingBlockDefinition        meshBuildingBlockDefinitionClient
	BuildingBlockDefinitionVersion meshBuildingBlockDefinitionVersionClient
	BuildingBlockRunner            MeshBuildingBlockRunnerClient
	BuildingBlockV2                MeshBuildingBlockV2Client
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
	Workspace                      MeshWorkspaceClient
	WorkspaceGroupBinding          MeshWorkspaceGroupBindingClient
	WorkspaceUserBinding           MeshWorkspaceUserBindingClient
}

func (c *Client) AsClient() client.Client {
	return client.Client{
		ApiKey:                         c.ApiKey,
		BuildingBlock:                  c.BuildingBlock,
		BuildingBlockRun:               c.BuildingBlockRun,
		BuildingBlockDefinition:        c.BuildingBlockDefinition,
		BuildingBlockDefinitionVersion: c.BuildingBlockDefinitionVersion,
		BuildingBlockRunner:            c.BuildingBlockRunner,
		BuildingBlockV2:                c.BuildingBlockV2,
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
		Workspace:                      c.Workspace,
		WorkspaceGroupBinding:          c.WorkspaceGroupBinding,
		WorkspaceUserBinding:           c.WorkspaceUserBinding,
	}
}

func NewMock() Client {
	bbdVersionStore := NewStore[client.MeshBuildingBlockDefinitionVersion]()
	buildingBlockRunStore := NewStore[client.MeshBuildingBlockRun]()
	buildingBlockRunLogStore := NewStore[client.MeshBuildingBlockRunLogs]()
	// The v1 and v2/v3 building block clients share one store: on the real backend a building block
	// is a single entity exposed by both APIs (same uuid), so a v1-created block is readable via
	// v2/v3. The v1 client maps to/from the v2 representation (see meshBuildingBlockClient), which is
	// what lets the v1->v3 `moved` migration refresh-Read find the block. The tenant store is shared
	// so the v1 client can resolve tenant_identifier <-> tenant target_ref uuid.
	buildingBlockStore := NewStore[client.MeshBuildingBlockV2]()
	tenantStore := NewStore[client.MeshTenant]()
	return Client{
		ApiKey:                         MeshApiKeyClient{Store: NewStore[client.MeshApiKey]()},
		BuildingBlock:                  meshBuildingBlockClient{Store: buildingBlockStore, BbdVersionStore: bbdVersionStore, TenantStore: tenantStore},
		BuildingBlockRun:               MeshBuildingBlockRunClient{Store: buildingBlockRunStore, LogStore: buildingBlockRunLogStore},
		BuildingBlockDefinition:        meshBuildingBlockDefinitionClient{Store: NewStore[client.MeshBuildingBlockDefinition](), StoreVersion: bbdVersionStore},
		BuildingBlockDefinitionVersion: meshBuildingBlockDefinitionVersionClient{Store: bbdVersionStore},
		BuildingBlockRunner:            MeshBuildingBlockRunnerClient{Store: NewStore[client.MeshBuildingBlockRunner]()},
		BuildingBlockV2:                MeshBuildingBlockV2Client{Store: buildingBlockStore, BbdVersionStore: bbdVersionStore},
		Integration:                    MeshIntegrationClient{Store: NewStore[client.MeshIntegration]()},
		LandingZone:                    MeshLandingZoneClient{Store: NewStore[client.MeshLandingZone]()},
		Location:                       MeshLocationClient{Store: NewStore[client.MeshLocation]()},
		PaymentMethod:                  MeshPaymentMethodClient{Store: NewStore[client.MeshPaymentMethod]()},
		Platform:                       MeshPlatformClient{Store: NewStore[client.MeshPlatform]()},
		PlatformType:                   MeshPlatformTypeClient{Store: NewStore[client.MeshPlatformType]()},
		Project:                        MeshProjectClient{Store: NewStore[client.MeshProject]()},
		ProjectGroupBinding:            MeshProjectGroupBindingClient{Store: NewStore[client.MeshProjectGroupBinding]()},
		ProjectUserBinding:             MeshProjectUserBindingClient{Store: NewStore[client.MeshProjectUserBinding]()},
		ServiceInstance:                MeshServiceInstanceClient{Store: NewStore[client.MeshServiceInstance]()},
		TagDefinition:                  MeshTagDefinitionClient{Store: NewStore[client.MeshTagDefinition]()},
		Tenant:                         MeshTenantClient{Store: tenantStore},
		Workspace:                      MeshWorkspaceClient{Store: NewStore[client.MeshWorkspace]()},
		WorkspaceGroupBinding:          MeshWorkspaceGroupBindingClient{Store: NewStore[client.MeshWorkspaceGroupBinding]()},
		WorkspaceUserBinding:           MeshWorkspaceUserBindingClient{Store: NewStore[client.MeshWorkspaceUserBinding]()},
	}
}

// Store is a concurrency-safe key-value store for mock client data.
// Always use NewStore to create instances; pass *Store to mock client structs.
type Store[M any] struct {
	mu   sync.RWMutex
	data map[string]*M
}

func NewStore[M any]() *Store[M] {
	return &Store[M]{data: make(map[string]*M)}
}

func (s *Store[M]) Get(key string) (*M, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

func (s *Store[M]) Set(key string, val *M) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

func (s *Store[M]) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// Values returns a snapshot of all stored values.
func (s *Store[M]) Values() []*M {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*M, 0, len(s.data))
	for _, v := range s.data {
		result = append(result, v)
	}
	return result
}

func (s *Store[M]) SortedKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.SortedFunc(maps.Keys(s.data), strings.Compare)
}

// backendSecretBehavior mocks backend behavior in the sense that it consumes the plaintext secret and returns a hash of the secret only.
func backendSecretBehavior[T any](allowSecretHashOnlyOnCreate bool, dto, existingDto *T) {
	handleSecret := func(secret, existingSecret *clientTypes.Secret) {
		if secret != nil && secret.Plaintext != nil && *secret.Plaintext != "" {
			secret.Hash = new(fmt.Sprintf("sha256:%s", *secret.Plaintext))
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
