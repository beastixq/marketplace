package mongorepo

import (
	"context"
	"sort"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *Store) GetAdminOrders(ctx context.Context, opts m.AdminOrderListOptions) ([]m.Order, error) {
	filter := bson.M{"status": bson.M{"$ne": m.StatusDraft}}
	if opts.Status != nil {
		filter["status"] = *opts.Status
	}
	return s.findOrders(ctx, filter, opts.Pagination)
}

func (s *Store) GetPlatformStats(ctx context.Context) (m.PlatformStats, error) {
	var stats m.PlatformStats
	var err error

	stats.TotalUsers, err = s.collection(usersCollection).CountDocuments(ctx, bson.M{"deleted_at": bson.M{"$exists": false}})
	if err != nil {
		return m.PlatformStats{}, err
	}
	stats.TotalSellers, err = s.collection(sellersCollection).CountDocuments(ctx, bson.M{})
	if err != nil {
		return m.PlatformStats{}, err
	}
	stats.TotalProducts, err = s.collection(productsCollection).CountDocuments(ctx, bson.M{"deleted_at": bson.M{"$exists": false}})
	if err != nil {
		return m.PlatformStats{}, err
	}
	stats.TotalOrders, err = s.collection(ordersCollection).CountDocuments(ctx, bson.M{"status": bson.M{"$ne": m.StatusDraft}})
	if err != nil {
		return m.PlatformStats{}, err
	}
	stats.TotalReviews, err = s.collection(reviewsCollection).CountDocuments(ctx, bson.M{})
	if err != nil {
		return m.PlatformStats{}, err
	}

	orders, err := s.findOrders(ctx, bson.M{}, m.PaginationOpts{})
	if err != nil {
		return m.PlatformStats{}, err
	}
	revenue := decimal.Zero
	statusCounts := map[m.OrderStatus]int64{}
	for _, order := range orders {
		statusCounts[order.Status]++
		if isRevenueStatus(order.Status) {
			revenue = revenue.Add(order.TotalAmount)
		}
	}
	stats.TotalRevenue = revenue
	for status, count := range statusCounts {
		stats.OrdersByStatus = append(stats.OrdersByStatus, m.StatusCount{Status: status, Count: count})
	}
	sort.Slice(stats.OrdersByStatus, func(i, j int) bool { return stats.OrdersByStatus[i].Count > stats.OrdersByStatus[j].Count })

	users, err := s.GetUsers(ctx, m.UserListOptions{})
	if err != nil {
		return m.PlatformStats{}, err
	}
	roleCounts := map[m.UserRole]int64{}
	for _, user := range users {
		roleCounts[user.Role]++
	}
	for role, count := range roleCounts {
		stats.UsersByRole = append(stats.UsersByRole, m.RoleCount{Role: role, Count: count})
	}
	sort.Slice(stats.UsersByRole, func(i, j int) bool { return stats.UsersByRole[i].Count > stats.UsersByRole[j].Count })

	top, err := s.topProducts(ctx, nil, nil, 10)
	if err != nil {
		return m.PlatformStats{}, err
	}
	stats.TopProducts = top
	return stats, nil
}

func (s *Store) GetOrderDynamics(ctx context.Context, opts m.ReportOptions) ([]m.OrderDynamicsPoint, error) {
	filter := bson.M{"status": bson.M{"$ne": m.StatusDraft}}
	if opts.DateFrom != nil || opts.DateTo != nil {
		created := bson.M{}
		if opts.DateFrom != nil {
			created["$gte"] = *opts.DateFrom
		}
		if opts.DateTo != nil {
			created["$lte"] = *opts.DateTo
		}
		filter["created_at"] = created
	}
	orders, err := s.findOrders(ctx, filter, m.PaginationOpts{})
	if err != nil {
		return nil, err
	}

	pointsByPeriod := map[time.Time]*m.OrderDynamicsPoint{}
	for _, order := range orders {
		period := truncateReportPeriod(order.CreatedAt, opts.Period)
		point := pointsByPeriod[period]
		if point == nil {
			point = &m.OrderDynamicsPoint{PeriodStart: period}
			pointsByPeriod[period] = point
		}
		point.OrdersCount++
		if isRevenueStatus(order.Status) {
			point.Revenue = point.Revenue.Add(order.TotalAmount)
		}
	}
	points := make([]m.OrderDynamicsPoint, 0, len(pointsByPeriod))
	for _, point := range pointsByPeriod {
		points = append(points, *point)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].PeriodStart.Before(points[j].PeriodStart) })
	return points, nil
}

