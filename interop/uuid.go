package interop

import (
	"github.com/google/uuid"
)

type UUID = uuid.UUID

var NewUniqueIdString = uuid.NewString
