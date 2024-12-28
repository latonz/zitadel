package middleware

import (
	"context"
	"github.com/zitadel/zitadel/internal/api/authz"
	zhttp "github.com/zitadel/zitadel/internal/api/http/middleware"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/internal/zerrors"
	smetadata "github.com/zitadel/zitadel/private/api/scim/metadata"
	"net/http"
)

func ScimContextMiddleware(q *query.Queries) func(next zhttp.HandlerFuncWithError) zhttp.HandlerFuncWithError {
	return func(next zhttp.HandlerFuncWithError) zhttp.HandlerFuncWithError {
		return func(w http.ResponseWriter, r *http.Request) error {
			ctx, err := initScimContext(r.Context(), q)
			if err != nil {
				return err
			}

			return next(w, r.WithContext(ctx))
		}
	}
}

func initScimContext(ctx context.Context, q *query.Queries) (context.Context, error) {
	// TODO can this metadata be loaded somewhere from the token?
	data := &smetadata.ScimContextData{
		ProvisioningDomain:          "",
		ExternalIDScopedMetadataKey: smetadata.ScopedKey(smetadata.KeyExternalId),
	}
	ctx = smetadata.SetScimContextData(ctx, data)

	userID := authz.GetCtxData(ctx).UserID
	metadata, err := q.GetUserMetadataByKey(ctx, false, userID, string(smetadata.KeyProvisioningDomain), false)
	if err != nil {
		if zerrors.IsNotFound(err) {
			return ctx, nil
		}

		return ctx, err
	}

	if metadata == nil {
		return ctx, nil
	}

	data.ProvisioningDomain = string(metadata.Value)
	if data.ProvisioningDomain != "" {
		data.ExternalIDScopedMetadataKey = smetadata.ScopeExternalIdKey(data.ProvisioningDomain)
	}
	return ctx, nil
}
