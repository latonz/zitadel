package scim

import "github.com/zitadel/zitadel/internal/api/authz"

// TODO
var AuthMapping = authz.MethodMapping{
	"GET:/scim/v2/Users/{id}": authz.Option{
		Permission: "authenticated",
	},
	"DELETE:/scim/v2/Users/{id}": authz.Option{
		Permission: "authenticated",
	},
}
