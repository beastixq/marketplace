package mongorepo

import (
	"context"
	"errors"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *Store) GetOrderByID(ctx context.Context, id int64) (m.Order, error) {
	var doc orderDoc
	err := s.collection(ordersCollection).FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if notFound(err) {
		return m.Order{}, service.ErrNotFound
	}
	if err != nil {
		return m.Order{}, err
	}
	return doc.toModel()
}

func (s *Store) GetOrdersByUserID(ctx context.Context, userID int64, pg m.PaginationOpts) ([]m.Order, error) {
	filter := bson.M{"user_id": userID}
	return s.findOrders(ctx, filter, pg)
}

func (s *Store) GetSellerOrdersBySellerID(ctx context.Context, sellerID int64, pg m.PaginationOpts) ([]m.Order, error) {
	filter := bson.M{"seller_id": sellerID}
	return s.findOrders(ctx, filter, pg)
}

func (s *Store) findOrders(ctx context.Context, filter bson.M, pg m.PaginationOpts) ([]m.Order, error) {
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if pg.Page > 0 && pg.Limit > 0 {
		findOpts.SetSkip(int64((pg.Page - 1) * pg.Limit))
		findOpts.SetLimit(int64(pg.Limit))
	}
	cursor, err := s.collection(ordersCollection).Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	orders := make([]m.Order, 0)
	for cursor.Next(ctx) {
		var doc orderDoc
		if err = cursor.Decode(&doc); err != nil {
			return nil, err
		}
		order, err := doc.toModel()
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, cursor.Err()
}

func (s *Store) GetExpiredPendingOrders(ctx context.Context, deadline time.Time) ([]m.Order, error) {
	cursor, err := s.collection(ordersCollection).Find(ctx, bson.M{
		"status":     m.StatusPending,
		"created_at": bson.M{"$lte": deadline},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	orders := make([]m.Order, 0)
	for cursor.Next(ctx) {
		var doc orderDoc
		if err = cursor.Decode(&doc); err != nil {
			return nil, err
		}
		order, err := doc.toModel()
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, cursor.Err()
}

func (s *Store) CreateOrder(ctx context.Context, oc m.OrderCreate) (int64, error) {
	id, err := s.nextID(ctx, ordersCollection)
	if err != nil {
		return 0, err
	}
	doc, err := newOrderDoc(id, oc)
	if err != nil {
		return 0, err
	}
	if _, err = s.collection(ordersCollection).InsertOne(ctx, doc); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) UpdateOrder(ctx context.Context, id int64, ou m.OrderUpdate) (m.Order, error) {
	set := bson.M{"updated_at": nowUTC()}
	if ou.UserID != nil {
		set["user_id"] = *ou.UserID
	}
	if ou.SellerID != nil {
		set["seller_id"] = *ou.SellerID
	}
	if ou.AddressID != nil {
		set["address_id"] = *ou.AddressID
	}
	if ou.Status != nil {
		set["status"] = *ou.Status
	}
	if ou.TotalAmount != nil {
		total, err := decimalToBSON(*ou.TotalAmount)
		if err != nil {
			return m.Order{}, err
		}
		set["total_amount"] = total
	}
	if len(set) == 1 {
		return m.Order{}, service.ErrNoChangesInUpdate
	}

	var doc orderDoc
	err := s.collection(ordersCollection).
		FindOneAndUpdate(ctx, bson.M{"id": id}, bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&doc)
	if notFound(err) {
		return m.Order{}, service.ErrNotFound
	}
	if err != nil {
		return m.Order{}, err
	}
	return doc.toModel()
}

func (s *Store) UpdateOrderStatus(ctx context.Context, id int64, from []m.OrderStatus, to m.OrderStatus) error {
	res, err := s.collection(ordersCollection).UpdateOne(
		ctx,
		bson.M{"id": id, "status": bson.M{"$in": from}},
		bson.M{"$set": bson.M{"status": to, "updated_at": nowUTC()}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (s *Store) LockUserCart(ctx context.Context, userID int64) error {
	if mongodriver.SessionFromContext(ctx) == nil {
		return service.ErrMustBeInTransaction
	}
	_, err := s.collection(cartLocksCollection).UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{
			"$inc":         bson.M{"lock_version": int64(1)},
			"$setOnInsert": bson.M{"user_id": userID, "created_at": nowUTC()},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *Store) DeleteOrderByID(ctx context.Context, id int64) error {
	return s.WithTransaction(ctx, func(ctx context.Context) error {
		res, err := s.collection(ordersCollection).DeleteOne(ctx, bson.M{"id": id})
		if err != nil {
			return err
		}
		if res.DeletedCount == 0 {
			return service.ErrNotFound
		}
		_, err = s.collection(orderItemsCollection).DeleteMany(ctx, bson.M{"order_id": id})
		return err
	})
}

func (s *Store) GetOrderItemByID(ctx context.Context, id int64) (m.OrderItem, error) {
	var doc orderItemDoc
	err := s.collection(orderItemsCollection).FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if notFound(err) {
		return m.OrderItem{}, service.ErrNotFound
	}
	if err != nil {
		return m.OrderItem{}, err
	}
	return doc.toModel()
}

func (s *Store) GetOrderItemsByOrderID(ctx context.Context, orderID int64) ([]m.OrderItem, error) {
	cursor, err := s.collection(orderItemsCollection).Find(ctx, bson.M{"order_id": orderID}, options.Find().SetSort(bson.D{{Key: "id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]m.OrderItem, 0)
	for cursor.Next(ctx) {
		var doc orderItemDoc
		if err = cursor.Decode(&doc); err != nil {
			return nil, err
		}
		item, err := doc.toModel()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, cursor.Err()
}

func (s *Store) CreateOrderItem(ctx context.Context, oic m.OrderItemCreate) (int64, error) {
	if _, err := s.GetOrderByID(ctx, oic.OrderID); err != nil {
		return 0, err
	}
	if _, err := s.GetProductByID(ctx, oic.ProductID); err != nil {
		return 0, err
	}
	id, err := s.nextID(ctx, orderItemsCollection)
	if err != nil {
		return 0, err
	}
	doc, err := newOrderItemDoc(id, oic)
	if err != nil {
		return 0, err
	}
	if _, err = s.collection(orderItemsCollection).InsertOne(ctx, doc); err != nil {
		return 0, mapDuplicate(err, service.ErrProductAlreadyInCart)
	}
	return id, nil
}

func (s *Store) UpdateOrderItem(ctx context.Context, id int64, oiu m.OrderItemUpdate) (m.OrderItem, error) {
	set := bson.M{}
	if oiu.OrderID != nil {
		set["order_id"] = *oiu.OrderID
	}
	if oiu.ProductID != nil {
		set["product_id"] = *oiu.ProductID
	}
	if oiu.Quantity != nil {
		set["quantity"] = *oiu.Quantity
	}
	if oiu.PriceAtPurchase != nil {
		price, err := decimalToBSON(*oiu.PriceAtPurchase)
		if err != nil {
			return m.OrderItem{}, err
		}
		set["price_at_purchase"] = price
	}
	if len(set) == 0 {
		return m.OrderItem{}, service.ErrNoChangesInUpdate
	}
	var doc orderItemDoc
	err := s.collection(orderItemsCollection).
		FindOneAndUpdate(ctx, bson.M{"id": id}, bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&doc)
	if notFound(err) {
		return m.OrderItem{}, service.ErrNotFound
	}
	if err != nil {
		return m.OrderItem{}, mapDuplicate(err, service.ErrProductAlreadyInCart)
	}
	return doc.toModel()
}

func (s *Store) UpdateOrderItemQtyIfDraft(ctx context.Context, id int64, userID int64, qty int) error {
	item, err := s.GetOrderItemByID(ctx, id)
	if err != nil {
		return err
	}
	res, err := s.collection(ordersCollection).UpdateOne(
		ctx,
		bson.M{"id": item.OrderID, "user_id": userID, "status": m.StatusDraft},
		bson.M{"$set": bson.M{"updated_at": nowUTC()}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return service.ErrNotFound
	}
	res, err = s.collection(orderItemsCollection).UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"quantity": qty}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteOrderItemByID(ctx context.Context, id int64) error {
	res, err := s.collection(orderItemsCollection).DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteOrderItemIfDraft(ctx context.Context, id int64, userID int64) error {
	item, err := s.GetOrderItemByID(ctx, id)
	if err != nil {
		return err
	}
	order, err := s.GetOrderByID(ctx, item.OrderID)
	if err != nil {
		return err
	}
	if order.UserID != userID || order.Status != m.StatusDraft {
		return service.ErrNotFound
	}
	return s.DeleteOrderItemByID(ctx, id)
}

func (s *Store) findSellerRevenueOrders(ctx context.Context, sellerID int64, dateFrom, dateTo *time.Time) ([]m.Order, error) {
	filter := bson.M{"seller_id": sellerID, "status": bson.M{"$in": revenueStatuses()}}
	if dateFrom != nil || dateTo != nil {
		created := bson.M{}
		if dateFrom != nil {
			created["$gte"] = *dateFrom
		}
		if dateTo != nil {
			created["$lte"] = *dateTo
		}
		filter["created_at"] = created
	}
	return s.findOrders(ctx, filter, m.PaginationOpts{})
}

func revenueStatuses() []m.OrderStatus {
	return []m.OrderStatus{m.StatusPaid, m.StatusShipped, m.StatusDelivered}
}

func isRevenueStatus(status m.OrderStatus) bool {
	for _, allowed := range revenueStatuses() {
		if status == allowed {
			return true
		}
	}
	return false
}

func isDuplicateOrderItem(err error) bool {
	return errors.Is(err, service.ErrProductAlreadyInCart)
}
