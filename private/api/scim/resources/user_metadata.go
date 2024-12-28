package resources

import (
	"context"
	"encoding/json"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/private/api/scim/metadata"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
)

func (h *UsersHandler) queryMetadataForUsers(ctx context.Context, userIds []string) (map[string]map[metadata.ScopedKey][]byte, error) {
	queries := h.buildMetadataQueries(ctx)

	md, err := h.query.SearchUserMetadataForUsers(ctx, false, userIds, queries)
	if err != nil {
		return nil, err
	}

	metadataMap := make(map[string]map[metadata.ScopedKey][]byte, len(md.Metadata))
	for _, entry := range md.Metadata {
		userMetadata, ok := metadataMap[entry.UserID]
		if !ok {
			userMetadata = make(map[metadata.ScopedKey][]byte)
			metadataMap[entry.UserID] = userMetadata
		}

		userMetadata[metadata.ScopedKey(entry.Key)] = entry.Value
	}

	return metadataMap, nil
}

func (h *UsersHandler) queryMetadataForUser(ctx context.Context, id string) (map[metadata.ScopedKey][]byte, error) {
	queries := h.buildMetadataQueries(ctx)

	md, err := h.query.SearchUserMetadata(ctx, false, id, queries, false)
	if err != nil {
		return nil, err
	}

	metadataMap := make(map[metadata.ScopedKey][]byte, len(md.Metadata))
	for _, entry := range md.Metadata {
		metadataMap[metadata.ScopedKey(entry.Key)] = entry.Value
	}

	return metadataMap, nil
}

func (h *UsersHandler) buildMetadataQueries(ctx context.Context) *query.UserMetadataSearchQueries {
	keyQueries := make([]query.SearchQuery, len(metadata.ScimUserRelevantMetadataKeys))
	for i, key := range metadata.ScimUserRelevantMetadataKeys {
		keyQueries[i] = buildMetadataKeyQuery(ctx, key)
	}

	queries := &query.UserMetadataSearchQueries{
		SearchRequest: query.SearchRequest{},
		Queries:       []query.SearchQuery{query.Or(keyQueries...)},
	}
	return queries
}

func buildMetadataKeyQuery(ctx context.Context, key metadata.Key) query.SearchQuery {
	scopedKey := metadata.ScopeKey(ctx, key)
	q, err := query.NewUserMetadataKeySearchQuery(string(scopedKey), query.TextEquals)
	if err != nil {
		logging.Panic("Error build user metadata query for key " + key)
	}

	return q
}

func (h *UsersHandler) mapMetadataToCommands(ctx context.Context, user *ScimUser) []*command.AddMetadataEntry {
	md := make([]*command.AddMetadataEntry, 0, len(metadata.ScimUserRelevantMetadataKeys))
	for _, key := range metadata.ScimUserRelevantMetadataKeys {
		value, err := getValueForMetadataKey(user, key)
		if err != nil {
			logging.OnError(err).Warn("Failed to serialize scim metadata")
			continue
		}

		if len(value) > 0 {
			md = append(md, &command.AddMetadataEntry{
				Key:   string(metadata.ScopeKey(ctx, key)),
				Value: value,
			})
		}
	}

	return md
}

func getValueForMetadataKey(user *ScimUser, key metadata.Key) ([]byte, error) {
	value := getRawValueForMetadataKey(user, key)
	if value == nil {
		return nil, nil
	}

	switch key {
	case metadata.KeyEntitlements:
		fallthrough
	case metadata.KeyIms:
		fallthrough
	case metadata.KeyPhotos:
		fallthrough
	case metadata.KeyAddresses:
		fallthrough
	case metadata.KeyRoles:
		return json.Marshal(value)
	case metadata.KeyProfileUrl:
		return []byte(value.(*schemas.HttpURL).String()), nil
	default:
		valueStr := value.(string)
		if valueStr == "" {
			return nil, nil
		}
		return []byte(valueStr), nil
	}
}

func getRawValueForMetadataKey(user *ScimUser, key metadata.Key) interface{} {
	switch key {
	case metadata.KeyIms:
		return user.Ims
	case metadata.KeyPhotos:
		return user.Photos
	case metadata.KeyAddresses:
		return user.Addresses
	case metadata.KeyEntitlements:
		return user.Entitlements
	case metadata.KeyRoles:
		return user.Roles
	case metadata.KeyMiddleName:
		if user.Name == nil {
			return ""
		}
		return user.Name.MiddleName
	case metadata.KeyHonorificPrefix:
		if user.Name == nil {
			return ""
		}
		return user.Name.HonorificPrefix
	case metadata.KeyHonorificSuffix:
		if user.Name == nil {
			return ""
		}
		return user.Name.HonorificSuffix
	case metadata.KeyExternalId:
		return user.ExternalID
	case metadata.KeyProfileUrl:
		return user.ProfileUrl
	case metadata.KeyTitle:
		return user.Title
	case metadata.KeyLocale:
		return user.Locale
	case metadata.KeyTimezone:
		return user.Timezone
	default:
		panic("unknown metadata key" + key)
	}
}

func extractScalarMetadata(ctx context.Context, md map[metadata.ScopedKey][]byte, key metadata.Key) string {
	val, ok := md[metadata.ScopeKey(ctx, key)]
	if !ok {
		return ""
	}

	return string(val)
}

func extractHttpURLMetadata(ctx context.Context, md map[metadata.ScopedKey][]byte, key metadata.Key) *schemas.HttpURL {
	val, ok := md[metadata.ScopeKey(ctx, key)]
	if !ok {
		return nil
	}

	url, err := schemas.ParseHTTPURL(string(val))
	if err != nil {
		logging.OnError(err).Warn("Failed to parse scim url metadata for " + key)
		return nil
	}

	return url
}

func extractJsonMetadata(ctx context.Context, md map[metadata.ScopedKey][]byte, key metadata.Key, v interface{}) error {
	val, ok := md[metadata.ScopeKey(ctx, key)]
	if !ok {
		return nil
	}

	return json.Unmarshal(val, v)
}
