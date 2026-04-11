package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
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
	Kind       string
	ApiVersion string
	ApiUrl     *url.URL
}

// NewMeshObjectClient creates a new [MeshObjectClient] for a specific meshObject type with automatic URL path inference.
// The meshObject kind is inferred from type M. T
// The API URL is constructed from explicitApiPathElems if provided,
// otherwise the pluralized and lowercased kind is used as a single element.
func NewMeshObjectClient[M any](ctx context.Context, httpClient *HttpClient, apiVersion string, explicitApiPathElems ...string) MeshObjectClient[M] {
	kind := InferKind[M]()

	if len(explicitApiPathElems) == 0 {
		explicitApiPathElems = []string{strings.ToLower(pluralizeKind(kind))}
	}
	explicitApiPathElems = slices.Insert(explicitApiPathElems, 0, "/api/meshobjects")
	apiUrl := httpClient.RootUrl.JoinPath(explicitApiPathElems...)
	Log.Info(ctx, fmt.Sprintf("initialized %s client", reflect.TypeFor[M]().Name()), "url", apiUrl.String(), "kind", kind, "version", apiVersion)
	return MeshObjectClient[M]{httpClient, kind, apiVersion, apiUrl}
}

var versionSuffixRe = regexp.MustCompile(`V\d+$`)

// InferKind infers the meshObject kind from a struct type name using the same convention
// as the meshObject API: MeshWorkspace → "meshWorkspace", MeshTenantV4 → "meshTenant".
// Version suffixes (V\d+) are stripped.
// Tested when client.Kind is statically initialized.
func InferKind[M any]() string {
	typeName := reflect.TypeFor[M]().Name()

	runes := []rune(typeName)
	runes[0] = unicode.ToLower(runes[0])
	kind := string(runes)

	return versionSuffixRe.ReplaceAllString(kind, "")
}

func pluralizeKind(kind string) string {
	if strings.HasSuffix(kind, "y") {
		// this is ok, as we don't have meshObjects ending in 'y' yet, so take this shortcut
		panic(fmt.Sprintf("Correctly pluralizing meshObject kind '%s' is not supported yet", kind))
	}
	return fmt.Sprintf("%ss", kind)
}

func (c MeshObjectClient[M]) meshObjectMimeType() string {
	return fmt.Sprintf("application/vnd.meshcloud.api.%s.%s.hal+json", c.Kind, c.ApiVersion)
}

// Get retrieves a meshObject by ID. Returns nil if not found.
func (c MeshObjectClient[M]) Get(ctx context.Context, id string) (*M, error) {
	body, err := c.doAuthorizedRequest(ctx, http.MethodGet, c.ApiUrl.JoinPath(id), withAccept(c.meshObjectMimeType()))
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	return unmarshalBody[M](body, err)
}

// Post creates a new meshObject with the given payload.
// Automatically injects apiVersion and kind into the JSON payload.
func (c MeshObjectClient[M]) Post(ctx context.Context, payload any, options ...RequestOption) (*M, error) {
	return unmarshalBody[M](c.doAuthorizedRequest(
		ctx,
		http.MethodPost,
		c.ApiUrl,
		append(options, c.withMeshObjectPayload(payload))...,
	))
}

// Put updates an existing meshObject by ID with the given payload.
// Automatically injects apiVersion and kind into the JSON payload.
func (c MeshObjectClient[M]) Put(ctx context.Context, id string, payload any) (*M, error) {
	return unmarshalBody[M](c.doAuthorizedRequest(ctx, http.MethodPut, c.ApiUrl.JoinPath(id), c.withMeshObjectPayload(payload)))
}

// withMeshObjectPayload returns a RequestOption that sets the payload with apiVersion and kind injected,
// using the meshObject MIME type for content negotiation.
// Panics on marshal errors which indicates a programming error (payload is always a well-typed struct).
//
// The double marshal/unmarshal round-trip converts the typed struct to a map[string]any so we can
// inject the top-level apiVersion and kind fields without coupling the struct type to those fields.
func (c MeshObjectClient[M]) withMeshObjectPayload(payload any) RequestOption {
	intermediate, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal %T: %v", payload, err))
	}

	var m map[string]any
	if err := json.Unmarshal(intermediate, &m); err != nil {
		panic(fmt.Sprintf("failed to unmarshal %T to map: %v", payload, err))
	}

	m["apiVersion"] = c.ApiVersion
	m["kind"] = c.Kind

	return withPayload(m, c.meshObjectMimeType())
}

// Delete removes a meshObject by ID.
func (c MeshObjectClient[M]) Delete(ctx context.Context, id string, options ...RequestOption) (err error) {
	_, err = c.doAuthorizedRequest(ctx, http.MethodDelete, c.ApiUrl.JoinPath(id), append(options, withAccept(c.meshObjectMimeType()))...)
	return
}

// List retrieves all meshObjects with automatic pagination handling.
// Accepts optional [RequestOption] parameters for filtering and querying.
func (c MeshObjectClient[M]) List(ctx context.Context, options ...RequestOption) ([]M, error) {
	var result []M
	embeddedKey := pluralizeKind(c.Kind)
	pageNumber := 0

	for {
		body, err := c.doAuthorizedRequest(ctx, http.MethodGet, c.ApiUrl, append(options,
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

func (c MeshObjectClient[M]) doAuthorizedRequest(ctx context.Context, method string, url *url.URL, options ...RequestOption) ([]byte, error) {
	if err := c.ensureAuthorization(ctx); err != nil {
		return nil, err
	}
	return c.doRequest(ctx, method, url, append(options, withHeader("Authorization", c.Authorization))...)
}

func (c MeshObjectClient[M]) ensureAuthorization(ctx context.Context) error {
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

	loginResult, err := unmarshalBody[loginResponse](c.doRequest(ctx, "POST", loginApiUrl,
		withPayload(loginRequest{ClientId: c.ApiKey, ClientSecret: c.ApiSecret}, "application/json")),
	)
	if err != nil {
		return fmt.Errorf("login request to %s with API Key '%s' failed: %w", loginApiUrl, c.ApiKey, err)
	}

	c.Authorization = fmt.Sprintf("Bearer %s", loginResult.Token)
	c.AuthorizationExpiresAt = time.Now().Add(time.Duration(loginResult.ExpireSec) * time.Second)
	return nil
}
