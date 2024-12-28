//go:build integration

package users

import (
	"github.com/stretchr/testify/assert"
	"github.com/zitadel/zitadel/internal/integration"
	"github.com/zitadel/zitadel/internal/integration/scim"
	"github.com/zitadel/zitadel/pkg/grpc/user/v2"
	"google.golang.org/grpc/codes"
	"net/http"
	"strconv"
	"testing"
)

func TestDeleteUser(t *testing.T) {
	// create user
	createUserResp := Instance.CreateHumanUser(CTX)

	// delete user via scim
	err := Instance.Client.SCIM.Users.Delete(CTX, createUserResp.UserId)
	assert.NoError(t, err)

	// try to delete again => should 404
	err = Instance.Client.SCIM.Users.Delete(CTX, createUserResp.UserId)
	assert.IsType(t, new(scim.ScimError), err)
	assert.Equal(t, strconv.Itoa(http.StatusNotFound), err.(*scim.ScimError).Status)

	// try to get user via api => should 404
	_, err = Instance.Client.UserV2.GetUserByID(CTX, &user.GetUserByIDRequest{UserId: createUserResp.UserId})
	integration.AssertGrpcStatus(t, codes.NotFound, err)
}
