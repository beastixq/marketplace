package mongorepo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	usersCollection             = "users"
	sellersCollection           = "sellers"
	categoriesCollection        = "categories"
	productsCollection          = "products"
	productCategoriesCollection = "product_categories"
	addressesCollection         = "addresses"
	ordersCollection            = "orders"
	orderItemsCollection        = "order_items"
	reviewsCollection           = "reviews"
	priceHistoryCollection      = "product_price_history"
	favoritesCollection         = "product_favorites"
	countersCollection          = "counters"
	cartLocksCollection         = "cart_locks"
)

var (
	_ service.UserRepo              = (*Store)(nil)
	_ service.SellerRepo            = (*Store)(nil)
	_ service.AddressRepo           = (*Store)(nil)
	_ service.CategoryRepo          = (*Store)(nil)
	_ service.ProductRepo           = (*Store)(nil)
	_ service.ProductCategoryRepo   = (*Store)(nil)
	_ service.OrderRepo             = (*Store)(nil)
	_ service.OrderItemRepo         = (*Store)(nil)
	_ service.ReviewRepo            = (*Store)(nil)
	_ service.ReviewPurchaseChecker = (*Store)(nil)
	_ service.FavoriteRepo          = (*Store)(nil)
	_ service.BackofficeRepo        = (*Store)(nil)
	_ service.TxManager             = (*Store)(nil)
)

type Store struct {
	client *mongodriver.Client
	db     *mongodriver.Database
}

func New(ctx context.Context, uri string, dbName string) (*Store, error) {
	client, err := mongodriver.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}
	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	store := &Store{client: client, db: client.Database(dbName)}
	if err = store.EnsureIndexes(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	if err = store.EnsureSystemAccounts(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	return store, nil
}

func NewFromClient(client *mongodriver.Client, dbName string) *Store {
	return &Store{client: client, db: client.Database(dbName)}
}

func (s *Store) Close(ctx context.Context) error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Disconnect(ctx)
}

func (s *Store) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	if mongodriver.SessionFromContext(ctx) != nil {
		return fn(ctx)
	}

	session, err := s.client.StartSession()
	if err != nil {
		return fmt.Errorf("start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	if err = session.StartTransaction(); err != nil {
		return fmt.Errorf("start mongo transaction: %w", err)
	}
	txCtx := mongodriver.NewSessionContext(ctx, session)
	if err = fn(txCtx); err != nil {
		_ = session.AbortTransaction(context.Background())
		return err
	}
	if err = session.CommitTransaction(ctx); err != nil {
		return fmt.Errorf("commit mongo transaction: %w", err)
	}
	return nil
}

func (s *Store) EnsureIndexes(ctx context.Context) error {
	indexes := map[string][]mongodriver.IndexModel{
		usersCollection: {
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "role", Value: 1}}},
		},
		sellersCollection: {
			{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		categoriesCollection: {
			{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "parent_id", Value: 1}}},
		},
		productsCollection: {
			{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "seller_id", Value: 1}}},
			{Keys: bson.D{{Key: "price", Value: 1}}},
			{Keys: bson.D{{Key: "deleted_at", Value: 1}}},
		},
		productCategoriesCollection: {
			{Keys: bson.D{{Key: "product_id", Value: 1}, {Key: "category_id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "category_id", Value: 1}}},
		},
		addressesCollection: {
			{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}}},
		},
		ordersCollection: {
			{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetName("idx_orders_user_id")},
			{Keys: bson.D{{Key: "seller_id", Value: 1}}},
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}}},
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}},
				Options: options.Index().SetName("ux_orders_one_draft_per_user").SetUnique(true).SetPartialFilterExpression(bson.M{"status": m.StatusDraft}),
			},
		},
		orderItemsCollection: {
			{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "order_id", Value: 1}}},
			{Keys: bson.D{{Key: "order_id", Value: 1}, {Key: "product_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		reviewsCollection: {
			{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "product_id", Value: 1}}},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "product_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		priceHistoryCollection: {
			{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "product_id", Value: 1}, {Key: "changed_at", Value: 1}}},
		},
		favoritesCollection: {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "product_id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		},
		cartLocksCollection: {
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
	}

	for collection, models := range indexes {
		if len(models) == 0 {
			continue
		}
		if _, err := s.db.Collection(collection).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("create mongodb indexes for %s: %w", collection, err)
		}
	}
	return nil
}

func (s *Store) EnsureSystemAccounts(ctx context.Context) error {
	accounts := []userDoc{
		{
			Email:        "admin@marketplace.local",
			PasswordHash: "$2a$10$NCZo//GZesuCZdRDpb3kE.Uya2mnWx0f5m.eBxGzDCNN5YNof.V.W",
			FullName:     "System Admin",
			Role:         m.RoleAdmin,
		},
		{
			Email:        "analyst@marketplace.local",
			PasswordHash: "$2a$10$fRwTo9oql3Jwf.TDPpxpH.RCmcOkjFaOr.eSBJafHdkoCJUpyxH26",
			FullName:     "System Analyst",
			Role:         m.RoleAnalyst,
		},
	}

	for _, account := range accounts {
		if err := s.ensureSystemAccount(ctx, account); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureSystemAccount(ctx context.Context, account userDoc) error {
	count, err := s.collection(usersCollection).CountDocuments(ctx, bson.M{"email": account.Email})
	if err != nil {
		return fmt.Errorf("check system account %s: %w", account.Email, err)
	}
	if count > 0 {
		return nil
	}

	account.ID, err = s.nextID(ctx, usersCollection)
	if err != nil {
		return err
	}
	account.CreatedAt = nowUTC()
	if _, err = s.collection(usersCollection).InsertOne(ctx, account); err != nil {
		if mongodriver.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("insert system account %s: %w", account.Email, err)
	}
	return nil
}

func (s *Store) collection(name string) *mongodriver.Collection {
	return s.db.Collection(name)
}

func (s *Store) nextID(ctx context.Context, name string) (int64, error) {
	var doc struct {
		Seq int64 `bson:"seq"`
	}
	err := s.collection(countersCollection).
		FindOneAndUpdate(
			ctx,
			bson.M{"_id": name},
			bson.M{"$inc": bson.M{"seq": int64(1)}},
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
		).
		Decode(&doc)
	if err != nil {
		return 0, fmt.Errorf("next mongo id for %s: %w", name, err)
	}
	return doc.Seq, nil
}

func decimalToBSON(value decimal.Decimal) (bson.Decimal128, error) {
	d, err := bson.ParseDecimal128(value.String())
	if err != nil {
		return bson.Decimal128{}, fmt.Errorf("decimal to bson %q: %w", value.String(), err)
	}
	return d, nil
}

func decimalFromBSON(value bson.Decimal128) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(value.String())
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("decimal from bson %q: %w", value.String(), err)
	}
	return d, nil
}

func regexContains(value string) bson.M {
	return bson.M{"$regex": regexp.QuoteMeta(value)}
}

func regexContainsInsensitive(value string) bson.M {
	return bson.M{"$regex": regexp.QuoteMeta(value), "$options": "i"}
}

func notFound(err error) bool {
	return errors.Is(err, mongodriver.ErrNoDocuments)
}

func mapDuplicate(err error, sentinel error) error {
	if mongodriver.IsDuplicateKeyError(err) {
		return sentinel
	}
	return err
}

func round2(v float64) float32 {
	return float32(math.Round(v*100) / 100)
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
