package internal

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"
)

// MeshObjectClient provides typed CRUD operations for meshStack API objects.
// It embeds [HttpClient] and adds meshObject-specific functionality including automatic
// MIME type handling and pagination.
// Also handles authentication in doAuthorizedRequest using the ApiKey/ApiSecret values,
// which are embedded in HttpClient for convenient construction with NewMeshObjectClient.
type MeshObjectClient[M any] struct {
	*HttpClient
	Name       string
	ApiVersion string
	ApiUrl     *url.URL
}

// NewMeshObjectClient creates a new [MeshObjectClient] for a specific meshObject type with automatic URL path inference.
// The meshObject name is inferred from type M, and the API URL is constructed from explicitApiPaths or the pluralized type name.
func NewMeshObjectClient[M any](httpClient *HttpClient, apiVersion string, explicitApiPaths ...string) MeshObjectClient[M] {
	name := inferMeshObjectName[M]()

	if len(explicitApiPaths) == 0 {
		explicitApiPaths = []string{strings.ToLower(pluralizeName(name))}
	}
	explicitApiPaths = slices.Insert(explicitApiPaths, 0, "/api/meshobjects")
	apiUrl := httpClient.RootUrl.JoinPath(explicitApiPaths...)
	log.Printf("Using API at '%s' for meshObject '%s', version '%s'", apiUrl, name, apiVersion)
	return MeshObjectClient[M]{httpClient, name, apiVersion, apiUrl}
}

func inferMeshObjectName[M any]() string {
	var zero M
	typeName := reflect.TypeOf(zero).Name()
	return lowercaseFirst(typeName)
}

func lowercaseFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func pluralizeName(name string) string {
	if strings.HasSuffix(name, "y") {
		// this is ok, as we don't have meshObjects ending in 'y' yet, so take this shortcut
		panic(fmt.Sprintf("Correctly pluralizing '%s' is not supported yet", name))
	}
	return fmt.Sprintf("%ss", name)
}

func (c MeshObjectClient[M]) meshObjectMimeType() string {
	return fmt.Sprintf("application/vnd.meshcloud.api.%s.%s.hal+json", c.Name, c.ApiVersion)
}

// Get retrieves a meshObject by ID. Returns nil if not found.
func (c MeshObjectClient[M]) Get(id string) (*M, error) {
	body, err := c.doAuthorizedRequest(http.MethodGet, c.ApiUrl.JoinPath(id), withAccept(c.meshObjectMimeType()))
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	return unmarshalBody[M](body, err)
}

// Post creates a new meshObject with the given payload.
func (c MeshObjectClient[M]) Post(payload any) (*M, error) {
	return unmarshalBody[M](c.doAuthorizedRequest(http.MethodPost, c.ApiUrl, withPayload(payload, c.meshObjectMimeType())))
}

// Put updates an existing meshObject by ID with the given payload.
func (c MeshObjectClient[M]) Put(id string, payload any) (*M, error) {
	return unmarshalBody[M](c.doAuthorizedRequest(http.MethodPut, c.ApiUrl.JoinPath(id), withPayload(payload, c.meshObjectMimeType())))
}

// Delete removes a meshObject by ID.
func (c MeshObjectClient[M]) Delete(id string) (err error) {
	_, err = c.doAuthorizedRequest(http.MethodDelete, c.ApiUrl.JoinPath(id), withAccept(c.meshObjectMimeType()))
	return
}

// List retrieves all meshObjects with automatic pagination handling.
// Accepts optional [RequestOption] parameters for filtering and querying.
func (c MeshObjectClient[M]) List(options ...RequestOption) ([]M, error) {
	var result []M
	embeddedKey := pluralizeName(c.Name)
	pageNumber := 0

	for {
		body, err := c.doAuthorizedRequest(http.MethodGet, c.ApiUrl, append(options,
			withAccept(c.meshObjectMimeType()),
			WithUrlQuery("page", pageNumber),
		)...)
		if err != nil {
			return result, fmt.Errorf("cannot fetch page %d: %w", pageNumber, err)
		}
		type paginatedResponse struct {
			Embedded map[string][]M `json:"_embedded"`
			Page     struct {
				TotalPages int `json:"totalPages"`
				Number     int `json:"number"`
			} `json:"page"`
		}
		response, err := unmarshalBody[paginatedResponse](body, err)
		if err != nil {
			return result, fmt.Errorf("cannot unmarshal paginated response, page %d: %w", pageNumber, err)
		} else if items, ok := response.Embedded[embeddedKey]; !ok {
			return result, fmt.Errorf("embedded key %s not found in paginated response", embeddedKey)
		} else {
			result = append(result, items...)
		}
		if response.Page.Number >= response.Page.TotalPages-1 {
			return result, nil
		}
		pageNumber++
	}
}

func (c MeshObjectClient[M]) doAuthorizedRequest(method string, url *url.URL, options ...RequestOption) ([]byte, error) {
	if err := c.ensureAuthorization(); err != nil {
		return nil, err
	}
	return c.doRequest(method, url, append(options,
		appendRequestModifier(func(req *http.Request) {
			log.Println(req)
		}),
		withHeader("Authorization", c.Authorization),
	)...)
}

func (c MeshObjectClient[M]) ensureAuthorization() error {
	if c.Authorization != "" && time.Until(c.AuthorizationExpiresAt) > 30*time.Second {
		return nil
	}

	loginApiUrl := c.RootUrl.JoinPath("/api/login")

	type loginRequest struct {
		ClientId     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}

	type loginResponse struct {
		Token     string `json:"access_token"`
		ExpireSec int    `json:"expires_in"`
	}

	loginResult, err := unmarshalBody[loginResponse](c.doRequest("POST", loginApiUrl,
		withPayload(loginRequest{ClientId: c.ApiKey, ClientSecret: c.ApiSecret}, "application/json")),
	)
	if err != nil {
		return fmt.Errorf("login request to %s with API Key '%s' failed: %w", loginApiUrl, c.ApiKey, err)
	}

	c.Authorization = fmt.Sprintf("Bearer %s", loginResult.Token)
	c.AuthorizationExpiresAt = time.Now().Add(time.Duration(loginResult.ExpireSec) * time.Second)
	return nil
}
