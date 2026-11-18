package clientmock

import (
	"context"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshUserClient struct {
	Store *Store[client.MeshUser]
}

func (m MeshUserClient) Read(_ context.Context, username string) (*client.MeshUser, error) {
	user, _ := m.Store.Get(username)
	return user, nil
}

func (m MeshUserClient) List(_ context.Context, filter client.MeshUserListFilter) ([]client.MeshUser, error) {
	var result []client.MeshUser
	for _, username := range m.Store.SortedKeys() {
		user, _ := m.Store.Get(username)
		if matchesUserFilter(*user, filter) {
			result = append(result, *user)
		}
	}
	return result, nil
}

func matchesUserFilter(user client.MeshUser, filter client.MeshUserListFilter) bool {
	return (filter.Uuid == "" || user.Metadata.Uuid == filter.Uuid) &&
		(filter.Username == "" || user.Metadata.Name == filter.Username) &&
		(filter.Email == "" || user.Spec.Email == filter.Email) &&
		(filter.Euid == "" || user.Spec.Euid != nil && *user.Spec.Euid == filter.Euid)
}

const SeededUsername = "customer@meshcloud.io"

func seededUser() *client.MeshUser {
	return &client.MeshUser{
		Metadata: client.MeshUserMetadata{Name: SeededUsername, Uuid: "5f0c1a64-8a3e-4f43-9d2a-0d3b7f1c2e9a"},
		Spec:     client.MeshUserSpec{Email: SeededUsername, FirstName: "Customer", LastName: "Meshcloud", Tags: map[string][]string{}},
		Status:   client.MeshUserStatus{Lifecycle: client.MeshUserLifecycle{State: "ACTIVE"}},
	}
}
