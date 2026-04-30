package service

import (
	"time"

	payment "github.com/beastixq/marketplace/internal/adapter/payment"
	repocomponent "github.com/beastixq/marketplace/internal/component/repository"
	svc "github.com/beastixq/marketplace/internal/service"
)

type Config struct {
	BcryptCost            int
	JWTSecret             string
	TokenTTL              time.Duration
	PaymentTTL            time.Duration
	PaymentGatewayBaseURL string
	TokenBlocklist        svc.TokenBlocklist
}

type Component struct {
	User       svc.UserService
	Seller     svc.SellerService
	Address    svc.AddressService
	Review     svc.ReviewService
	Product    svc.ProductService
	Order      svc.OrderService
	Category   svc.CategoryService
	Backoffice svc.BackofficeService
	Auth       svc.AuthService
	Payment    *svc.PaymentService
}

func New(repos *repocomponent.Component, cfg Config) *Component {
	cfg = cfg.withDefaults()

	user := svc.NewUserService(repos.User, cfg.BcryptCost)
	seller := svc.NewSellerService(repos.Seller)
	address := svc.NewAddressService(repos.Address)
	review := svc.NewReviewService(repos.Review, repos.ReviewPurchase)
	product := svc.NewProductService(repos.Product, repos.Review, repos.Seller)
	order := svc.NewOrderService(repos.Order, repos.OrderItem, repos.Product, repos.Address, repos.Seller, repos.TxManager)
	category := svc.NewCategoryService(repos.Category)
	backoffice := svc.NewBackofficeService(repos.Backoffice)
	auth := svc.NewAuthService(user, cfg.TokenBlocklist, cfg.JWTSecret, cfg.TokenTTL)
	gateway := payment.NewMockBankGateway(cfg.PaymentGatewayBaseURL)
	payments := svc.NewPaymentService(repos.Order, gateway, cfg.PaymentTTL)

	return &Component{
		User:       user,
		Seller:     seller,
		Address:    address,
		Review:     review,
		Product:    product,
		Order:      order,
		Category:   category,
		Backoffice: backoffice,
		Auth:       auth,
		Payment:    payments,
	}
}

func (cfg Config) withDefaults() Config {
	if cfg.BcryptCost == 0 {
		cfg.BcryptCost = 10
	}
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = 24 * time.Hour
	}
	if cfg.PaymentTTL == 0 {
		cfg.PaymentTTL = 15 * time.Minute
	}
	if cfg.PaymentGatewayBaseURL == "" {
		cfg.PaymentGatewayBaseURL = "http://localhost:8080"
	}
	return cfg
}
