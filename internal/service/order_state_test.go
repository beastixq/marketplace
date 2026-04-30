package service

import (
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
)

func TestValidateOrderStatusTransitionMatrix(t *testing.T) {
	statuses := []m.OrderStatus{
		m.StatusDraft,
		m.StatusPending,
		m.StatusPaid,
		m.StatusShipped,
		m.StatusDelivered,
		m.StatusCancelled,
	}
	transitions := []orderTransition{
		orderTransitionCheckout,
		orderTransitionPay,
		orderTransitionCancel,
		orderTransitionExpire,
		orderTransitionShip,
		orderTransitionDeliver,
	}
	valid := map[orderTransition]map[m.OrderStatus]bool{
		orderTransitionCheckout: {m.StatusDraft: true},
		orderTransitionPay:      {m.StatusPending: true},
		orderTransitionCancel:   {m.StatusPending: true, m.StatusPaid: true},
		orderTransitionExpire:   {m.StatusPending: true},
		orderTransitionShip:     {m.StatusPaid: true},
		orderTransitionDeliver:  {m.StatusShipped: true},
	}

	for _, transition := range transitions {
		for _, status := range statuses {
			err := validateOrderStatusTransition(status, transition)
			wantErr := !valid[transition][status]
			if wantErr {
				if !errors.Is(err, ErrOrderStatusInvalid) {
					t.Fatalf("%s from %s: expected ErrOrderStatusInvalid, got %v", transition, status, err)
				}
				continue
			}
			if err != nil {
				t.Fatalf("%s from %s: unexpected error: %v", transition, status, err)
			}
		}
	}
}

func TestOrderTransitionStatuses(t *testing.T) {
	from, to := orderTransitionStatuses(orderTransitionCancel)
	if to != m.StatusCancelled {
		t.Fatalf("to: got %q, want %q", to, m.StatusCancelled)
	}
	if len(from) != 2 || from[0] != m.StatusPending || from[1] != m.StatusPaid {
		t.Fatalf("from: got %v, want [%s %s]", from, m.StatusPending, m.StatusPaid)
	}

	from[0] = m.StatusDraft
	again, _ := orderTransitionStatuses(orderTransitionCancel)
	if again[0] != m.StatusPending {
		t.Fatalf("from slice must be cloned, got %v", again)
	}
}
