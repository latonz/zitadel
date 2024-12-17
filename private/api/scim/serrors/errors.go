package serrors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zitadel/logging"
	http_util "github.com/zitadel/zitadel/internal/api/http"
	"github.com/zitadel/zitadel/internal/i18n"
	"github.com/zitadel/zitadel/internal/zerrors"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"golang.org/x/text/language"
	"net/http"
	"strconv"
)

type scimErrorType = string

type scimError struct {
	Type         scimErrorType
	Detail       string
	Status       int
	zitadelError *zerrors.ZitadelError
}

type scimJsonError struct {
	Schemas       []string               `json:"schemas"`
	ScimType      scimErrorType          `json:"scimType,omitempty"`
	Detail        string                 `json:"detail,omitempty"`
	Status        string                 `json:"status"`
	ZitadelDetail zitadelJsonErrorDetail `json:"urn:ietf:params:scim:api:zitadel:messages:2.0:ErrorDetail,omitempty"`
}

type zitadelJsonErrorDetail struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

var translator *i18n.Translator

func ErrorHandlerMiddleware(next func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	var err error
	translator, err = i18n.NewZitadelTranslator(language.English)
	logging.OnError(err).Panic("unable to get translator")

	return func(w http.ResponseWriter, r *http.Request) {
		if err := next(w, r); err != nil {
			scimErr := mapToScimError(r.Context(), err)
			w.WriteHeader(scimErr.Status)

			resp, jsonErr := json.Marshal(scimErr)
			if jsonErr != nil {
				logging.WithError(jsonErr).Warn("Failed to marshal scim error response")
				return
			}

			if _, err = w.Write(resp); err != nil {
				logging.WithError(err).Warn("Failed to write scim error response body")
			}
		}
	}
}

func (err *scimError) UnmarshalJSON(_ []byte) error {
	panic("UnmarshalJSON is not supported for json errors")
}

func (err *scimError) MarshalJSON() ([]byte, error) {
	jsonErr := scimJsonError{
		Schemas:  []string{schemas.IdError, schemas.IdZitadelErrorDetail},
		ScimType: err.Type,
		Detail:   err.Detail,
		Status:   strconv.Itoa(err.Status),
		ZitadelDetail: zitadelJsonErrorDetail{
			ID:      err.zitadelError.GetID(),
			Message: err.zitadelError.GetMessage(),
		},
	}
	return json.Marshal(jsonErr)
}

func (err *scimError) Error() string {
	return fmt.Sprintf("SCIM Error: %s: %s (%v)", err.Type, err.Detail, err.Status)
}

func mapToScimError(ctx context.Context, err error) *scimError {
	zitadelErr := new(zerrors.ZitadelError)
	if ok := errors.As(err, &zitadelErr); !ok {
		return &scimError{
			Detail: err.Error(),
			Status: http.StatusInternalServerError,
		}
	}

	statusCode, ok := http_util.ZitadelErrorToHTTPStatusCode(err)
	if !ok {
		statusCode = http.StatusInternalServerError
	}

	localizedMsg := translator.LocalizeFromCtx(ctx, zitadelErr.GetMessage(), nil)
	return &scimError{
		Detail:       localizedMsg,
		Status:       statusCode,
		zitadelError: zitadelErr,
	}
}
