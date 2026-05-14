package model

import "time"

type ProductFavorite struct {
	UserID    int64
	ProductID int64
	CreatedAt time.Time
}

type FavoriteState struct {
	ProductID  int64
	IsFavorite bool
}
