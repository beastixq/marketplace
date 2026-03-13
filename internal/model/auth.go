package model

import (
	"time"

	"github.com/google/uuid"
)

type TokenClaims struct {
	UserID int64
	Role   UserRole
	Exp    time.Time
	JTI    uuid.UUID
}
