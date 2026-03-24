package model

import "time"

type Review struct {
	ID        int64
	UserID    int64
	ProductID int64
	Rating    int8
	Comment   *string
	CreatedAt time.Time
}

type ReviewCreate struct {
	UserID    int64
	ProductID int64
	Rating    int8
	Comment   *string
}

type ReviewUpdate struct {
	Rating  *int8
	Comment *string
}
