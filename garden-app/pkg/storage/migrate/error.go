package migrate

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound        = errors.New("migration not found")
	ErrInvalidToType   = errors.New("unexpected To type")
	ErrInvalidFromType = errors.New("unexpected From type")
)

type Error struct {
	Err     error
	Name    string
	Version uint
}

func (e Error) Unwrap() error {
	return e.Err
}

func (e Error) Error() string {
	return fmt.Sprintf("error running migration %q/%d: %s", e.Name, e.Version, e.Err.Error())
}

func errNotFound(version uint) Error {
	return Error{ErrNotFound, "Unknown", version}
}
