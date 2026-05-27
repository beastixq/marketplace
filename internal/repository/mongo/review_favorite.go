package mongorepo

import (
	"context"
	"errors"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *Store) GetReviewByID(ctx context.Context, id int64) (m.Review, error) {
	var doc reviewDoc
	err := s.collection(reviewsCollection).FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if notFound(err) {
		return m.Review{}, service.ErrNotFound
	}
	if err != nil {
		return m.Review{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) GetReviewsByProductID(ctx context.Context, pid int64, opts m.PaginationOpts) ([]m.Review, error) {
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "id", Value: -1}})
	if opts.Page > 0 && opts.Limit > 0 {
		findOpts.SetSkip(int64((opts.Page - 1) * opts.Limit))
		findOpts.SetLimit(int64(opts.Limit))
	}
	cursor, err := s.collection(reviewsCollection).Find(ctx, bson.M{"product_id": pid}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	reviews := make([]m.Review, 0)
	for cursor.Next(ctx) {
		var doc reviewDoc
		if err = cursor.Decode(&doc); err != nil {
			return nil, err
		}
		reviews = append(reviews, doc.toModel())
	}
	return reviews, cursor.Err()
}

func (s *Store) UserPurchasedProduct(ctx context.Context, userID int64, productID int64) (bool, error) {
	cursor, err := s.collection(ordersCollection).Find(ctx, bson.M{
		"user_id": userID,
		"status":  bson.M{"$in": revenueStatuses()},
	})
	if err != nil {
		return false, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var order orderDoc
		if err = cursor.Decode(&order); err != nil {
			return false, err
		}
		count, err := s.collection(orderItemsCollection).CountDocuments(ctx, bson.M{"order_id": order.ID, "product_id": productID})
		if err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, cursor.Err()
}

func (s *Store) CreateReview(ctx context.Context, rc m.ReviewCreate) (int64, error) {
	var id int64
	err := s.WithTransaction(ctx, func(ctx context.Context) error {
		var err error
		id, err = s.nextID(ctx, reviewsCollection)
		if err != nil {
			return err
		}
		doc := reviewDoc{
			ID:        id,
			UserID:    rc.UserID,
			ProductID: rc.ProductID,
			Rating:    rc.Rating,
			Comment:   rc.Comment,
			CreatedAt: nowUTC(),
		}
		if _, err = s.collection(reviewsCollection).InsertOne(ctx, doc); err != nil {
			return mapDuplicate(err, service.ErrReviewAlreadyExists)
		}
		return s.recalculateRatings(ctx, rc.ProductID)
	})
	return id, err
}

func (s *Store) UpdateReview(ctx context.Context, id int64, ru m.ReviewUpdate) (m.Review, error) {
	if ru.Rating == nil && ru.Comment == nil {
		return m.Review{}, service.ErrNoChangesInUpdate
	}

	var review m.Review
	err := s.WithTransaction(ctx, func(ctx context.Context) error {
		set := bson.M{}
		if ru.Rating != nil {
			set["rating"] = *ru.Rating
		}
		if ru.Comment != nil {
			set["comment"] = *ru.Comment
		}
		var doc reviewDoc
		err := s.collection(reviewsCollection).
			FindOneAndUpdate(ctx, bson.M{"id": id}, bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).
			Decode(&doc)
		if notFound(err) {
			return service.ErrNotFound
		}
		if err != nil {
			return err
		}
		review = doc.toModel()
		return s.recalculateRatings(ctx, doc.ProductID)
	})
	return review, err
}

func (s *Store) DeleteReviewByID(ctx context.Context, id int64) error {
	return s.WithTransaction(ctx, func(ctx context.Context) error {
		review, err := s.GetReviewByID(ctx, id)
		if err != nil {
			return err
		}
		res, err := s.collection(reviewsCollection).DeleteOne(ctx, bson.M{"id": id})
		if err != nil {
			return err
		}
		if res.DeletedCount == 0 {
			return service.ErrNotFound
		}
		return s.recalculateRatings(ctx, review.ProductID)
	})
}

func (s *Store) recalculateRatings(ctx context.Context, productID int64) error {
	product, err := s.GetProductByID(ctx, productID)
	if err != nil {
		return err
	}

	cursor, err := s.collection(reviewsCollection).Find(ctx, bson.M{"product_id": productID})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var sum int
	var count int
	for cursor.Next(ctx) {
		var review reviewDoc
		if err = cursor.Decode(&review); err != nil {
			return err
		}
		sum += int(review.Rating)
		count++
	}
	if err = cursor.Err(); err != nil {
		return err
	}

	update := bson.M{"$unset": bson.M{"rating": ""}}
	if count > 0 {
		update = bson.M{"$set": bson.M{"rating": round2(float64(sum) / float64(count))}}
	}
	if _, err = s.collection(productsCollection).UpdateOne(ctx, bson.M{"id": productID}, update); err != nil {
		return err
	}
	return s.recalculateSellerRating(ctx, product.SellerID)
}

func (s *Store) recalculateSellerRating(ctx context.Context, sellerID int64) error {
	cursor, err := s.collection(productsCollection).Find(ctx, bson.M{
		"seller_id":  sellerID,
		"deleted_at": bson.M{"$exists": false},
		"rating":     bson.M{"$exists": true},
	})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var sum float64
	var count int
	for cursor.Next(ctx) {
		var product productDoc
		if err = cursor.Decode(&product); err != nil {
			return err
		}
		if product.Rating != nil {
			sum += float64(*product.Rating)
			count++
		}
	}
	if err = cursor.Err(); err != nil {
		return err
	}

	update := bson.M{"$unset": bson.M{"rating": ""}}
	if count > 0 {
		update = bson.M{"$set": bson.M{"rating": round2(sum / float64(count))}}
	}
	_, err = s.collection(sellersCollection).UpdateOne(ctx, bson.M{"id": sellerID}, update)
	return err
}

func (s *Store) AddFavorite(ctx context.Context, userID int64, productID int64) (bool, error) {
	if _, err := s.GetProductByID(ctx, productID); err != nil {
		return false, err
	}
	_, err := s.collection(favoritesCollection).InsertOne(ctx, favoriteDoc{
		UserID:    userID,
		ProductID: productID,
		CreatedAt: nowUTC(),
	})
	if err != nil {
		if mongodriver.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Store) DeleteFavorite(ctx context.Context, userID int64, productID int64) error {
	_, err := s.collection(favoritesCollection).DeleteOne(ctx, bson.M{"user_id": userID, "product_id": productID})
	return err
}

func (s *Store) ListFavoriteProductsByUserID(ctx context.Context, userID int64, opts m.PaginationOpts) ([]m.Product, error) {
	cursor, err := s.collection(favoritesCollection).Find(
		ctx,
		bson.M{"user_id": userID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "product_id", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	products := make([]m.Product, 0)
	for cursor.Next(ctx) {
		var favorite favoriteDoc
		if err = cursor.Decode(&favorite); err != nil {
			return nil, err
		}
		product, err := s.GetProductByID(ctx, favorite.ProductID)
		if errors.Is(err, service.ErrNotFound) || (err == nil && product.DeletedAt != nil) {
			continue
		}
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	if err = cursor.Err(); err != nil {
		return nil, err
	}
	if opts.Page > 0 && opts.Limit > 0 {
		start := (opts.Page - 1) * opts.Limit
		if start >= len(products) {
			return []m.Product{}, nil
		}
		end := start + opts.Limit
		if end > len(products) {
			end = len(products)
		}
		products = products[start:end]
	}
	return products, nil
}

func (s *Store) IsFavorite(ctx context.Context, userID int64, productID int64) (bool, error) {
	count, err := s.collection(favoritesCollection).CountDocuments(ctx, bson.M{"user_id": userID, "product_id": productID})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
