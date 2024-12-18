package scim

import (
	"github.com/zitadel/zitadel/internal/api/authz"
	"github.com/zitadel/zitadel/internal/domain"
)

var AuthMapping = authz.MethodMapping{
	"POST:/scim/v2/Users": {
		Permission: domain.PermissionUserWrite,
	},
	"POST:/scim/v2/Users/.search": {
		Permission: domain.PermissionUserRead,
	},
	"GET:/scim/v2/Users": {
		Permission: domain.PermissionUserRead,
	},
	"GET:/scim/v2/Users/{id}": {
		Permission: domain.PermissionUserRead,
	},
	"DELETE:/scim/v2/Users/{id}": {
		Permission: domain.PermissionUserDelete,
	},
}
