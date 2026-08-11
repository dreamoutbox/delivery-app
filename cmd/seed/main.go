package main

import (
	"log"

	"github.com/dreamoutbox/delivery-app/internal/config"
	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	log.Println("Starting database seeder...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect DB: %v", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	passStr := string(hashedPassword)

	// 1. Seed Users
	users := []models.User{
		{
			Name:         "System Admin",
			Email:        "admin@example.com",
			PasswordHash: passStr,
			Role:         models.RoleAdmin,
			Phone:        "0811111111",
		},
		{
			Name:         "Staff 1",
			Email:        "staff1@example.com",
			PasswordHash: passStr,
			Role:         models.RoleStaff,
			Phone:        "0822222222",
		},
		{
			Name:         "Chef Somchai (Owner 1)",
			Email:        "owner1@example.com",
			PasswordHash: passStr,
			Role:         models.RoleRestaurant,
			Phone:        "0833333333",
		},
		{
			Name:         "John Burger (Owner 2)",
			Email:        "owner2@example.com",
			PasswordHash: passStr,
			Role:         models.RoleRestaurant,
			Phone:        "0844444444",
		},
		{
			Name:         "Rider Sam",
			Email:        "rider1@example.com",
			PasswordHash: passStr,
			Role:         models.RoleRider,
			Phone:        "0855555555",
		},
		{
			Name:         "Alice Customer",
			Email:        "customer1@example.com",
			PasswordHash: passStr,
			Role:         models.RoleCustomer,
			Phone:        "0866666666",
			Address:      "Home Customer 1",
			Lat:          floatPtr(13.7563),
			Lng:          floatPtr(100.5018),
		},
	}

	for _, u := range users {
		var existing models.User
		if err := db.Where("email = ?", u.Email).First(&existing).Error; err != nil {
			db.Create(&u)
			log.Printf("Seeded user: %s (%s)", u.Email, u.Role)
		}
	}

	// Fetch owners
	var owner1, owner2 models.User
	db.Where("email = ?", "owner1@example.com").First(&owner1)
	db.Where("email = ?", "owner2@example.com").First(&owner2)

	// 2. Seed Restaurants
	r1 := models.Restaurant{
		OwnerID:               owner1.ID,
		Name:                  "Somtum & Grill Express",
		Description:           "Authentic Isan street food & grilled chicken",
		Address:               "Victory Monument, Bangkok",
		Lat:                   13.7649,
		Lng:                   100.5383,
		DeliveryBaseFeeSatang: 1500, // 15 THB
		DeliveryPerKmSatang:   500,  // 5 THB/km
		IsOpen:                true,
	}
	var existingR1 models.Restaurant
	if err := db.Where("name = ?", r1.Name).First(&existingR1).Error; err != nil {
		db.Create(&r1)
		log.Printf("Seeded restaurant: %s", r1.Name)
	} else {
		r1 = existingR1
	}

	r2 := models.Restaurant{
		OwnerID:               owner2.ID,
		Name:                  "Bangkok Burger & Wings",
		Description:           "Smash burgers, crispy wings, and craft shakes",
		Address:               "Siam Paragon Area, Bangkok",
		Lat:                   13.7460,
		Lng:                   100.5347,
		DeliveryBaseFeeSatang: 2000, // 20 THB
		DeliveryPerKmSatang:   600,  // 6 THB/km
		IsOpen:                true,
	}
	var existingR2 models.Restaurant
	if err := db.Where("name = ?", r2.Name).First(&existingR2).Error; err != nil {
		db.Create(&r2)
		log.Printf("Seeded restaurant: %s", r2.Name)
	} else {
		r2 = existingR2
	}

	// 3. Seed Menu Items
	menuItemsR1 := []models.MenuItem{
		{
			RestaurantID: r1.ID,
			Name:         "Somtum Thai (Papaya Salad)",
			Description:  "Spicy papaya salad with peanuts and dried shrimp",
			PriceSatang:  6000, // 60 THB
			IsAvailable:  true,
		},
		{
			RestaurantID: r1.ID,
			Name:         "Grilled Chicken Half",
			Description:  "Juicy grilled chicken marinated in lemongrass and garlic",
			PriceSatang:  12000, // 120 THB
			IsAvailable:  true,
		},
		{
			RestaurantID: r1.ID,
			Name:         "Sticky Rice",
			Description:  "Warm bamboo-steamed sticky rice",
			PriceSatang:  1500, // 15 THB
			IsAvailable:  true,
		},
	}

	for _, item := range menuItemsR1 {
		var existing models.MenuItem
		if err := db.Where("restaurant_id = ? AND name = ?", item.RestaurantID, item.Name).First(&existing).Error; err != nil {
			db.Create(&item)
			log.Printf("Seeded menu item: %s (%s)", item.Name, r1.Name)
		}
	}

	menuItemsR2 := []models.MenuItem{
		{
			RestaurantID: r2.ID,
			Name:         "Double Cheeseburger",
			Description:  "Two beef patties, cheddar cheese, special sauce",
			PriceSatang:  18900, // 189 THB
			IsAvailable:  true,
		},
		{
			RestaurantID: r2.ID,
			Name:         "Crispy Buffalo Wings (6pcs)",
			Description:  "Tossed in spicy buffalo glaze with ranch dip",
			PriceSatang:  14900, // 149 THB
			IsAvailable:  true,
		},
		{
			RestaurantID: r2.ID,
			Name:         "French Fries",
			Description:  "Golden sea salted fries",
			PriceSatang:  5900, // 59 THB
			IsAvailable:  true,
		},
	}

	for _, item := range menuItemsR2 {
		var existing models.MenuItem
		if err := db.Where("restaurant_id = ? AND name = ?", item.RestaurantID, item.Name).First(&existing).Error; err != nil {
			db.Create(&item)
			log.Printf("Seeded menu item: %s (%s)", item.Name, r2.Name)
		}
	}

	log.Println("Seeder completed successfully!")
}

func floatPtr(v float64) *float64 {
	return &v
}
