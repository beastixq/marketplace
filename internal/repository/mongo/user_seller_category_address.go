package mongorepo

import (
	"context"
	"fmt"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *Store) GetUsers(ctx context.Context, opts m.UserListOptions) ([]m.User, error) {
	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	if opts.Role != nil {
		filter["role"] = *opts.Role
	}
	if opts.Search != nil && *opts.Search != "" {
		filter["$or"] = bson.A{
			bson.M{"email": regexContainsInsensitive(*opts.Search)},
			bson.M{"full_name": regexContainsInsensitive(*opts.Search)},
		}
	}
	findOpts := options.Find().SetSort(bson.D{{Key: "id", Value: 1}})
	if opts.Pagination.Page > 0 && opts.Pagination.Limit > 0 {
		findOpts.SetSkip(int64((opts.Pagination.Page - 1) * opts.Pagination.Limit))
		findOpts.SetLimit(int64(opts.Pagination.Limit))
	}
	cursor, err := s.collection(usersCollection).Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	users := make([]m.User, 0)
	for cursor.Next(ctx) {
		var doc userDoc
		if err = cursor.Decode(&doc); err != nil {
			return nil, err
		}
		users = append(users, doc.toModel())
	}
	return users, cursor.Err()
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (m.User, error) {
	var doc userDoc
	err := s.collection(usersCollection).FindOne(ctx, bson.M{"id": id, "deleted_at": bson.M{"$exists": false}}).Decode(&doc)
	if notFound(err) {
		return m.User{}, service.ErrNotFound
	}
	if err != nil {
		return m.User{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (m.User, error) {
	var doc userDoc
	err := s.collection(usersCollection).FindOne(ctx, bson.M{"email": email, "deleted_at": bson.M{"$exists": false}}).Decode(&doc)
	if notFound(err) {
		return m.User{}, service.ErrNotFound
	}
	if err != nil {
		return m.User{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) CreateUser(ctx context.Context, uc m.UserCreate) (int64, error) {
	id, err := s.nextID(ctx, usersCollection)
	if err != nil {
		return 0, err
	}
	doc := userDoc{
		ID:           id,
		Email:        uc.Email,
		PasswordHash: uc.Password,
		FullName:     uc.FullName,
		Phone:        uc.Phone,
		Role:         uc.Role,
		CreatedAt:    nowUTC(),
	}
	if _, err = s.collection(usersCollection).InsertOne(ctx, doc); err != nil {
		return 0, mapDuplicate(err, service.ErrAccountWithEmailAlreadyExists)
	}
	return id, nil
}

func (s *Store) UpdateUser(ctx context.Context, id int64, uu m.UserUpdate) (m.User, error) {
	set := bson.M{}
	if uu.Email != nil {
		set["email"] = *uu.Email
	}
	if uu.FullName != nil {
		set["full_name"] = *uu.FullName
	}
	if uu.Phone != nil {
		set["phone"] = *uu.Phone
	}
	if uu.Role != nil {
		set["role"] = *uu.Role
	}
	if len(set) == 0 {
		return m.User{}, service.ErrNoChangesInUpdate
	}

	var doc userDoc
	err := s.collection(usersCollection).
		FindOneAndUpdate(
			ctx,
			bson.M{"id": id, "deleted_at": bson.M{"$exists": false}},
			bson.M{"$set": set},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).
		Decode(&doc)
	if notFound(err) {
		return m.User{}, service.ErrNotFound
	}
	if err != nil {
		if mongodriver.IsDuplicateKeyError(err) {
			return m.User{}, fmt.Errorf("users_email_key: %w", err)
		}
		return m.User{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) ChangePasswordUser(ctx context.Context, id int64, newPassHash string) error {
	res, err := s.collection(usersCollection).UpdateOne(
		ctx,
		bson.M{"id": id, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"password_hash": newPassHash}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteUserByID(ctx context.Context, id int64) error {
	res, err := s.collection(usersCollection).UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"deleted_at": nowUTC()}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (s *Store) GetSellerByID(ctx context.Context, id int64) (m.Seller, error) {
	var doc sellerDoc
	err := s.collection(sellersCollection).FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if notFound(err) {
		return m.Seller{}, service.ErrNotFound
	}
	if err != nil {
		return m.Seller{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) GetSellerByUserID(ctx context.Context, userID int64) (m.Seller, error) {
	var doc sellerDoc
	err := s.collection(sellersCollection).FindOne(ctx, bson.M{"user_id": userID}).Decode(&doc)
	if notFound(err) {
		return m.Seller{}, service.ErrNotFound
	}
	if err != nil {
		return m.Seller{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) GetSellerStats(ctx context.Context, sellerID int64, dateFrom, dateTo time.Time) (m.SellerStats, error) {
	orders, err := s.findSellerRevenueOrders(ctx, sellerID, &dateFrom, &dateTo)
	if err != nil {
		return m.SellerStats{}, err
	}

	totalRevenue := decimal.Zero
	productUnits := make(map[int64]int64)
	for _, order := range orders {
		totalRevenue = totalRevenue.Add(order.TotalAmount)
		items, err := s.GetOrderItemsByOrderID(ctx, order.ID)
		if err != nil {
			return m.SellerStats{}, err
		}
		for _, item := range items {
			productUnits[item.ProductID] += int64(item.Quantity)
		}
	}

	topName := ""
	var topProductID int64
	var topUnits int64
	for productID, units := range productUnits {
		if units > topUnits || topProductID == 0 {
			topProductID = productID
			topUnits = units
		}
	}
	if topProductID != 0 {
		if product, err := s.GetProductByID(ctx, topProductID); err == nil {
			topName = product.Name
		}
	}

	avg := decimal.Zero
	if len(orders) > 0 {
		avg = totalRevenue.Div(decimal.NewFromInt(int64(len(orders))))
	}
	return m.SellerStats{
		TotalOrders:    int64(len(orders)),
		TotalRevenue:   totalRevenue,
		AvgOrderValue:  avg,
		TopProductName: topName,
	}, nil
}

func (s *Store) CreateSeller(ctx context.Context, sc m.SellerCreate) (int64, error) {
	if _, err := s.GetUserByID(ctx, sc.UserID); err != nil {
		return 0, err
	}
	id, err := s.nextID(ctx, sellersCollection)
	if err != nil {
		return 0, err
	}
	doc := sellerDoc{
		ID:          id,
		UserID:      sc.UserID,
		CompanyName: sc.CompanyName,
		Description: sc.Description,
		Rating:      sc.Rating,
		CreatedAt:   nowUTC(),
	}
	if _, err = s.collection(sellersCollection).InsertOne(ctx, doc); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) UpdateSeller(ctx context.Context, id int64, su m.SellerUpdate) (m.Seller, error) {
	set := bson.M{}
	if su.UserID != nil {
		set["user_id"] = *su.UserID
	}
	if su.CompanyName != nil {
		set["company_name"] = *su.CompanyName
	}
	if su.Description != nil {
		set["description"] = *su.Description
	}
	if su.Rating != nil {
		set["rating"] = *su.Rating
	}
	if len(set) == 0 {
		return m.Seller{}, service.ErrNoChangesInUpdate
	}

	var doc sellerDoc
	err := s.collection(sellersCollection).
		FindOneAndUpdate(ctx, bson.M{"id": id}, bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&doc)
	if notFound(err) {
		return m.Seller{}, service.ErrNotFound
	}
	if err != nil {
		return m.Seller{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) DeleteSellerByID(ctx context.Context, id int64) error {
	res, err := s.collection(sellersCollection).DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (s *Store) GetCategories(ctx context.Context, opts m.CategoryListOptions) ([]m.Category, error) {
	filter := bson.M{}
	if opts.OnlyRoot {
		filter["parent_id"] = bson.M{"$exists": false}
	}
	if opts.ParentID != nil {
		filter["parent_id"] = *opts.ParentID
	}
	if opts.Search != nil && *opts.Search != "" {
		filter["$or"] = bson.A{
			bson.M{"name": regexContainsInsensitive(*opts.Search)},
			bson.M{"description": regexContainsInsensitive(*opts.Search)},
		}
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}, {Key: "id", Value: 1}})
	if opts.Pagination.Page > 0 && opts.Pagination.Limit > 0 {
		findOpts.SetSkip(int64((opts.Pagination.Page - 1) * opts.Pagination.Limit))
		findOpts.SetLimit(int64(opts.Pagination.Limit))
	}
	cursor, err := s.collection(categoriesCollection).Find(ctx, filter, findOpts)
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

func (s *Store) GetCategoryByID(ctx context.Context, id int64) (m.Category, error) {
	var doc categoryDoc
	err := s.collection(categoriesCollection).FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if notFound(err) {
		return m.Category{}, service.ErrNotFound
	}
	if err != nil {
		return m.Category{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) CreateCategory(ctx context.Context, cc m.CategoryCreate) (int64, error) {
	if cc.ParentID != nil {
		if _, err := s.GetCategoryByID(ctx, *cc.ParentID); err != nil {
			return 0, err
		}
	}
	id, err := s.nextID(ctx, categoriesCollection)
	if err != nil {
		return 0, err
	}
	doc := categoryDoc{ID: id, ParentID: cc.ParentID, Name: cc.Name, Description: cc.Description}
	if _, err = s.collection(categoriesCollection).InsertOne(ctx, doc); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) UpdateCategory(ctx context.Context, id int64, cu m.CategoryUpdate) (m.Category, error) {
	set := bson.M{}
	if cu.ParentID != nil {
		set["parent_id"] = *cu.ParentID
	}
	if cu.Name != nil {
		set["name"] = *cu.Name
	}
	if cu.Description != nil {
		set["description"] = *cu.Description
	}
	if len(set) == 0 {
		return m.Category{}, service.ErrNoChangesInUpdate
	}
	var doc categoryDoc
	err := s.collection(categoriesCollection).
		FindOneAndUpdate(ctx, bson.M{"id": id}, bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&doc)
	if notFound(err) {
		return m.Category{}, service.ErrNotFound
	}
	if err != nil {
		return m.Category{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) DeleteCategoryByID(ctx context.Context, id int64) error {
	return s.WithTransaction(ctx, func(ctx context.Context) error {
		res, err := s.collection(categoriesCollection).DeleteOne(ctx, bson.M{"id": id})
		if err != nil {
			return err
		}
		if res.DeletedCount == 0 {
			return service.ErrNotFound
		}
		if _, err = s.collection(productCategoriesCollection).DeleteMany(ctx, bson.M{"category_id": id}); err != nil {
			return err
		}
		_, err = s.collection(categoriesCollection).UpdateMany(ctx, bson.M{"parent_id": id}, bson.M{"$unset": bson.M{"parent_id": ""}})
		return err
	})
}

func (s *Store) GetAddressByID(ctx context.Context, id int64) (m.Address, error) {
	var doc addressDoc
	err := s.collection(addressesCollection).FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if notFound(err) {
		return m.Address{}, service.ErrNotFound
	}
	if err != nil {
		return m.Address{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) GetAddressesByUserID(ctx context.Context, userID int64) ([]m.Address, error) {
	cursor, err := s.collection(addressesCollection).Find(ctx, bson.M{"user_id": userID}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	addresses := make([]m.Address, 0)
	for cursor.Next(ctx) {
		var doc addressDoc
		if err = cursor.Decode(&doc); err != nil {
			return nil, err
		}
		addresses = append(addresses, doc.toModel())
	}
	return addresses, cursor.Err()
}

func (s *Store) CreateAddress(ctx context.Context, ac m.AddressCreate) (int64, error) {
	if _, err := s.GetUserByID(ctx, ac.UserID); err != nil {
		return 0, err
	}
	id, err := s.nextID(ctx, addressesCollection)
	if err != nil {
		return 0, err
	}
	doc := addressDoc{
		ID:        id,
		UserID:    ac.UserID,
		City:      ac.City,
		Street:    ac.Street,
		House:     ac.House,
		ZipCode:   ac.ZipCode,
		IsDefault: ac.IsDefault,
		CreatedAt: nowUTC(),
	}
	if _, err = s.collection(addressesCollection).InsertOne(ctx, doc); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) UpdateAddress(ctx context.Context, id int64, au m.AddressUpdate) (m.Address, error) {
	set := bson.M{}
	if au.UserID != nil {
		set["user_id"] = *au.UserID
	}
	if au.City != nil {
		set["city"] = *au.City
	}
	if au.Street != nil {
		set["street"] = *au.Street
	}
	if au.House != nil {
		set["house"] = *au.House
	}
	if au.ZipCode != nil {
		set["zip_code"] = *au.ZipCode
	}
	if au.IsDefault != nil {
		set["is_default"] = *au.IsDefault
	}
	if len(set) == 0 {
		return m.Address{}, service.ErrNoChangesInUpdate
	}
	var doc addressDoc
	err := s.collection(addressesCollection).
		FindOneAndUpdate(ctx, bson.M{"id": id}, bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&doc)
	if notFound(err) {
		return m.Address{}, service.ErrNotFound
	}
	if err != nil {
		return m.Address{}, err
	}
	return doc.toModel(), nil
}

func (s *Store) DeleteAddressByID(ctx context.Context, id int64) error {
	res, err := s.collection(addressesCollection).DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return service.ErrNotFound
	}
	return nil
}
