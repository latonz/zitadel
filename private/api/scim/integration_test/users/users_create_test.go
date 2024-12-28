//go:build integration

package users

import (
	"github.com/zitadel/zitadel/private/api/scim/resources"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"testing"
)

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "successfully creates a new user",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Instance.Client.SCIM.Users.Create(CTX, &resources.ScimUser{
				ExternalID: "external-id-1",
				UserName:   "my-user-name",
				Password:   schemas.NewWriteOnlyString("Password1!"),
			})
		})
	}
}
