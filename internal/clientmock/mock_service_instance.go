package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshServiceInstanceClient struct {
	Store Store[client.MeshServiceInstance]
}

func (m MeshServiceInstanceClient) Read(_ context.Context, instanceId string) (*client.MeshServiceInstance, error) {
	if serviceInstance, ok := m.Store[instanceId]; ok {
		return serviceInstance, nil
	}
	return nil, nil
}

func (m MeshServiceInstanceClient) List(_ context.Context, filter *client.MeshServiceInstanceFilter) ([]client.MeshServiceInstance, error) {
	var result []client.MeshServiceInstance
	for _, serviceInstance := range m.Store {
		// Apply filters if provided
		if filter != nil {
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
		}
		result = append(result, *serviceInstance)
	}
	return result, nil
}
