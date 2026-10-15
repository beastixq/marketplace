//go:build integration

package service_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/beastixq/marketplace/internal/adapter/payment"
	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	s "github.com/beastixq/marketplace/internal/service"
)

func TestPaymentService_ProcessOrderPayment_Integration(t *testing.T) {
	for _, scenario := range []string{"success and repeat", "wrong amount", "declined", "expired"} {
		t.Run(scenario, func(t *testing.T) {
			f := newServiceIntegration(t)
			ctx := context.Background()
			buyer := f.createActor(m.RoleBuyer)
			seller, sellerID := f.createSeller()
			orderID := createPendingOrder(t, f.orderService(), buyer, f.createProduct(seller, sellerID, 1), f.createAddress(buyer))
			paymentService := s.NewPaymentService(repo.NewOrderRepo(f.pool), payment.NewMockBankGateway("http://mock.test"), time.Minute)
			order, err := repo.NewOrderRepo(f.pool).GetOrderByID(ctx, orderID)
			mustIntegration(t, err)
			paymentService.Clock = fakeClock{now: order.CreatedAt.Add(time.Second)}
			link, _, err := paymentService.GetOrderPaymentURL(ctx, buyer, orderID)
			mustIntegration(t, err)
			parsed, err := url.Parse(link)
			mustIntegration(t, err)
			token := parsed.Query().Get("token")
			var want error
			switch scenario {
			case "wrong amount":
				token = base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"order_id":%d,"amount":"99","success":true}`, orderID)))
				want = s.ErrInvalidPaymentAmount
			case "declined":
				token = base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"order_id":%d,"amount":"100","success":false}`, orderID)))
				want = s.ErrPaymentDeclined
			case "expired":
				paymentService.Clock = fakeClock{now: order.CreatedAt.Add(time.Minute)}
				want = s.ErrPaymentExpired
			}

			err = paymentService.ProcessOrderPayment(ctx, token)
			if !errors.Is(err, want) {
				t.Fatalf("payment = %v, want %v", err, want)
			}
			wantStatus := m.StatusPending
			if want == nil {
				wantStatus = m.StatusPaid
				mustIntegration(t, paymentService.ProcessOrderPayment(ctx, token))
			}
			order, err = repo.NewOrderRepo(f.pool).GetOrderByID(ctx, orderID)
			mustIntegration(t, err)
			if order.Status != wantStatus {
				t.Fatalf("persisted status = %s", order.Status)
			}
		})
	}
}
