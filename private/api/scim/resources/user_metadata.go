package resources

import (
	"context"
	"encoding/json"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/query"
)

type metadataKey string

const (
	metadataKeyPrefix                      = "urn:zitadel:scim:"
	metadataKeyMiddleName      metadataKey = metadataKeyPrefix + "name.middleName"
	metadataKeyHonorificPrefix metadataKey = metadataKeyPrefix + "name.honorificPrefix"
	metadataKeyHonorificSuffix metadataKey = metadataKeyPrefix + "name.honorificSuffix"
	metadataKeyExternalId      metadataKey = metadataKeyPrefix + "externalId"
	metadataKeyProfileUrl      metadataKey = metadataKeyPrefix + "profileURL"
	metadataKeyTitle           metadataKey = metadataKeyPrefix + "title"
	metadataKeyLocale          metadataKey = metadataKeyPrefix + "locale"
	metadataKeyTimezone        metadataKey = metadataKeyPrefix + "timezone"
	metadataKeyIms             metadataKey = metadataKeyPrefix + "ims"
	metadataKeyPhotos          metadataKey = metadataKeyPrefix + "photos"
	metadataKeyAddresses       metadataKey = metadataKeyPrefix + "addresses"
	metadataKeyEntitlements    metadataKey = metadataKeyPrefix + "entitlements"
	metadataKeyRoles           metadataKey = metadataKeyPrefix + "roles"
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
	metadataKeyIms,
	metadataKeyPhotos,
	metadataKeyAddresses,
	metadataKeyEntitlements,
	metadataKeyRoles,
}

func (h *UsersHandler) queryMetadataForUsers(ctx context.Context, userIds []string) (map[string]map[metadataKey][]byte, error) {
	queries := h.buildMetadataQueries()

	metadata, err := h.query.SearchUserMetadataForUsers(ctx, false, userIds, queries)
	if err != nil {
		return nil, err
	}

	metadataMap := make(map[string]map[metadataKey][]byte, len(metadata.Metadata))
	for _, entry := range metadata.Metadata {
		userMetadata, ok := metadataMap[entry.UserID]
		if !ok {
			userMetadata = make(map[metadataKey][]byte)
			metadataMap[entry.UserID] = userMetadata
		}

		userMetadata[metadataKey(entry.Key)] = entry.Value
	}

	return metadataMap, nil
}

func (h *UsersHandler) queryMetadataForUser(ctx context.Context, id string) (map[metadataKey][]byte, error) {
	queries := h.buildMetadataQueries()

	metadata, err := h.query.SearchUserMetadata(ctx, false, id, queries, false)
	if err != nil {
		return nil, err
	}

	metadataMap := make(map[metadataKey][]byte, len(metadata.Metadata))
	for _, entry := range metadata.Metadata {
		metadataMap[metadataKey(entry.Key)] = entry.Value
	}

	return metadataMap, nil
}

func (h *UsersHandler) buildMetadataQueries() *query.UserMetadataSearchQueries {
	keyQueries := make([]query.SearchQuery, len(allRelevantMetadataKeys))
	for i, key := range allRelevantMetadataKeys {
		keyQueries[i] = buildMetadataKeyQuery(key)
	}

	queries := &query.UserMetadataSearchQueries{
		SearchRequest: query.SearchRequest{},
		Queries:       []query.SearchQuery{query.Or(keyQueries...)},
	}
	return queries
}

func buildMetadataKeyQuery(key metadataKey) query.SearchQuery {
	q, err := query.NewUserMetadataKeySearchQuery(string(key), query.TextEquals)
	if err != nil {
		logging.Panic("Error build user metadata query for key " + key)
	}

	return q
}

func (h *UsersHandler) mapMetadataToCommands(user *ScimUser) []*command.AddMetadataEntry {
	metadata := make([]*command.AddMetadataEntry, 0, len(allRelevantMetadataKeys))
	for _, key := range allRelevantMetadataKeys {
		value, err := getValueForMetadataKey(user, key)
		if err != nil {
			logging.OnError(err).Warn("Failed to serialize scim metadata")
			continue
		}

		if len(value) > 0 {
			metadata = append(metadata, &command.AddMetadataEntry{
				Key:   string(key),
				Value: value,
			})
		}
	}

	return metadata
}

func getValueForMetadataKey(user *ScimUser, key metadataKey) ([]byte, error) {
	value := getRawValueForMetadataKey(user, key)
	if value == nil {
		return nil, nil
	}

	switch key {
	case metadataKeyEntitlements:
		fallthrough
	case metadataKeyIms:
		fallthrough
	case metadataKeyPhotos:
		fallthrough
	case metadataKeyAddresses:
		fallthrough
	case metadataKeyRoles:
		return json.Marshal(value)
	default:
		valueStr := value.(string)
		if valueStr == "" {
			return nil, nil
		}
		return []byte(valueStr), nil
	}
}

func getRawValueForMetadataKey(user *ScimUser, key metadataKey) interface{} {
	switch key {
	case metadataKeyIms:
		return user.Ims
	case metadataKeyPhotos:
		return user.Photos
	case metadataKeyAddresses:
		return user.Addresses
	case metadataKeyEntitlements:
		return user.Entitlements
	case metadataKeyRoles:
		return user.Roles
	case metadataKeyMiddleName:
		if user.Name == nil {
			return ""
		}
		return user.Name.MiddleName
	case metadataKeyHonorificPrefix:
		if user.Name == nil {
			return ""
		}
		return user.Name.HonorificPrefix
	case metadataKeyHonorificSuffix:
		if user.Name == nil {
			return ""
		}
		return user.Name.HonorificSuffix
	case metadataKeyExternalId:
		return user.ExternalID
	case metadataKeyProfileUrl:
		return user.ProfileUrl
	case metadataKeyTitle:
		return user.Title
	case metadataKeyLocale:
		return user.Locale
	case metadataKeyTimezone:
		return user.Timezone
	default:
		panic("unknown metadata key" + key)
	}
}

func extractScalarMetadata(metadata map[metadataKey][]byte, key metadataKey) string {
	val, ok := metadata[key]
	if !ok {
		return ""
	}

	return string(val)
}

func extractJsonMetadata(metadata map[metadataKey][]byte, key metadataKey, v interface{}) error {
	val, ok := metadata[key]
	if !ok {
		return nil
	}

	return json.Unmarshal(val, v)
}
