package generators

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func weightedChoice(items []string, weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	for i, w := range weights {
		r -= w
		if r < 0 {
			return items[i]
		}
	}
	return items[len(items)-1]
}

type SeedOrder struct {
	SellerID int64
	Status   string
}

func (so SeedOrder) IsDraft() bool {
	return so.Status == "draft"
}

type SeedOrders map[int64]SeedOrder

type orderHolidayBoost struct {
	month      time.Month
	day        int
	radiusDays int
	boost      float64
}

var russianOrderHolidayBoosts = []orderHolidayBoost{
	{month: time.January, day: 1, radiusDays: 10, boost: 5.0},
	{month: time.January, day: 7, radiusDays: 4, boost: 2.0},
	{month: time.February, day: 14, radiusDays: 5, boost: 3.0},
	{month: time.February, day: 23, radiusDays: 4, boost: 2.5},
	{month: time.March, day: 8, radiusDays: 7, boost: 4.0},
	{month: time.May, day: 1, radiusDays: 3, boost: 1.8},
	{month: time.May, day: 9, radiusDays: 4, boost: 2.0},
	{month: time.September, day: 1, radiusDays: 5, boost: 1.6},
	{month: time.November, day: 4, radiusDays: 3, boost: 1.4},
	{month: time.December, day: 31, radiusDays: 14, boost: 5.0},
}

func CreateOrders(tx pgx.Tx, ctx context.Context, buyerIDs, addressesIDs []int64, sellersWithProducts []int64, count int, dateFrom, dateTo time.Time) (ordersIDs []int64, seedOrders SeedOrders, err error) {
	createdInLoop := 0
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("orders").Columns("user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at")
	ordersIDs = make([]int64, count)
	sellerPerIndex := make([]int64, count) // 0 means draft (no seller)
	statusPerIndex := make([]string, count)
	usedDraftBuyers := make(map[int64]struct{})

	createdAtPerIndex, err := distributedOrderDates(count, dateFrom, dateTo)
	if err != nil {
		return nil, nil, fmt.Errorf("Orders invalid date range: %v", err)
	}

	for i := range count {
		createdAt := createdAtPerIndex[i]
		status := randomSeedOrderStatus(createdAt, dateTo)
		buyerID, ok := randomSeedOrderBuyer(buyerIDs, status, usedDraftBuyers)
		if !ok {
			status = "pending"
			buyerID = buyerIDs[rand.Intn(len(buyerIDs))]
		}
		statusPerIndex[i] = status

		var addressID interface{}
		var sellerID interface{}
		var totalAmount float64

		if status == "draft" {
			addressID = nil
			sellerID = nil
			totalAmount = 0
		} else {
			addressID = addressesIDs[rand.Intn(len(addressesIDs))]
			sid := sellersWithProducts[rand.Intn(len(sellersWithProducts))]
			sellerID = sid
			sellerPerIndex[i] = sid
			totalAmount = 0
		}

		insertBuilder = insertBuilder.Values(buyerID, addressID, sellerID, status, totalAmount, createdAt, seedOrderUpdatedAt(createdAt, status, dateTo))
		createdInLoop++

		if i%10 == 9 || i == count-1 {
			sql, args, err := insertBuilder.Suffix("RETURNING id").ToSql()
			if err != nil {
				return nil, nil, fmt.Errorf("Orders %s: %v", ErrToSql, err)
			}
			rows, err := tx.Query(ctx, sql, args...)
			if err != nil {
				return nil, nil, fmt.Errorf("Orders %s: %v", ErrQuery, err)
			}
			curInd := i - createdInLoop + 1
			for rows.Next() {
				err = rows.Scan(&ordersIDs[curInd])
				if err != nil {
					return nil, nil, fmt.Errorf("Orders %s: %v", ErrScan, err)
				}
				curInd++
			}
			rows.Close()
			if err = rows.Err(); err != nil {
				return nil, nil, fmt.Errorf("Orders %s: %v", ErrCloseRows, err)
			}
			insertBuilder = psql.Insert("orders").Columns("user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at")
			createdInLoop = 0
		}
	}

	seedOrders = make(SeedOrders, count)
	for i, oid := range ordersIDs {
		seedOrders[oid] = SeedOrder{
			SellerID: sellerPerIndex[i],
			Status:   statusPerIndex[i],
		}
	}

	return ordersIDs, seedOrders, nil
}

