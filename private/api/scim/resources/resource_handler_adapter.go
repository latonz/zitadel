package resources

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/zitadel/schema"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/internal/zerrors"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"github.com/zitadel/zitadel/private/api/scim/serrors"
	"net/http"
	"slices"
)

const (
	defaultListCount = 100
	maxListCount     = 100
)

type ResourceHandlerAdapter[T ResourceHolder] struct {
	handler ResourceHandler[T]
}

type ListRequest struct {
	// Count An integer indicating the desired maximum number of query results per page. OPTIONAL.
	Count uint64 `json:"count" schema:"count"`

	// StartIndex An integer indicating the 1-based index of the first query result. Optional.
	StartIndex uint64 `json:"startIndex" schema:"startIndex"`
}

type ListResponse[T any] struct {
	Schemas      []schemas.ScimSchemaType `json:"schemas"`
	ItemsPerPage uint64                   `json:"itemsPerPage"`
	TotalResults uint64                   `json:"totalResults"`
	StartIndex   uint64                   `json:"startIndex"`
	Resources    []T                      `json:"Resources"` // according to the rfc this is the only field in PascalCase...
}

var schemaDecoder = schema.NewDecoder()

func init() {
	schemaDecoder.IgnoreUnknownKeys(true)
}

func NewResourceHandlerAdapter[T ResourceHolder](handler ResourceHandler[T]) *ResourceHandlerAdapter[T] {
	return &ResourceHandlerAdapter[T]{
		handler,
	}
}

func newListResponse[T any](totalResultCount uint64, q query.SearchRequest, resources []T) *ListResponse[T] {
	return &ListResponse[T]{
		Schemas:      []schemas.ScimSchemaType{schemas.IdListResponse},
		ItemsPerPage: q.Limit,
		TotalResults: totalResultCount,
		StartIndex:   q.Offset + 1, // start index is 1 based
		Resources:    resources,
	}
}

func (adapter *ResourceHandlerAdapter[T]) Create(r *http.Request) (T, error) {
	entity, err := adapter.readEntityFromBody(r)
	if err != nil {
		return entity, err
	}

	return adapter.handler.Create(r.Context(), entity)
}

func (adapter *ResourceHandlerAdapter[T]) List(r *http.Request) (*ListResponse[T], error) {
	request := &ListRequest{
		Count:      defaultListCount,
		StartIndex: 1,
	}

	switch r.Method {
	case http.MethodGet:
		if err := r.ParseForm(); err != nil {
			return nil, zerrors.ThrowInvalidArgument(nil, "SCIM-uliform", "Could not deserialize form: "+err.Error())
		}

		if err := schemaDecoder.Decode(request, r.Form); err != nil {
			return nil, zerrors.ThrowInvalidArgument(nil, "SCIM-ullform", "Could not decode form: "+err.Error())
		}
		break
	case http.MethodPost:
		if err := json.NewDecoder(r.Body).Decode(request); err != nil {
			return nil, zerrors.ThrowInvalidArgument(nil, "SCIM-ulljson", "Could not decode json: "+err.Error())
		}
		break
	}

	// according to the spec values < 1 are treated as 1
	if request.StartIndex < 1 {
		request.StartIndex = 1
	}

	// according to the spec values < 0 are treated as 0
	if request.Count < 0 {
		request.Count = 0
	} else if request.Count > maxListCount {
		return nil, zerrors.ThrowInvalidArgumentf(nil, "SCIM-ucr", "Limit count exceeded, set a count <= %v", maxListCount)
	}

	return adapter.handler.List(r.Context(), request)
}

func (adapter *ResourceHandlerAdapter[T]) Get(r *http.Request) (T, error) {
	id := mux.Vars(r)["id"]
	return adapter.handler.Get(r.Context(), id)
}

func (adapter *ResourceHandlerAdapter[T]) Delete(r *http.Request) error {
	id := mux.Vars(r)["id"]
	return adapter.handler.Delete(r.Context(), id)
}

func (adapter *ResourceHandlerAdapter[T]) readEntityFromBody(r *http.Request) (T, error) {
	entity := adapter.handler.NewResource()
	err := json.NewDecoder(r.Body).Decode(entity)
	if err != nil {
		return entity, serrors.ThrowInvalidSyntax(zerrors.ThrowInvalidArgumentf(nil, "SCIM-ucrjson", "Could not deserialize json: %v", err.Error()))
	}

	resource := entity.GetResource()
	if resource == nil {
		return entity, serrors.ThrowInvalidSyntax(zerrors.ThrowInvalidArgument(nil, "SCIM-xxrjson", "Could not get resource"))
	}

	if !slices.Contains(resource.Schemas, adapter.handler.SchemaType()) {
		return entity, serrors.ThrowInvalidSyntax(zerrors.ThrowInvalidArgumentf(nil, "SCIM-xxrschema", "Expected schema %v is not provided", adapter.handler.SchemaType()))
	}

	return entity, nil
}
