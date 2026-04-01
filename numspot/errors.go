package numspot

import "errors"

var (
	errAuthenticationFailed       = errors.New("authentication failed")
	errTokenExpirationTooShort    = errors.New("token expiration too short")
	errTimeoutWaitingForVMDeleted = errors.New("timeout waiting for VM to be deleted")
	errTimeoutWaitingForVMState   = errors.New("timeout waiting for VM to reach target state")
	errVMUnexpectedState          = errors.New("VM entered unexpected state")
	errTimeoutWaitingForImage     = errors.New("timeout waiting for image to become available")
	errImageEnteredErrorState     = errors.New("image entered error state")
)
