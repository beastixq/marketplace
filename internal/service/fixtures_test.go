package service_test

import m "github.com/beastixq/marketplace/internal/model"

func buyerMother() m.User {
	user := someUser
	user.Role = m.RoleBuyer
	phone := testPhone
	user.Phone = &phone
	return user
}

func productMother() m.Product {
	product := someProduct
	description := someProductDescription
	product.Description = &description
	return product
}

type productBuilder struct {
	product m.Product
}

func newProductBuilder() productBuilder {
	return productBuilder{product: productMother()}
}

func (b productBuilder) withID(id int64) productBuilder {
	b.product.ID = id
	return b
}

func (b productBuilder) withSellerID(id int64) productBuilder {
	b.product.SellerID = id
	return b
}

func (b productBuilder) withStock(stock, reserved int) productBuilder {
	b.product.StockQuantity = stock
	b.product.ReservedQuantity = reserved
	return b
}

func (b productBuilder) build() m.Product {
	return b.product
}
