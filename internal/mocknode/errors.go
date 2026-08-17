package mocknode

import "errors"

var (
	ErrInvalidOffer      = errors.New("invalid task offer")
	ErrDuplicateConflict = errors.New("duplicate offer conflicts with the first message")
	ErrOfferRejected     = errors.New("task offer rejected")
	ErrOfferTimeout      = errors.New("task offer timed out")
	ErrDisconnected      = errors.New("mock node disconnected")
	ErrLeaseInvalid      = errors.New("lease is not active")
	ErrLateResult        = errors.New("chat result arrived too late")
)
