package ids

import "fmt"

// IDSStatus represents Apple IDS service status codes.
type IDSStatus int

const (
	IDSStatusSuccess                          IDSStatus = 0
	IDSStatusUnauthenticated                  IDSStatus = 6004
	IDSStatusInvalidNameOrPassword            IDSStatus = 6014
	IDSStatusActionRefreshCredentials         IDSStatus = 6030
	IDSStatusWebTunnelServiceResponseTooLarge IDSStatus = 6054
)

func (s IDSStatus) String() string {
	switch s {
	case IDSStatusSuccess:
		return "success"
	case IDSStatusUnauthenticated:
		return "unauthenticated (2FA required)"
	case IDSStatusInvalidNameOrPassword:
		return "invalid credentials"
	case IDSStatusActionRefreshCredentials:
		return "refresh credentials required"
	case IDSStatusWebTunnelServiceResponseTooLarge:
		return "response too large"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

type IDSError struct {
	ErrorCode IDSStatus
}

func (e IDSError) Error() string {
	return e.ErrorCode.String()
}

func (e IDSError) Is(other error) bool {
	o, ok := other.(IDSError)
	return ok && e.ErrorCode == o.ErrorCode
}

var (
	Err2FARequired              = IDSError{ErrorCode: IDSStatusUnauthenticated}
	ErrInvalidNameOrPassword    = IDSError{ErrorCode: IDSStatusInvalidNameOrPassword}
	ErrActionRefreshCredentials = IDSError{ErrorCode: IDSStatusActionRefreshCredentials}
)
