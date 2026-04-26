package techui

import (
	"context"
	"io"
	"log/slog"

	servicecomponent "github.com/beastixq/marketplace/internal/component/service"
	ui "github.com/beastixq/marketplace/internal/techui"
)

type Component struct {
	app *ui.App
}

func New(services *servicecomponent.Component, in io.Reader, out io.Writer, logger *slog.Logger) *Component {
	return &Component{
		app: ui.New(ui.ServicePorts{
			Auth:               services.Auth,
			UserProfile:        services.User,
			UserAdministration: services.User,
			SellerProfile:      services.Seller,
			SellerStatistics:   services.Seller,
			ProductCatalog:     services.Product,
			ProductDetails:     services.Product,
			ProductManagement:  services.Product,
			CategoryBrowser:    services.Category,
			CategoryManagement: services.Category,
			AddressBook:        services.Address,
			Cart:               services.Order,
			BuyerOrders:        services.Order,
			SellerOrders:       services.Order,
			Payments:           services.Payment,
			ReviewManagement:   services.Review,
		}, in, out, logger),
	}
}

func (c *Component) Run(ctx context.Context) error {
	return c.app.Run(ctx)
}
