package model

import "time"

type Favorite struct {
	UserID    int64
	ProductID int64
	CreatedAt time.Time
}
