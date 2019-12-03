package userentity

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/require"
)

func TestGetUserFromStore(t *testing.T) {
	th := Setup(t).Init()

	user, err := th.User.getUserFromStore()
	require.Nil(t, user)
	require.Error(t, err)
	require.EqualError(t, err, "user was not initialized")

	th.User.store.SetUser(&model.User{
		Id: "someid",
	})
	user, err = th.User.getUserFromStore()
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, "someid", user.Id)
}
