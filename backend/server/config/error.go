package config

import (
	"fmt"

	"github.com/pkg/errors"
)

// An error returned when specified host is not found in the database.
type HostNotFoundError struct {
	hostID int64
}

// Create new instance of the HostNotFoundError.
func NewHostNotFoundError(hostID int64) error {
	return &HostNotFoundError{
		hostID: hostID,
	}
}

// Returns error string.
func (e HostNotFoundError) Error() string {
	return fmt.Sprintf("host with ID %d not found", e.hostID)
}

// An error returned when it was not possible to lock daemons' configuration.
var ErrLock = errors.New("problem with locking daemons configuration")
