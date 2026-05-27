package mongorepo

import (
	"context"
	"errors"
	"sort"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *Store) GetProducts(ctx context.Context, opts m.CatalogOptions) ([]m.Product, error) {
	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	if opts.MinPrice != nil || opts.MaxPrice != nil {
		priceFilter := bson.M{}
		if opts.MinPrice != nil {
			minPrice, err := decimalToBSON(*opts.MinPrice)
			if err != nil {
				return nil, err
			}
			priceFilter["$gte"] = minPrice
		}
		if opts.MaxPrice != nil {
			maxPrice, err := decimalToBSON(*opts.MaxPrice)
			if err != nil {
				return nil, err
			}
			priceFilter["$lte"] = maxPrice
		}
		filter["price"] = priceFilter
	}
	if opts.SellerID != nil {
		filter["seller_id"] = *opts.SellerID
	}
	if opts.FilterName != nil && *opts.FilterName != "" {
		filter["name"] = regexContains(*opts.FilterName)
	}
	if len(opts.Categories) > 0 {
		productIDs, err := s.productIDsByCategoryNames(ctx, opts.Categories)
		if err != nil {
			return nil, err
		}
		filter["id"] = bson.M{"$in": productIDs}
	}

	findOpts := options.Find()
	if opts.SortingOrder != nil {
		switch *opts.SortingOrder {
		case m.SortingOrderAsc:
			findOpts.SetSort(bson.D{{Key: "price", Value: 1}})
		case m.SortingOrderDesc:
			findOpts.SetSort(bson.D{{Key: "price", Value: -1}})
		}
	}
	if opts.Pagination != nil && opts.Pagination.Page > 0 && opts.Pagination.Limit > 0 {
		findOpts.SetSkip(int64(opts.Pagination.Limit * (opts.Pagination.Page - 1)))
		findOpts.SetLimit(int64(opts.Pagination.Limit))
	}

	cursor, err := s.collection(productsCollection).Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	products := make([]m.Product, 0)
	for cursor.Next(ctx) {
		var doc productDoc
		if err = cursor.Decode(&doc); err != nil {
			return nil, err
		}
		product, err := doc.toModel()
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, cursor.Err()
}

func (s *Store) productIDsByCategoryNames(ctx context.Context, names []string) ([]int64, error) {
	cursor, err := s.collection(categoriesCollection).Find(ctx, bson.M{"name": bson.M{"$in": names}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	categoryIDs := make([]int64, 0)
	for cursor.Next(ctx) {
		var category categoryDoc
		if err = cursor.Decode(&category); err != nil {
			return nil, err
		}
		categoryIDs = append(categoryIDs, category.ID)
	}
	if err = cursor.Err(); err != nil {
		return nil, err
	}
	if len(categoryIDs) == 0 {
		return []int64{}, nil
	}

	cursor, err = s.collection(productCategoriesCollection).Find(ctx, bson.M{"category_id": bson.M{"$in": categoryIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	seen := map[int64]struct{}{}
	for cursor.Next(ctx) {
		var link productCategoryDoc
		if err = cursor.Decode(&link); err != nil {
			return nil, err
		}
		seen[link.ProductID] = struct{}{}
	}
	if err = cursor.Err(); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Store) GetProductByID(ctx context.Context, id int64) (m.Product, error) {
	var doc productDoc
	err := s.collection(productsCollection).FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if notFound(err) {
		return m.Product{}, service.ErrNotFound
	}
	if err != nil {
		return m.Product{}, err
	}
	return doc.toModel()
}

func (s *Store) GetProductByIDForUpdate(ctx context.Context, id int64) (m.Product, error) {
	if mongodriver.SessionFromContext(ctx) == nil {
		return m.Product{}, service.ErrMustBeInTransaction
	}
	var doc productDoc
	err := s.collection(productsCollection).
		FindOneAndUpdate(
			ctx,
			bson.M{"id": id},
			bson.M{"$inc": bson.M{"lock_version": int64(1)}},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).
		Decode(&doc)
	if notFound(err) {
		return m.Product{}, service.ErrNotFound
	}
	if err != nil {
		return m.Product{}, err
	}
	return doc.toModel()
}

func (s *Store) GetProductPriceHistory(ctx context.Context, pid int64, dateFrom, dateTo time.Time) ([]m.ProductPriceHistory, error) {
	cursor, err := s.collection(priceHistoryCollection).Find(
		ctx,
		bson.M{"product_id": pid, "changed_at": bson.M{"$gte": dateFrom, "$lte": dateTo}},
		options.Find().SetSort(bson.D{{Key: "changed_at", Value: 1}, {Key: "id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	history := make([]m.ProductPriceHistory, 0)
	for cursor.Next(ctx) {
		var doc priceHistoryDoc
		if err = cursor.Decode(&doc); err != nil {
			return nil, err
		}
		item, err := doc.toModel()
		if err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	return history, cursor.Err()
}

func (s *Store) CreateProduct(ctx context.Context, pc m.ProductCreate) (int64, error) {
	if _, err := s.GetSellerByID(ctx, pc.SellerID); err != nil {
		return 0, err
	}
	id, err := s.nextID(ctx, productsCollection)
	if err != nil {
		return 0, err
	}
	doc, err := newProductDoc(id, pc)
	if err != nil {
		return 0, err
	}
	if _, err = s.collection(productsCollection).InsertOne(ctx, doc); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) UpdateProduct(ctx context.Context, id int64, pu m.ProductUpdate) (m.Product, error) {
	if pu.SellerID == nil && pu.Name == nil && pu.Description == nil && pu.Price == nil && pu.StockQuantity == nil {
		return m.Product{}, service.ErrNoChangesInUpdate
	}

	var updated m.Product
	err := s.WithTransaction(ctx, func(ctx context.Context) error {
		oldProduct, err := s.GetProductByID(ctx, id)
		if err != nil {
			return err
		}

		set := bson.M{}
		if pu.SellerID != nil {
			if _, err := s.GetSellerByID(ctx, *pu.SellerID); err != nil {
				return err
			}
			set["seller_id"] = *pu.SellerID
		}
		if pu.Name != nil {
			set["name"] = *pu.Name
		}
		if pu.Description != nil {
			set["description"] = *pu.Description
		}
		if pu.Price != nil {
			price, err := decimalToBSON(*pu.Price)
			if err != nil {
				return err
			}
			set["price"] = price
		}
		if pu.StockQuantity != nil {
			if *pu.StockQuantity < oldProduct.ReservedQuantity {
				return service.ErrStockBelowReserved
			}
			set["stock_quantity"] = *pu.StockQuantity
		}

		filter := bson.M{"id": id}
		if pu.StockQuantity != nil {
			filter["reserved_quantity"] = bson.M{"$lte": *pu.StockQuantity}
		}

		var doc productDoc
		err = s.collection(productsCollection).
			FindOneAndUpdate(ctx, filter, bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).
			Decode(&doc)
		if notFound(err) {
			if pu.StockQuantity != nil {
				return service.ErrStockBelowReserved
			}
			return service.ErrNotFound
		}
		if err != nil {
			return err
		}
		updated, err = doc.toModel()
		if err != nil {
			return err
		}
		if pu.Price != nil && !pu.Price.Equal(oldProduct.Price) {
			changedBy := "mongodb"
			if pu.ChangedBy != nil {
				changedBy = *pu.ChangedBy
			}
			historyID, err := s.nextID(ctx, priceHistoryCollection)
			if err != nil {
				return err
			}
			history, err := newPriceHistoryDoc(historyID, id, oldProduct.Price, *pu.Price, changedBy)
			if err != nil {
				return err
			}
			if _, err = s.collection(priceHistoryCollection).InsertOne(ctx, history); err != nil {
				return err
			}
		}
		return nil
	})
	return updated, err
}

func (s *Store) ChangeStockAndReserved(ctx context.Context, productID int64, stockDelta int, reservedDelta int) error {
	filter := bson.M{
		"id": productID,
		"$expr": bson.M{"$and": bson.A{
			bson.M{"$gte": bson.A{bson.M{"$add": bson.A{"$stock_quantity", stockDelta}}, 0}},
			bson.M{"$gte": bson.A{bson.M{"$add": bson.A{"$reserved_quantity", reservedDelta}}, 0}},
			bson.M{"$lte": bson.A{
				bson.M{"$add": bson.A{"$reserved_quantity", reservedDelta}},
				bson.M{"$add": bson.A{"$stock_quantity", stockDelta}},
			}},
		}},
	}
	res, err := s.collection(productsCollection).UpdateOne(ctx, filter, bson.M{
		"$inc": bson.M{
			"stock_quantity":    stockDelta,
			"reserved_quantity": reservedDelta,
		},
	})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		if _, err := s.GetProductByID(ctx, productID); errors.Is(err, service.ErrNotFound) {
			return service.ErrNotFound
		}
		return service.ErrStockInvariantViolated
	}
	return nil
}

func (s *Store) DeleteProductByID(ctx context.Context, id int64) error {
	res, err := s.collection(productsCollection).UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"deleted_at": nowUTC()}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (s *Store) GetProductCategories(ctx context.Context, productID int64) ([]m.Category, error) {
	cursor, err := s.collection(productCategoriesCollection).Find(ctx, bson.M{"product_id": productID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	categoryIDs := make([]int64, 0)
	for cursor.Next(ctx) {
		var link productCategoryDoc
		if err = cursor.Decode(&link); err != nil {
			return nil, err
		}
		categoryIDs = append(categoryIDs, link.CategoryID)
	}
	if err = cursor.Err(); err != nil {
		return nil, err
	}
	if len(categoryIDs) == 0 {
		return []m.Category{}, nil
	}
	cursor, err = s.collection(categoriesCollection).Find(
		ctx,
		bson.M{"id": bson.M{"$in": categoryIDs}},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}, {Key: "id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	categories := make([]m.Category, 0)
	for cursor.Next(ctx) {
		var doc categoryDoc
		if err = cursor.Decode(&doc); err != nil {
			return nil, err
		}
		categories = append(categories, doc.toModel())
	}
	return categories, cursor.Err()
}

func (s *Store) ReplaceProductCategories(ctx context.Context, productID int64, categoryIDs []int64) error {
	return s.WithTransaction(ctx, func(ctx context.Context) error {
		if _, err := s.GetProductByID(ctx, productID); err != nil {
			return err
		}

		categoryIDs = uniqueInt64s(categoryIDs)
		for _, categoryID := range categoryIDs {
			if _, err := s.GetCategoryByID(ctx, categoryID); err != nil {
				if errors.Is(err, service.ErrNotFound) {
					return service.ErrCategoryNotFound
				}
				return err
			}
		}

		if _, err := s.collection(productCategoriesCollection).DeleteMany(ctx, bson.M{"product_id": productID}); err != nil {
			return err
		}
		if len(categoryIDs) == 0 {
			return nil
		}
		docs := make([]any, 0, len(categoryIDs))
		for _, categoryID := range categoryIDs {
			docs = append(docs, productCategoryDoc{ProductID: productID, CategoryID: categoryID})
		}
		_, err := s.collection(productCategoriesCollection).InsertMany(ctx, docs)
		return err
	})
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	unique := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
	return unique
}
