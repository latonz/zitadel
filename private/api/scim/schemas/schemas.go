package schemas

type ScimSchemaType = string

const (
	idPrefixMessages        = "urn:ietf:params:scim:api:messages:2.0:"
	idPrefixZitadelMessages = "urn:ietf:params:scim:api:zitadel:messages:2.0:"

	IdError ScimSchemaType = idPrefixMessages + "Error"

	IdZitadelErrorDetail ScimSchemaType = idPrefixZitadelMessages + "ErrorDetail"
)
