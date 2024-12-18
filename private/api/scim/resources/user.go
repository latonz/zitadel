package resources

import (
	"context"
	"github.com/zitadel/zitadel/internal/api/authz"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/crypto"
	"github.com/zitadel/zitadel/internal/query"
	scim_config "github.com/zitadel/zitadel/private/api/scim/config"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"golang.org/x/text/language"
)

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
	memberships, grants, err := h.queryUserDependencies(ctx, id)
	if err != nil {
		return err
	}

	_, err = h.command.RemoveUserV2(ctx, id, memberships, grants...)
	return err
}

func (h *UsersHandler) Get(ctx context.Context, id string) (*ScimUser, error) {
	user, err := h.query.GetUserByID(ctx, false, id)
	if err != nil {
		return nil, err
	}

	metadata, err := h.queryMetadataForUser(ctx, id)
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

	if request.Count == 0 {
		count, err := h.query.CountUsers(ctx, q)
		if err != nil {
			return nil, err
		}

		return newListResponse(count, q.SearchRequest, make([]*ScimUser, 0)), nil
	}

	users, err := h.query.SearchUsers(ctx, q, nil)
	if err != nil {
		return nil, err
	}

	metadata, err := h.queryMetadataForUsers(ctx, userIDs(users.Users))
	if err != nil {
		return nil, err
	}

	scimUsers := h.mapToScimUsers(ctx, users.Users, metadata)
	return newListResponse(users.SearchResponse.Count, q.SearchRequest, scimUsers), nil
}

func (u *ScimUser) GetResource() *Resource {
	return u.Resource
}

func (h *UsersHandler) queryUserDependencies(ctx context.Context, userID string) ([]*command.CascadingMembership, []string, error) {
	userGrantUserQuery, err := query.NewUserGrantUserIDSearchQuery(userID)
	if err != nil {
		return nil, nil, err
	}

	grants, err := h.query.UserGrants(ctx, &query.UserGrantsQueries{
		Queries: []query.SearchQuery{userGrantUserQuery},
	}, true)
	if err != nil {
		return nil, nil, err
	}

	membershipsUserQuery, err := query.NewMembershipUserIDQuery(userID)
	if err != nil {
		return nil, nil, err
	}

	memberships, err := h.query.Memberships(ctx, &query.MembershipSearchQuery{
		Queries: []query.SearchQuery{membershipsUserQuery},
	}, false)

	if err != nil {
		return nil, nil, err
	}
	return cascadingMemberships(memberships.Memberships), userGrantsToIDs(grants.UserGrants), nil
}