func distributedOrderDates(count int, from, to time.Time) ([]time.Time, error) {
	if count == 0 {
		return nil, nil
	}
	from = dateOnly(from)
	if to.Before(from) {
		return nil, fmt.Errorf("%s is after %s", from.Format(time.DateOnly), to.Format(time.DateOnly))
	}

	days := make([]time.Time, 0, int(dateOnly(to).Sub(from).Hours()/24)+1)
	weights := make([]float64, 0, cap(days))
	for day := from; !day.After(dateOnly(to)); day = day.AddDate(0, 0, 1) {
		days = append(days, day)
		weights = append(weights, orderDayWeight(day))
	}

	result := make([]time.Time, 0, count)
	for range count {
		result = append(result, randomTimeInDay(weightedDay(days, weights), to))
	}
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result, nil
}

func orderDayWeight(day time.Time) float64 {
	weight := 1.0
	if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
		weight += 0.2
	}
	for _, h := range russianOrderHolidayBoosts {
		holiday := time.Date(day.Year(), h.month, h.day, 0, 0, 0, 0, day.Location())
		distance := absDayDistance(day, holiday)
		if distance <= h.radiusDays {
			weight += h.boost * (1 - float64(distance)/float64(h.radiusDays+1))
		}
	}
	return weight
}

func weightedDay(days []time.Time, weights []float64) time.Time {
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	r := rand.Float64() * total
	for i, weight := range weights {
		r -= weight
		if r < 0 {
			return days[i]
		}
	}
	return days[len(days)-1]
}

func randomTimeInDay(day, max time.Time) time.Time {
	start := dateOnly(day)
	end := start.AddDate(0, 0, 1)
	if dateOnly(max).Equal(start) {
		end = max
	}
	span := end.Sub(start)
	if span <= 0 {
		return start
	}
	return start.Add(time.Duration(rand.Int63n(int64(span))))
}

func randomSeedOrderStatus(createdAt, now time.Time) string {
	age := now.Sub(createdAt)
	switch {
	case age > 30*24*time.Hour:
		return weightedChoice(
			[]string{"paid", "shipped", "delivered", "cancelled"},
			[]int{5, 15, 70, 10},
		)
	case age > 24*time.Hour:
		return weightedChoice(
			[]string{"paid", "shipped", "delivered", "cancelled"},
			[]int{15, 30, 45, 10},
		)
	default:
		return weightedChoice(
			[]string{"draft", "pending", "paid", "shipped", "delivered", "cancelled"},
			[]int{10, 20, 25, 20, 15, 10},
		)
	}
}

func randomSeedOrderBuyer(buyerIDs []int64, status string, usedDraftBuyers map[int64]struct{}) (int64, bool) {
	if status != "draft" {
		return buyerIDs[rand.Intn(len(buyerIDs))], true
	}
	if len(usedDraftBuyers) >= len(buyerIDs) {
		return 0, false
	}
	start := rand.Intn(len(buyerIDs))
	for offset := range len(buyerIDs) {
		buyerID := buyerIDs[(start+offset)%len(buyerIDs)]
		if _, ok := usedDraftBuyers[buyerID]; !ok {
			usedDraftBuyers[buyerID] = struct{}{}
			return buyerID, true
		}
	}
	return 0, false
}

func seedOrderUpdatedAt(createdAt time.Time, status string, max time.Time) time.Time {
	maxLag := 12 * time.Hour
	switch status {
	case "pending":
		maxLag = 24 * time.Hour
	case "paid":
		maxLag = 48 * time.Hour
	case "shipped":
		maxLag = 7 * 24 * time.Hour
	case "delivered", "cancelled":
		maxLag = 14 * 24 * time.Hour
	}
	updatedAt := createdAt.Add(time.Duration(rand.Int63n(int64(maxLag))))
	if updatedAt.After(max) {
		return max
	}
	return updatedAt
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func absDayDistance(a, b time.Time) int {
	if a.Before(b) {
		a, b = b, a
	}
	return int(a.Sub(b).Hours() / 24)
}
