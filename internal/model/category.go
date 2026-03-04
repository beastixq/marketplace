package model

type Category struct {
	ID          int64
	ParentID    *int64
	Name        string
	Description *string
}

type CategoryCreate struct {
	ParentID    *int64
	Name        string
	Description *string
}

type CategoryUpdate struct {
	ParentID    *int64
	Name        *string
	Description *string
}
