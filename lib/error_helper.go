package errors

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type Error struct {
	err Error
}

func  (e *Error) Error() string {
	return e.err.Error()
}