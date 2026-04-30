package service

import (
	"fmt"
	"slices"

	m "github.com/beastixq/marketplace/internal/model"
)

type orderTransition string

const (
	orderTransitionCheckout orderTransition = "checkout"
	orderTransitionPay      orderTransition = "pay"
	orderTransitionCancel   orderTransition = "cancel"
	orderTransitionExpire   orderTransition = "expire"
	orderTransitionShip     orderTransition = "ship"
	orderTransitionDeliver  orderTransition = "deliver"
)

type orderTransitionRule struct {
	from []m.OrderStatus
	to   m.OrderStatus
}

var orderTransitionRules = map[orderTransition]orderTransitionRule{
	orderTransitionCheckout: {from: []m.OrderStatus{m.StatusDraft}, to: m.StatusPending},
	orderTransitionPay:      {from: []m.OrderStatus{m.StatusPending}, to: m.StatusPaid},
	orderTransitionCancel:   {from: []m.OrderStatus{m.StatusPending, m.StatusPaid}, to: m.StatusCancelled},
	orderTransitionExpire:   {from: []m.OrderStatus{m.StatusPending}, to: m.StatusCancelled},
	orderTransitionShip:     {from: []m.OrderStatus{m.StatusPaid}, to: m.StatusShipped},
	orderTransitionDeliver:  {from: []m.OrderStatus{m.StatusShipped}, to: m.StatusDelivered},
}

func validateOrderStatusTransition(current m.OrderStatus, transition orderTransition) error {
	rule := orderTransitionRuleFor(transition)
	if slices.Contains(rule.from, current) {
		return nil
	}
	return fmt.Errorf("%w: cannot %s order with status %q", ErrOrderStatusInvalid, transition, current)
}

func orderTransitionStatuses(transition orderTransition) (from []m.OrderStatus, to m.OrderStatus) {
	rule := orderTransitionRuleFor(transition)
	return slices.Clone(rule.from), rule.to
}

func orderTransitionRuleFor(transition orderTransition) orderTransitionRule {
	rule, ok := orderTransitionRules[transition]
	if !ok {
		panic(fmt.Sprintf("unknown order transition %q", transition))
	}
	return rule
}
