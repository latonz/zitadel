package resources

import (
	"context"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/api/authz"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/crypto"
	"github.com/zitadel/zitadel/internal/query"
	scim_config "github.com/zitadel/zitadel/private/api/scim/config"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"golang.org/x/text/language"
)

type metadataKey = string

const (
	userResourceNameSingular = "User"
	userResourceNamePlural   = "Users"

	metadataKeyMiddleName      metadataKey = metadataKeyPrefix + "name.middleName"
	metadataKeyHonorificPrefix             = metadataKeyPrefix + "name.honorificPrefix"
	metadataKeyHonorificSuffix             = metadataKeyPrefix + "name.honorificSuffix"
	metadataKeyExternalId                  = metadataKeyPrefix + "externalId"
	metadataKeyProfileUrl                  = metadataKeyPrefix + "profileURL"
	metadataKeyTitle                       = metadataKeyPrefix + "title"
	metadataKeyLocale                      = metadataKeyPrefix + "locale"
	metadataKeyTimezone                    = metadataKeyPrefix + "timezone"
)

var allRelevantMetadataKeys = []metadataKey{
	metadataKeyMiddleName,
	metadataKeyHonorificPrefix,
	metadataKeyHonorificSuffix,
	metadataKeyExternalId,
	metadataKeyProfileUrl,
	metadataKeyTitle,
	metadataKeyLocale,
	metadataKeyTimezone,
}

type UsersHandler struct {
	command     *command.Commands
	query       *query.Queries
	userCodeAlg crypto.EncryptionAlgorithm
	config      *scim_config.Config
}

type ScimUser struct {
	*Resource
	ID                string             `json:"id"`
	ExternalID        string             `json:"externalId,omitempty"`
	UserName          string             `json:"userName,omitempty"`
	Name              *ScimUserName      `json:"name,omitempty"`
	DisplayName       string             `json:"displayName,omitempty"`
	NickName          string             `json:"nickName,omitempty"`
	ProfileUrl        string             `json:"profileUrl,omitempty"`
	Title             string             `json:"title,omitempty"`
	PreferredLanguage language.Tag       `json:"preferredLanguage,omitempty"`
	Locale            string             `json:"locale,omitempty"`
	Timezone          string             `json:"timezone,omitempty"`
	Active            bool               `json:"active,omitempty"`
	Emails            []*ScimEmail       `json:"emails,omitempty"`
	PhoneNumbers      []*ScimPhoneNumber `json:"phoneNumbers,omitempty"`
	Password          string             `json:"password,omitempty"`

	// TODO add ims and later attributes
}

type ScimEmail struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type ScimPhoneNumber struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type ScimUserName struct {
	Formatted       string `json:"formatted,omitempty"`
	FamilyName      string `json:"familyName,omitempty"`
	GivenName       string `json:"givenName,omitempty"`
	MiddleName      string `json:"middleName,omitempty"`
	HonorificPrefix string `json:"honorificPrefix,omitempty"`
	HonorificSuffix string `json:"honorificSuffix,omitempty"`
}

func NewUsersHandler(
	command *command.Commands,
	query *query.Queries,
	userCodeAlg crypto.EncryptionAlgorithm,
	config *scim_config.Config) ResourceHandler[*ScimUser] {
	return &UsersHandler{command, query, userCodeAlg, config}
}

func (h *UsersHandler) ResourceNamePlural() string {
	return userResourceNamePlural
}

func (h *UsersHandler) NewResource() *ScimUser {
	return &ScimUser{}
}

func (h *UsersHandler) Schema() schemas.ScimSchemaType {
	return schemas.IdUser
}

func (h *UsersHandler) Create(ctx context.Context, user *ScimUser) (*ScimUser, error) {
	orgID := authz.GetCtxData(ctx).OrgID
	addHuman, err := h.mapToAddHuman(user)
	if err != nil {
		return nil, err
	}

	err = h.command.AddUserHuman(ctx, orgID, addHuman, true, h.userCodeAlg)
	if err != nil {
		return nil, err
	}

	user.ID = addHuman.Details.ID
	user.Resource = h.buildResourceForCommand(ctx, addHuman.Details)
	user.Password = ""
	return user, err
}

func (h *UsersHandler) Delete(ctx context.Context, id string) error {
	// TODO if-match support
	_, err := h.command.RemoveUserV2(ctx, id, nil)
	return err
}

func (h *UsersHandler) Get(ctx context.Context, id string) (*ScimUser, error) {
	user, err := h.query.GetUserByID(ctx, false, id)
	if err != nil {
		return nil, err
	}

	metadata, err := h.queryMetadata(ctx, id)
	if err != nil {
		return nil, err
	}
	return h.mapToScimUser(ctx, user, metadata), nil
}

func (h *UsersHandler) List(ctx context.Context, request *ListRequest) (*ListResponse[*ScimUser], error) {
	q := &query.UserSearchQueries{
		SearchRequest: query.SearchRequest{
			Offset: request.StartIndex - 1, // start index is 1 based
			Limit:  request.Count,
			Asc:    true,
		},
	}

	// TODO permissionCheck?
	users, err := h.query.SearchUsers(ctx, q, nil)
	if err != nil {
		return nil, err
	}

	// TODO handle count 0 correct
	if request.Count == 0 {
		users.Users = make([]*query.User, 0)
	}

	scimUsers := h.mapToScimUsers(ctx, users.Users)
	return newListResponse(users.SearchResponse, q.SearchRequest, scimUsers), nil
}

func (h *UsersHandler) queryMetadata(ctx context.Context, id string) (map[metadataKey][]byte, error) {
	keyQueries := make([]query.SearchQuery, len(allRelevantMetadataKeys))
	for i, key := range allRelevantMetadataKeys {
		keyQueries[i] = buildMetadataKeyQuery(key)
	}

	queries := &query.UserMetadataSearchQueries{
		SearchRequest: query.SearchRequest{},
		Queries:       []query.SearchQuery{query.Or(keyQueries...)},
	}

	metadata, err := h.query.SearchUserMetadata(ctx, false, id, queries, false)
	if err != nil {
		return nil, err
	}

	metadataMap := make(map[string][]byte, len(metadata.Metadata))
	for _, entry := range metadata.Metadata {
		metadataMap[entry.Key] = entry.Value
	}

	return metadataMap, nil
}

func buildMetadataKeyQuery(key metadataKey) query.SearchQuery {
	q, err := query.NewUserMetadataKeySearchQuery(key, query.TextEquals)
	if err != nil {
		logging.Panic("Error build user metadata query for key " + key)
	}

	return q
}

func (u *ScimUser) GetResource() *Resource {
	return u.Resource
}
