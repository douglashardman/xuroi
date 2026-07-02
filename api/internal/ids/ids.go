package ids

import (
	"crypto/rand"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

func New(prefix string) string {
	return prefix + strings.ToLower(ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String())
}