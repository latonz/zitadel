package resources

import (
	"context"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/i18n"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"golang.org/x/text/language"
	"strconv"
)

func (h *UsersHandler) mapToAddHuman(scimUser *ScimUser) (*command.AddHuman, error) {
	human := &command.AddHuman{
		Username:    scimUser.UserName,
		NickName:    scimUser.NickName,
		DisplayName: scimUser.DisplayName,
		Password:    scimUser.Password,
		Email:       h.mapPrimaryEmail(scimUser),
		Phone:       h.mapPrimaryPhone(scimUser),
		Metadata:    h.mapMetadata(scimUser),
	}

	if scimUser.Name != nil {
		human.FirstName = scimUser.Name.GivenName
		human.LastName = scimUser.Name.FamilyName

		// the direct mapping displayName => displayName has priority
		// over the formatted name assignment
		if human.DisplayName == "" {
			human.DisplayName = scimUser.Name.Formatted
		}
	}

	// TODO why is the language not validated on the command layer
	if err := domain.LanguageIsDefined(scimUser.PreferredLanguage); err != nil {
		human.PreferredLanguage = language.English
		scimUser.PreferredLanguage = language.English
	} else if err := domain.LanguagesAreSupported(i18n.SupportedLanguages(), scimUser.PreferredLanguage); err != nil {
		return nil, err
	}

	return human, nil
}

func (h *UsersHandler) mapPrimaryEmail(scimUser *ScimUser) command.Email {
	for _, email := range scimUser.Emails {
		if !email.Primary {
			continue
		}

		return command.Email{
			Address:  domain.EmailAddress(email.Value),
			Verified: h.config.EmailVerified,
		}
	}

	return command.Email{}
}

func (h *UsersHandler) mapPrimaryPhone(scimUser *ScimUser) command.Phone {
	for _, phone := range scimUser.PhoneNumbers {
		if !phone.Primary {
			continue
		}

		return command.Phone{
			Number:   domain.PhoneNumber(phone.Value),
			Verified: h.config.PhoneVerified,
		}
	}

	return command.Phone{}
}

func (h *UsersHandler) mapToScimUsers(ctx context.Context, users []*query.User, metadata map[string]map[metadataKey][]byte) []*ScimUser {
	result := make([]*ScimUser, len(users))
	for i, user := range users {
		userMetadata, ok := metadata[user.ID]
		if !ok {
			userMetadata = make(map[metadataKey][]byte)
		}

		result[i] = h.mapToScimUser(ctx, user, userMetadata)
	}

	return result
}

func (h *UsersHandler) mapToScimUser(ctx context.Context, user *query.User, metadata map[metadataKey][]byte) *ScimUser {
	scimUser := &ScimUser{
		Resource:   h.buildResourceForQuery(ctx, user),
		ID:         user.ID,
		ExternalID: extractMetadata(metadata, metadataKeyExternalId),
		UserName:   user.Username,
		ProfileUrl: extractMetadata(metadata, metadataKeyProfileUrl),
		Title:      extractMetadata(metadata, metadataKeyTitle),
		Locale:     extractMetadata(metadata, metadataKeyLocale),
		Timezone:   extractMetadata(metadata, metadataKeyTimezone),
		Active:     user.State.IsEnabled(),
	}

	if user.Human != nil {
		mapHumanToScimUser(user.Human, scimUser, metadata)
	}
	return scimUser
}

func mapHumanToScimUser(human *query.Human, user *ScimUser, metadata map[metadataKey][]byte) {
	user.DisplayName = human.DisplayName
	user.NickName = human.NickName
	user.PreferredLanguage = human.PreferredLanguage
	user.Name = &ScimUserName{
		Formatted:       human.DisplayName,
		FamilyName:      human.LastName,
		GivenName:       human.FirstName,
		MiddleName:      extractMetadata(metadata, metadataKeyMiddleName),
		HonorificPrefix: extractMetadata(metadata, metadataKeyHonorificPrefix),
		HonorificSuffix: extractMetadata(metadata, metadataKeyHonorificSuffix),
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

func (h *UsersHandler) buildResourceForCommand(ctx context.Context, userDetails *domain.ObjectDetails) *Resource {
	created := userDetails.CreationDate.UTC()
	if created.IsZero() {
		created = userDetails.EventDate.UTC()
	}

	return &Resource{
		Schemas: []schemas.ScimSchemaType{schemas.IdUser},
		Meta: &ResourceMeta{
			ResourceType: userResourceNameSingular,
			Created:      created,
			LastModified: userDetails.EventDate.UTC(),
			Version:      strconv.FormatUint(userDetails.Sequence, 10),
			Location:     buildLocation(ctx, h, userDetails.ID),
		},
	}
}

func (h *UsersHandler) buildResourceForQuery(ctx context.Context, user *query.User) *Resource {
	return &Resource{
		Schemas: []schemas.ScimSchemaType{schemas.IdUser},
		Meta: &ResourceMeta{
			ResourceType: userResourceNameSingular,
			Created:      user.CreationDate.UTC(),
			LastModified: user.ChangeDate.UTC(),
			Version:      strconv.FormatUint(user.Sequence, 10),
			Location:     buildLocation(ctx, h, user.ID),
		},
	}
}

func cascadingMemberships(memberships []*query.Membership) []*command.CascadingMembership {
	cascades := make([]*command.CascadingMembership, len(memberships))
	for i, membership := range memberships {
		cascades[i] = &command.CascadingMembership{
			UserID:        membership.UserID,
			ResourceOwner: membership.ResourceOwner,
			IAM:           cascadingIAMMembership(membership.IAM),
			Org:           cascadingOrgMembership(membership.Org),
			Project:       cascadingProjectMembership(membership.Project),
			ProjectGrant:  cascadingProjectGrantMembership(membership.ProjectGrant),
		}
	}
	return cascades
}

func cascadingIAMMembership(membership *query.IAMMembership) *command.CascadingIAMMembership {
	if membership == nil {
		return nil
	}
	return &command.CascadingIAMMembership{IAMID: membership.IAMID}
}

func cascadingOrgMembership(membership *query.OrgMembership) *command.CascadingOrgMembership {
	if membership == nil {
		return nil
	}
	return &command.CascadingOrgMembership{OrgID: membership.OrgID}
}

func cascadingProjectMembership(membership *query.ProjectMembership) *command.CascadingProjectMembership {
	if membership == nil {
		return nil
	}
	return &command.CascadingProjectMembership{ProjectID: membership.ProjectID}
}

func cascadingProjectGrantMembership(membership *query.ProjectGrantMembership) *command.CascadingProjectGrantMembership {
	if membership == nil {
		return nil
	}
	return &command.CascadingProjectGrantMembership{ProjectID: membership.ProjectID, GrantID: membership.GrantID}
}

func userGrantsToIDs(userGrants []*query.UserGrant) []string {
	converted := make([]string, len(userGrants))
	for i, grant := range userGrants {
		converted[i] = grant.ID
	}
	return converted
}

func userIDs(users []*query.User) []string {
	ids := make([]string, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}
