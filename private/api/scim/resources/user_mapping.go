package resources

import (
	"context"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/private/api/scim/metadata"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"golang.org/x/text/language"
	"strconv"
)

func (h *UsersHandler) mapToAddHuman(ctx context.Context, scimUser *ScimUser) (*command.AddHuman, error) {
	human := &command.AddHuman{
		Username:    scimUser.UserName,
		NickName:    scimUser.NickName,
		DisplayName: scimUser.DisplayName,
		Email:       h.mapPrimaryEmail(scimUser),
		Phone:       h.mapPrimaryPhone(scimUser),
		Metadata:    h.mapMetadataToCommands(ctx, scimUser),
	}

	if scimUser.Password != nil {
		human.Password = string(*scimUser.Password)
		scimUser.Password = nil
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

	if err := domain.LanguageIsDefined(scimUser.PreferredLanguage); err != nil {
		human.PreferredLanguage = language.English
		scimUser.PreferredLanguage = language.English
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

func (h *UsersHandler) mapToScimUsers(ctx context.Context, users []*query.User, md map[string]map[metadata.ScopedKey][]byte) []*ScimUser {
	result := make([]*ScimUser, len(users))
	for i, user := range users {
		userMetadata, ok := md[user.ID]
		if !ok {
			userMetadata = make(map[metadata.ScopedKey][]byte)
		}

		result[i] = h.mapToScimUser(ctx, user, userMetadata)
	}

	return result
}

func (h *UsersHandler) mapToScimUser(ctx context.Context, user *query.User, md map[metadata.ScopedKey][]byte) *ScimUser {
	scimUser := &ScimUser{
		Resource:     h.buildResourceForQuery(ctx, user),
		ID:           user.ID,
		ExternalID:   extractScalarMetadata(ctx, md, metadata.KeyExternalId),
		UserName:     user.Username,
		ProfileUrl:   extractHttpURLMetadata(ctx, md, metadata.KeyProfileUrl),
		Title:        extractScalarMetadata(ctx, md, metadata.KeyTitle),
		Locale:       extractScalarMetadata(ctx, md, metadata.KeyLocale),
		Timezone:     extractScalarMetadata(ctx, md, metadata.KeyTimezone),
		Active:       user.State.IsEnabled(),
		Ims:          make([]*ScimIms, 0),
		Addresses:    make([]*ScimAddress, 0),
		Photos:       make([]*ScimPhoto, 0),
		Entitlements: make([]*ScimEntitlement, 0),
		Roles:        make([]*ScimRole, 0),
	}

	if err := extractJsonMetadata(ctx, md, metadata.KeyIms, &scimUser.Ims); err != nil {
		logging.OnError(err).Warn("Could not deserialize scim ims metadata")
	}

	if err := extractJsonMetadata(ctx, md, metadata.KeyAddresses, &scimUser.Addresses); err != nil {
		logging.OnError(err).Warn("Could not deserialize scim addresses metadata")
	}

	if err := extractJsonMetadata(ctx, md, metadata.KeyPhotos, &scimUser.Photos); err != nil {
		logging.OnError(err).Warn("Could not deserialize scim photos metadata")
	}

	if err := extractJsonMetadata(ctx, md, metadata.KeyEntitlements, &scimUser.Entitlements); err != nil {
		logging.OnError(err).Warn("Could not deserialize scim entitlements metadata")
	}

	if err := extractJsonMetadata(ctx, md, metadata.KeyRoles, &scimUser.Roles); err != nil {
		logging.OnError(err).Warn("Could not deserialize scim roles metadata")
	}

	if user.Human != nil {
		mapHumanToScimUser(ctx, user.Human, scimUser, md)
	}

	return scimUser
}

func mapHumanToScimUser(ctx context.Context, human *query.Human, user *ScimUser, md map[metadata.ScopedKey][]byte) {
	user.DisplayName = human.DisplayName
	user.NickName = human.NickName
	user.PreferredLanguage = human.PreferredLanguage
	user.Name = &ScimUserName{
		Formatted:       human.DisplayName,
		FamilyName:      human.LastName,
		GivenName:       human.FirstName,
		MiddleName:      extractScalarMetadata(ctx, md, metadata.KeyMiddleName),
		HonorificPrefix: extractScalarMetadata(ctx, md, metadata.KeyHonorificPrefix),
		HonorificSuffix: extractScalarMetadata(ctx, md, metadata.KeyHonorificSuffix),
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

func (h *UsersHandler) buildResourceForQuery(ctx context.Context, user *query.User) *Resource {
	return &Resource{
		Schemas: []schemas.ScimSchemaType{schemas.IdUser},
		Meta: &ResourceMeta{
			ResourceType: schemas.UserResourceType,
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

func usersToIDs(users []*query.User) []string {
	ids := make([]string, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}
