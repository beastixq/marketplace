package generators

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Masterminds/squirrel"
	"github.com/go-faker/faker/v4"
"github.com/jackc/pgx/v5"
)

type categoryGroup struct {
	parent   string
	children []string
}

var realCategories = []categoryGroup{
	{"Electronics", []string{"Smartphones", "Laptops", "Tablets", "Cameras", "Headphones", "Smartwatches", "Televisions", "Speakers", "Gaming Consoles", "Computer Accessories"}},
	{"Clothing", []string{"Men's Clothing", "Women's Clothing", "Children's Clothing", "Shoes", "Bags & Luggage", "Sportswear", "Underwear", "Outerwear", "Fashion Accessories"}},
	{"Home & Garden", []string{"Furniture", "Kitchen Supplies", "Bathroom Accessories", "Home Decor", "Garden Tools", "Lighting", "Home Textiles", "Storage & Organization"}},
	{"Sports & Outdoors", []string{"Fitness Equipment", "Cycling", "Camping & Hiking", "Swimming", "Winter Sports", "Running", "Team Sports", "Yoga & Pilates"}},
	{"Books & Media", []string{"Fiction", "Non-Fiction", "Science & Technology", "Children's Books", "Educational", "Comics & Manga", "Audiobooks", "E-Books"}},
	{"Toys & Games", []string{"Board Games", "Puzzles", "Dolls & Figures", "Educational Toys", "Outdoor Toys", "Video Games", "Building Sets", "RC Toys"}},
	{"Health & Beauty", []string{"Skincare", "Haircare", "Makeup", "Perfume", "Supplements", "Medical Devices", "Personal Hygiene", "Men's Grooming"}},
	{"Automotive", []string{"Car Parts", "Car Electronics", "Car Care", "Motorcycle Parts", "Tires & Wheels", "Car Accessories", "Tools & Equipment"}},
	{"Food & Beverages", []string{"Snacks", "Drinks", "Organic Food", "Tea & Coffee", "Sweets & Chocolate", "Canned Food", "Spices & Seasonings", "Dairy Products"}},
	{"Pet Supplies", []string{"Dog Supplies", "Cat Supplies", "Fish & Aquarium", "Bird Supplies", "Small Animal Supplies", "Pet Food", "Pet Toys", "Pet Health"}},
	{"Office & Stationery", []string{"Writing Supplies", "Printers & Scanners", "Paper Products", "Desk Organization", "Office Furniture", "Ink & Toner", "Filing & Storage"}},
	{"Musical Instruments", []string{"Guitars", "Keyboards & Pianos", "Drums & Percussion", "Wind Instruments", "String Instruments", "DJ Equipment", "Music Accessories"}},
	{"Jewelry & Watches", []string{"Rings", "Necklaces", "Bracelets", "Watches", "Earrings", "Brooches & Pins"}},
	{"Baby & Kids", []string{"Strollers", "Car Seats", "Feeding Supplies", "Baby Clothing", "Nursery Furniture", "Baby Safety", "Diapers & Wipes"}},
	{"Tools & Hardware", []string{"Power Tools", "Hand Tools", "Measuring Tools", "Plumbing Supplies", "Electrical Supplies", "Safety Equipment", "Fasteners & Adhesives"}},
}

func CreateRealCategories(tx pgx.Tx, ctx context.Context, targetCount int) (categoriesIDs []int64, err error) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Phase 1: insert parent categories
	parentBuilder := psql.Insert("categories").Columns("name", "description")
	for _, group := range realCategories {
		parentBuilder = parentBuilder.Values(group.parent, "Category: "+group.parent)
	}

	sql, args, err := parentBuilder.Suffix("RETURNING id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("Categories parents %s: %v", ErrToSql, err)
	}

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("Categories parents %s: %v", ErrQuery, err)
	}

	parentIDs := make([]int64, 0, len(realCategories))
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("Categories parents %s: %v", ErrScan, err)
		}
		parentIDs = append(parentIDs, id)
		categoriesIDs = append(categoriesIDs, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Categories parents %s: %v", ErrCloseRows, err)
	}

	// Phase 2: insert real child categories
	childBuilder := psql.Insert("categories").Columns("parent_id", "name", "description")
	for i, group := range realCategories {
		for _, child := range group.children {
			childBuilder = childBuilder.Values(parentIDs[i], child, "Subcategory of "+group.parent)
		}
	}

	sql, args, err = childBuilder.Suffix("RETURNING id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("Categories children %s: %v", ErrToSql, err)
	}

	rows, err = tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("Categories children %s: %v", ErrQuery, err)
	}

	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("Categories children %s: %v", ErrScan, err)
		}
		categoriesIDs = append(categoriesIDs, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Categories children %s: %v", ErrCloseRows, err)
	}

	// Phase 3: generate faker subcategories to reach 1000+
	remaining := targetCount - len(categoriesIDs)
	if remaining <= 0 {
		return categoriesIDs, nil
	}

	createdInLoop := 0
	seen := make(map[string]bool)
	for _, group := range realCategories {
		seen[group.parent] = true
		for _, child := range group.children {
			seen[child] = true
		}
	}
	genBuilder := psql.Insert("categories").Columns("parent_id", "name", "description")
	for i := range remaining {
		parentIdx := rand.Intn(len(parentIDs))
		name := faker.Word() + " " + faker.Word()
		for seen[name] {
			name = faker.Word() + " " + faker.Word()
		}
		seen[name] = true
		genBuilder = genBuilder.Values(parentIDs[parentIdx], name, faker.Sentence())
		createdInLoop++

		if i%10 == 9 || i == remaining-1 {
			sql, args, err = genBuilder.Suffix("RETURNING id").ToSql()
			if err != nil {
				return nil, fmt.Errorf("Categories generated %s: %v", ErrToSql, err)
			}
			rows, err = tx.Query(ctx, sql, args...)
			if err != nil {
				return nil, fmt.Errorf("Categories generated %s: %v", ErrQuery, err)
			}
			for rows.Next() {
				var id int64
				if err = rows.Scan(&id); err != nil {
					return nil, fmt.Errorf("Categories generated %s: %v", ErrScan, err)
				}
				categoriesIDs = append(categoriesIDs, id)
			}
			rows.Close()
			if err = rows.Err(); err != nil {
				return nil, fmt.Errorf("Categories generated %s: %v", ErrCloseRows, err)
			}
			genBuilder = psql.Insert("categories").Columns("parent_id", "name", "description")
			createdInLoop = 0
		}
	}

	return categoriesIDs, nil
}
