package resources

import (
	"context"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"golang.org/x/text/language"
	"net/url"
	"strconv"
)

const (
	UserResourceNameSingular = "User"
	UserResourceNamePlural   = "Users"
)

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
	ProfileUrl        *url.URL           `json:"profileUrl,omitempty"`
	Title             string             `json:"title,omitempty"`
	PreferredLanguage language.Tag       `json:"preferredLanguage,omitempty"`
	Locale            string             `json:"locale,omitempty"`
	Timezone          string             `json:"timezone,omitempty"`
	Active            bool               `json:"active,omitempty"`
	Emails            []*ScimEmail       `json:"emails,omitempty"`
	PhoneNumbers      []*ScimPhoneNumber `json:"phoneNumbers,omitempty"`
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

	return mapToScimUser(user), nil
}

func (h *UsersHandler) Delete(ctx context.Context, id string) error {
	_, err := h.command.RemoveUserV2(ctx, id, nil)
	return err
}

func mapToScimUser(user *query.User) *ScimUser {
	scimUser := &ScimUser{
		Resource: &Resource{
			Schemas: []schemas.ScimSchemaType{schemas.IdUser},
			Meta: &ResourceMeta{
				ResourceType: UserResourceNameSingular,
				Created:      user.CreationDate.UTC(),
				LastModified: user.ChangeDate.UTC(),
				Version:      strconv.FormatUint(user.Sequence, 10),
				Location:     "",
			},
		},
		ID:         user.ID,
		ExternalID: "", // TODO from metadata
		UserName:   user.Username,
		ProfileUrl: nil, // TODO from metadata
		Title:      "",  // TODO from metadata
		Locale:     "",  // TODO from metadata
		Timezone:   "",  // TODO from metadata
		Active:     user.State.IsEnabled(),
	}

	if user.Human != nil {
		mapHuman(user.Human, scimUser)
	}
	return scimUser
}

func mapHuman(human *query.Human, user *ScimUser) {
	user.DisplayName = human.DisplayName
	user.NickName = human.NickName
	user.PreferredLanguage = human.PreferredLanguage
	user.Name = &ScimUserName{
		Formatted:       human.DisplayName,
		FamilyName:      human.LastName,
		GivenName:       human.FirstName,
		MiddleName:      "", // TODO from metadata
		HonorificPrefix: "", // TODO from metadata
		HonorificSuffix: "", // TODO from metadata
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

func (u *ScimUser) GetResource() *Resource {
	return u.Resource
}
