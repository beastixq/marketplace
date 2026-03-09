package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRepoImpl struct {
	pool *pgxpool.Pool
}

func NewProductRepo(pool *pgxpool.Pool) ProductRepoImpl {
	return ProductRepoImpl{pool: pool}
}

func (pr ProductRepoImpl) GetProducts(ctx context.Context, options m.CatalogOptions) (ps []m.Product, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	products := psql.Select("id", "seller_id", "name", "description", "price", "stock_quantity", "created_at", "deleted_at").Distinct().From("products").Where(sq.Eq{"deleted_at": nil})
	if options.MaxPrice != nil && options.MinPrice != nil {
		products = products.Where(sq.And{sq.GtOrEq{"price": options.MinPrice}, sq.LtOrEq{"price": options.MaxPrice}})
	} else if options.MinPrice != nil {
		products = products.Where(sq.GtOrEq{"price": options.MinPrice})
	} else if options.MaxPrice != nil {
		products = products.Where(sq.LtOrEq{"price": options.MaxPrice})
	}
	if options.FilterName != nil {
		products = products.Where("name LIKE ?", fmt.Sprint("%", *options.FilterName, "%"))
	}
	if options.Categories != nil {
		products = products.Join("product_categories ON products.id = product_categories.product_id")
		products = products.Join("categories ON product_categories.category_id = categories.id")
		products = products.Where(sq.Eq{"categories.name": options.Categories})
	}
	if options.SortingOrder != nil {
		switch *options.SortingOrder {
		case m.SortingOrderAsc:
			products = products.OrderBy("price ASC")
		case m.SortingOrderDesc:
			products = products.OrderBy("price DESC")
		}
	}
	if options.Pagination != nil {
		products = products.Offset(uint64(options.Pagination.Limit * options.Pagination.Page))
		products = products.Limit(uint64(options.Pagination.Limit))
	}
	sql, args, err := products.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := pr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()
	ps = make([]m.Product, 0)
	var prow productRow
	for rows.Next() {
		err = rows.Scan(&prow.ID, &prow.SellerID, &prow.Name, &prow.Description, &prow.Price, &prow.StockQuantity, &prow.CreatedAt, &prow.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		ps = append(ps, prow.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return ps, nil
}

func (pr ProductRepoImpl) GetProductByID(ctx context.Context, id int64) (p m.Product, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "seller_id", "name", "description", "price", "stock_quantity", "created_at", "deleted_at").From("products").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return m.Product{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := pr.pool.QueryRow(ctx, sql, args...)
	var product productRow
	if err = row.Scan(&product.ID, &product.SellerID, &product.Name, &product.Description, &product.Price, &product.StockQuantity, &product.CreatedAt, &product.DeletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.Product{}, service.ErrNotFound
		}
		return m.Product{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return product.toModel(), nil
}

func (pr ProductRepoImpl) GetProductPriceHistory(ctx context.Context, pid int64, dateFrom time.Time, dateTo time.Time) (ph []m.ProductPriceHistory, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sb := psql.Select("id", "product_id", "old_price", "new_price", "changed_at", "changed_by").From("product_price_history").Where(sq.Eq{"product_id": pid})
	sb = sb.Where(sq.And{sq.GtOrEq{"product_price_history.changed_at": dateFrom}, sq.LtOrEq{"product_price_history.changed_at": dateTo}})
	sql, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := pr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()

	ph = make([]m.ProductPriceHistory, 0)
	var hr productPriceHistoryRow
	for rows.Next() {
		if err = rows.Scan(&hr.ID, &hr.ProductID, &hr.OldPrice, &hr.NewPrice, &hr.ChangedAt, &hr.ChangedBy); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		ph = append(ph, hr.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return ph, nil
}

func (pr ProductRepoImpl) CreateProduct(ctx context.Context, pc m.ProductCreate) (id int64, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Insert("products").Columns("seller_id", "name", "description", "price", "stock_quantity").Values(pc.SellerID, pc.Name, pc.Description, pc.Price, pc.StockQuantity).Suffix("RETURNING id").ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := pr.pool.QueryRow(ctx, sql, args...)
	if err = row.Scan(&id); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return id, nil
}

func (pr ProductRepoImpl) UpdateProduct(ctx context.Context, id int64, pu m.ProductUpdate) (p m.Product, err error) {
	if pu.SellerID == nil && pu.Name == nil && pu.Description == nil && pu.Price == nil && pu.StockQuantity == nil {
		return m.Product{}, service.ErrNoChangesInUpdate
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("products").Where(sq.Eq{"id": id})
	if pu.SellerID != nil {
		ub = ub.Set("seller_id", *pu.SellerID)
	}
	if pu.Name != nil {
		ub = ub.Set("name", *pu.Name)
	}
	if pu.Description != nil {
		ub = ub.Set("description", *pu.Description)
	}
	if pu.Price != nil {
		ub = ub.Set("price", *pu.Price)
	}
	if pu.StockQuantity != nil {
		ub = ub.Set("stock_quantity", *pu.StockQuantity)
	}
	ub = ub.Suffix("RETURNING id, seller_id, name, description, price, stock_quantity, created_at, deleted_at")
	sql, args, err := ub.ToSql()
	if err != nil {
		return m.Product{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := pr.pool.QueryRow(ctx, sql, args...)
	var product productRow
	if err = row.Scan(&product.ID, &product.SellerID, &product.Name, &product.Description, &product.Price, &product.StockQuantity, &product.CreatedAt, &product.DeletedAt); err != nil {
		return m.Product{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return product.toModel(), nil
}

func (pr ProductRepoImpl) DeleteProductByID(ctx context.Context, id int64) (err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Update("products").Where(sq.Eq{"id": id}).Set("deleted_at", sq.Expr("NOW()")).ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = pr.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}
