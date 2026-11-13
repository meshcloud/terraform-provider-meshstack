package clientmock

import (
	"context"
	"iter"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshServiceInstanceClient struct {
	client.MeshServiceInstanceClient

	Store *Store[client.MeshServiceInstance]
}

func (m MeshServiceInstanceClient) Read(_ context.Context, instanceId string) (*client.MeshServiceInstance, error) {
	if serviceInstance, ok := m.Store.Get(instanceId); ok {
		return serviceInstance, nil
	}
	return nil, nil
}

func (m MeshServiceInstanceClient) List(_ context.Context, filter client.MeshServiceInstanceFilter, _ ...client.ListOption) iter.Seq2[client.MeshServiceInstance, error] {
	var result []client.MeshServiceInstance
	for _, serviceInstance := range m.Store.Values() {
		if filter.WorkspaceIdentifier != nil && serviceInstance.Metadata.OwnedByWorkspace != *filter.WorkspaceIdentifier {
			continue
		}
		if filter.ProjectIdentifier != nil && serviceInstance.Metadata.OwnedByProject != *filter.ProjectIdentifier {
			continue
		}
		if filter.MarketplaceIdentifier != nil && serviceInstance.Metadata.MarketplaceIdentifier != *filter.MarketplaceIdentifier {
			continue
		}
		if filter.ServiceIdentifier != nil && serviceInstance.Spec.ServiceId != *filter.ServiceIdentifier {
			continue
		}
		if filter.PlanIdentifier != nil && serviceInstance.Spec.PlanId != *filter.PlanIdentifier {
			continue
		}
		result = append(result, *serviceInstance)
	}
	return listOf(result, nil)
}