func (s *Store) GetSalesByCategory(ctx context.Context, opts m.ReportOptions) ([]m.CategorySalesStats, error) {
	orderFilter := bson.M{"status": bson.M{"$in": revenueStatuses()}}
	if opts.DateFrom != nil || opts.DateTo != nil {
		created := bson.M{}
		if opts.DateFrom != nil {
			created["$gte"] = *opts.DateFrom
		}
		if opts.DateTo != nil {
			created["$lte"] = *opts.DateTo
		}
		orderFilter["created_at"] = created
	}
	orders, err := s.findOrders(ctx, orderFilter, m.PaginationOpts{})
	if err != nil {
		return nil, err
	}

	statsByCategory := map[int64]*m.CategorySalesStats{}
	orderSeen := map[int64]map[int64]struct{}{}
	for _, order := range orders {
		items, err := s.GetOrderItemsByOrderID(ctx, order.ID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			categoryIDs, err := s.categoryIDsByProductID(ctx, item.ProductID)
			if err != nil {
				return nil, err
			}
			for _, categoryID := range categoryIDs {
				stat := statsByCategory[categoryID]
				if stat == nil {
					category, err := s.GetCategoryByID(ctx, categoryID)
					if err != nil {
						return nil, err
					}
					stat = &m.CategorySalesStats{CategoryID: category.ID, CategoryName: category.Name}
					statsByCategory[categoryID] = stat
				}
				if orderSeen[categoryID] == nil {
					orderSeen[categoryID] = map[int64]struct{}{}
				}
				if _, ok := orderSeen[categoryID][order.ID]; !ok {
					stat.OrdersCount++
					orderSeen[categoryID][order.ID] = struct{}{}
				}
				stat.UnitsSold += int64(item.Quantity)
				stat.Revenue = stat.Revenue.Add(item.PriceAtPurchase.Mul(decimal.NewFromInt(int64(item.Quantity))))
			}
		}
	}

	stats := make([]m.CategorySalesStats, 0, len(statsByCategory))
	for _, stat := range statsByCategory {
		stats = append(stats, *stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		if !stats[i].Revenue.Equal(stats[j].Revenue) {
			return stats[i].Revenue.GreaterThan(stats[j].Revenue)
		}
		return stats[i].CategoryName < stats[j].CategoryName
	})
	if opts.Limit > 0 && opts.Limit < len(stats) {
		stats = stats[:opts.Limit]
	}
	return stats, nil
}

func (s *Store) topProducts(ctx context.Context, dateFrom, dateTo *time.Time, limit int) ([]m.TopProductStats, error) {
	orderFilter := bson.M{"status": bson.M{"$in": revenueStatuses()}}
	if dateFrom != nil || dateTo != nil {
		created := bson.M{}
		if dateFrom != nil {
			created["$gte"] = *dateFrom
		}
		if dateTo != nil {
			created["$lte"] = *dateTo
		}
		orderFilter["created_at"] = created
	}
	orders, err := s.findOrders(ctx, orderFilter, m.PaginationOpts{})
	if err != nil {
		return nil, err
	}

	byProduct := map[int64]*m.TopProductStats{}
	for _, order := range orders {
		items, err := s.GetOrderItemsByOrderID(ctx, order.ID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			stat := byProduct[item.ProductID]
			if stat == nil {
				product, err := s.GetProductByID(ctx, item.ProductID)
				if err != nil {
					return nil, err
				}
				stat = &m.TopProductStats{ID: product.ID, Name: product.Name}
				byProduct[item.ProductID] = stat
			}
			stat.UnitsSold += int64(item.Quantity)
			stat.Revenue = stat.Revenue.Add(item.PriceAtPurchase.Mul(decimal.NewFromInt(int64(item.Quantity))))
		}
	}

	stats := make([]m.TopProductStats, 0, len(byProduct))
	for _, stat := range byProduct {
		stats = append(stats, *stat)
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Revenue.GreaterThan(stats[j].Revenue) })
	if limit > 0 && limit < len(stats) {
		stats = stats[:limit]
	}
	return stats, nil
}

func (s *Store) categoryIDsByProductID(ctx context.Context, productID int64) ([]int64, error) {
	cursor, err := s.collection(productCategoriesCollection).Find(ctx, bson.M{"product_id": productID}, options.Find().SetSort(bson.D{{Key: "category_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	ids := make([]int64, 0)
	for cursor.Next(ctx) {
		var link productCategoryDoc
		if err = cursor.Decode(&link); err != nil {
			return nil, err
		}
		ids = append(ids, link.CategoryID)
	}
	return ids, cursor.Err()
}

func truncateReportPeriod(t time.Time, period m.ReportPeriod) time.Time {
	t = t.UTC()
	switch period {
	case m.ReportPeriodWeek:
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := t.AddDate(0, 0, -(weekday - 1))
		return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	case m.ReportPeriodMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
}
