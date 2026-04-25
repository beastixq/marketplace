package service_test

import (
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
)

func testActor(userID int64, role m.UserRole) service.Actor {
	return service.Actor{UserID: userID, Role: role}
}
