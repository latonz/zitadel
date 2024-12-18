package resources

import (
	"context"
	"fmt"
	"github.com/zitadel/zitadel/internal/api/http"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"time"
)

type ResourceHandler[T ResourceHolder] interface {
	ResourceNamePlural() schemas.ScimResourceTypePlural
	SchemaType() schemas.ScimSchemaType
	NewResource() T

	Create(ctx context.Context, resource T) (T, error)
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (T, error)
	List(ctx context.Context, request *ListRequest) (*ListResponse[T], error)
}

type Resource struct {
	Schemas []schemas.ScimSchemaType `json:"schemas"`
	Meta    *ResourceMeta            `json:"meta"`
}

type ResourceMeta struct {
	ResourceType schemas.ScimResourceTypeSingular `json:"resourceType"`
	Created      time.Time                        `json:"created"`
	LastModified time.Time                        `json:"lastModified"`
	Version      string                           `json:"version"`
	Location     string                           `json:"location"`
}

type ResourceHolder interface {
	GetResource() *Resource
}

func buildLocation[T ResourceHolder](ctx context.Context, handler ResourceHandler[T], id string) string {
	return fmt.Sprintf("%s%s/%s/%s", http.DomainContext(ctx).Origin(), schemas.HandlerPrefix, handler.ResourceNamePlural(), id)
}
