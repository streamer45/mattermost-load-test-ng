package simulcontroller

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
)

// func (c *SimulController) newInfoStatus(info string) control.UserStatus {
// 	return control.UserStatus{
// 		ControllerId: c.id,
// 		User:         c.user,
// 		Code:         control.USER_STATUS_INFO,
// 		Info:         info,
// 		Err:          nil,
// 	}
// }

func (c *SimulController) newErrorStatus(err error) control.UserStatus {
	origin := control.GetErrOrigin()
	return control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Code:         control.USER_STATUS_ERROR,
		Info:         "",
		Err: &control.ControlError{
			Err:    err,
			Origin: origin,
		},
	}
}
