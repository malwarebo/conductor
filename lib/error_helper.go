package errors

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type Error struct {
	err Error
}

