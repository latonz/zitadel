package users

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestServer_GetUser(t *testing.T) {
	project, err := Instance.CreateProject(CTX)
	require.NoError(t, err)

	Instance.CreateMachineUser()
}
