package model

import "time"

type Address struct {
	ID        int64
	UserID    int64
	City      string
	Street    string
	ZipCode   string
	IsDefault bool
	CreatedAt time.Time
}

type AddressCreate struct {
	UserID    int64
	City      string
	Street    string
	ZipCode   string
	IsDefault bool
}

type AddressUpdate struct {
	UserID    *int64
	City      *string
	Street    *string
	ZipCode   *string
	IsDefault *bool
}
