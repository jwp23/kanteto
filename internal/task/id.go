package task

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// NewID generates a new ULID string for task identification.
func NewID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
