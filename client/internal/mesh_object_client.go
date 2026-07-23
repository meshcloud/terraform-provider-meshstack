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
	"unicode"
)

// MeshObjectClient provides typed CRUD operations for meshStack API objects.
// It embeds [HttpClient] and adds meshObject-specific functionality including automatic
// MIME type handling and pagination.
// Also handles authentication in doAuthorizedRequest using the ApiKey/ApiSecret values,
// which are embedded in HttpClient for convenient construction with NewMeshObjectClient.
type MeshObjectClient[M any] struct {
	HttpClient
	Kind       string
	ApiVersion string
	ApiUrl     *url.URL
}

// NewMeshObjectClient creates a new [MeshObjectClient] for a specific meshObject type with automatic URL path inference.
// The meshObject kind is inferred from type M. T
// The API URL is constructed from explicitApiPathElems if provided,
// otherwise the pluralized and lowercased kind is used as a single element.
func NewMeshObjectClient[M any](ctx context.Context, httpClient HttpClient, apiVersion string, explicitApiPathElems ...string) MeshObjectClient[M] {
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
// as the meshObject API: MeshWorkspace → "meshWorkspace", MeshBuildingBlockV2 → "meshBuildingBlock".
// Version suffixes (V\d+) are stripped.
// Tested when client.Kind is statically initialized.
func InferKind[M any]() string {
	typeName := reflect.TypeFor[M]().Name()

	runes := []rune(typeName)
	runes[0] = unicode.ToLower(runes[0])
	kind := string(runes)

	return versionSuffixRe.ReplaceAllString(kind, "")
}

var pluralExceptions = map[string]string{
	// Add exceptions here as needed, e.g. "meshPolicy": "meshPolicies"
}

func pluralizeKind(kind string) string {
	if plural, ok := pluralExceptions[kind]; ok {
		return plural
	}
	return kind + "s"
}

func (c MeshObjectClient[M]) MeshObjectMimeType() string {
	return fmt.Sprintf("application/vnd.meshcloud.api.%s.%s.hal+json", c.Kind, c.ApiVersion)
}

// Get retrieves a meshObject by ID. Returns nil if not found.
func (c MeshObjectClient[M]) Get(ctx context.Context, id string) (resp *M, err error) {
	resp, err = DoAuthorizedRequest[*M](ctx, c.HttpClient, http.MethodGet, c.ApiUrl.JoinPath(id), WithAccept(c.MeshObjectMimeType()))
	if httpErr, ok := errors.AsType[HttpError](err); ok && httpErr.IsNotFound() {
		return nil, nil
	}
	return
}

// Post creates a new meshObject with the given payload.
// Automatically injects apiVersion and kind into the JSON payload.
func (c MeshObjectClient[M]) Post(ctx context.Context, payload any, options ...RequestOption) (*M, error) {
	return DoAuthorizedRequest[*M](
		ctx,
		c.HttpClient,
		http.MethodPost,
		c.ApiUrl,
		append(options, c.withMeshObjectPayload(payload))...,
	)
}

// Put updates an existing meshObject by ID with the given payload.
// Automatically injects apiVersion and kind into the JSON payload.
func (c MeshObjectClient[M]) Put(ctx context.Context, id string, payload any) (*M, error) {
	return DoAuthorizedRequest[*M](ctx, c.HttpClient, http.MethodPut, c.ApiUrl.JoinPath(id), c.withMeshObjectPayload(payload))
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

	return withPayload(m, c.MeshObjectMimeType())
}

// Delete removes a meshObject by ID.
func (c MeshObjectClient[M]) Delete(ctx context.Context, id string, options ...RequestOption) (err error) {
	_, err = DoAuthorizedRequest[any](ctx, c.HttpClient, http.MethodDelete, c.ApiUrl.JoinPath(id), append(options, WithAccept(c.MeshObjectMimeType()))...)
	return
}

// List retrieves all meshObjects with automatic pagination handling.
// Accepts optional [RequestOption] parameters for filtering and querying.
func (c MeshObjectClient[M]) List(ctx context.Context, options ...RequestOption) ([]M, error) {
	var result []M
	embeddedKey := pluralizeKind(c.Kind)
	pageNumber := 0

	for {
		type paginatedResponse struct {
			Embedded map[string][]M `json:"_embedded"`
			Page     struct {
				TotalPages int `json:"totalPages"`
				Number     int `json:"number"`
			} `json:"page"`
		}
		response, err := DoAuthorizedRequest[paginatedResponse](ctx, c.HttpClient, http.MethodGet, c.ApiUrl, append(options,
			WithAccept(c.MeshObjectMimeType()),
			WithUrlQuery(map[string]any{"page": pageNumber}),
		)...)
		if err != nil {
			return result, fmt.Errorf("error getting page %d: %w", pageNumber, err)
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
