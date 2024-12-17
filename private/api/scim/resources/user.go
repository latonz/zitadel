package resources

import (
	"context"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"golang.org/x/text/language"
	"strconv"
)

type metadataKey = string

const (
	UserResourceNameSingular = "User"
	UserResourceNamePlural   = "Users"

	// TODO scoping?
	metadataKeyMiddleName      metadataKey = "name.middleName"
	metadataKeyHonorificPrefix             = "name.honorificPrefix"
	metadataKeyHonorificSuffix             = "name.honorificSuffix"
	metadataKeyExternalId                  = "externalId"
	metadataKeyProfileUrl                  = "profileURL"
	metadataKeyTitle                       = "title"
	metadataKeyLocale                      = "locale"
	metadataKeyTimezone                    = "timezone"
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
	command *command.Commands
	query   *query.Queries
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

func NewUsersHandler(command *command.Commands, query *query.Queries) ResourceHandler[*ScimUser] {
	return &UsersHandler{command, query}
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
	return mapToScimUser(ctx, user, metadata), nil
}

func (h *UsersHandler) Delete(ctx context.Context, id string) error {
	_, err := h.command.RemoveUserV2(ctx, id, nil)
	return err
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

func mapToScimUser(ctx context.Context, user *query.User, metadata map[metadataKey][]byte) *ScimUser {
	scimUser := &ScimUser{
		Resource: &Resource{
			Schemas: []schemas.ScimSchemaType{schemas.IdUser},
			Meta: &ResourceMeta{
				ResourceType: UserResourceNameSingular,
				Created:      user.CreationDate.UTC(),
				LastModified: user.ChangeDate.UTC(),
				Version:      strconv.FormatUint(user.Sequence, 10),
				Location:     buildLocation(ctx, UserResourceNamePlural, user.ID),
			},
		},
		ID:         user.ID,
		ExternalID: mapFromMetadata(metadata, metadataKeyExternalId),
		UserName:   user.Username,
		ProfileUrl: mapFromMetadata(metadata, metadataKeyProfileUrl),
		Title:      mapFromMetadata(metadata, metadataKeyTitle),
		Locale:     mapFromMetadata(metadata, metadataKeyLocale),
		Timezone:   mapFromMetadata(metadata, metadataKeyTimezone),
		Active:     user.State.IsEnabled(),
	}

	if user.Human != nil {
		mapHuman(user.Human, scimUser, metadata)
	}
	return scimUser
}

func mapHuman(human *query.Human, user *ScimUser, metadata map[metadataKey][]byte) {
	user.DisplayName = human.DisplayName
	user.NickName = human.NickName
	user.PreferredLanguage = human.PreferredLanguage
	user.Name = &ScimUserName{
		Formatted:       human.DisplayName,
		FamilyName:      human.LastName,
		GivenName:       human.FirstName,
		MiddleName:      mapFromMetadata(metadata, metadataKeyMiddleName),
		HonorificPrefix: mapFromMetadata(metadata, metadataKeyHonorificPrefix),
		HonorificSuffix: mapFromMetadata(metadata, metadataKeyHonorificSuffix),
	}

	if string(human.Email) != "" {
		user.Emails = []*ScimEmail{
			{
				Value:   string(human.Email),
				Primary: true,
			},
		}
	}

	if string(human.Phone) != "" {
		user.PhoneNumbers = []*ScimPhoneNumber{
			{
				Value:   string(human.Phone),
				Primary: true,
			},
		}
	}
}

func mapFromMetadata(metadata map[metadataKey][]byte, key metadataKey) string {
	val, ok := metadata[key]
	if !ok {
		return ""
	}

	return string(val)
}

func (u *ScimUser) GetResource() *Resource {
	return u.Resource
}
