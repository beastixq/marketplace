package model

import "time"

// Favorite is one entry in a user's favorites list: a product reference plus
// the time the user added it. Product fields reflect current state, not a
// snapshot from the moment of favoriting.
type Favorite struct {
	Product Product
	AddedAt time.Time
}
