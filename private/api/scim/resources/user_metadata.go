package resources

import (
	"context"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/query"
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

func (h *UsersHandler) queryMetadataForUsers(ctx context.Context, userIds []string) (map[string]map[metadataKey][]byte, error) {
	queries := h.buildMetadataQueries()

	metadata, err := h.query.SearchUserMetadataForUsers(ctx, false, userIds, queries)
	if err != nil {
		return nil, err
	}

	metadataMap := make(map[string]map[string][]byte, len(metadata.Metadata))
	for _, entry := range metadata.Metadata {
		userMetadata, ok := metadataMap[entry.UserID]
		if !ok {
			userMetadata = make(map[string][]byte)
			metadataMap[entry.UserID] = userMetadata
		}

		userMetadata[entry.Key] = entry.Value
	}

	return metadataMap, nil
}

func (h *UsersHandler) queryMetadataForUser(ctx context.Context, id string) (map[metadataKey][]byte, error) {
	queries := h.buildMetadataQueries()

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
	q, err := query.NewUserMetadataKeySearchQuery(key, query.TextEquals)
	if err != nil {
		logging.Panic("Error build user metadata query for key " + key)
	}

	return q
}

func (h *UsersHandler) mapMetadata(user *ScimUser) []*command.AddMetadataEntry {
	metadata := make([]*command.AddMetadataEntry, 0, len(allRelevantMetadataKeys))
	for _, key := range allRelevantMetadataKeys {
		value := getValueForMetadataKey(user, key)
		if value != "" {
			metadata = append(metadata, &command.AddMetadataEntry{
				Key:   key,
				Value: []byte(value),
			})
		}
	}

	return metadata
}

func getValueForMetadataKey(user *ScimUser, key metadataKey) string {
	switch key {
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
	}

	return ""
}

func extractMetadata(metadata map[metadataKey][]byte, key metadataKey) string {
	val, ok := metadata[key]
	if !ok {
		return ""
	}

	return string(val)
}
