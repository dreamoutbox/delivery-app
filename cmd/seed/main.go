package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/dreamoutbox/delivery-app/internal/config"
	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

func uploadLocalImage(storageSvc *storage.StorageService, localPath string) string {
	if storageSvc == nil {
		return ""
	}
	f, err := os.Open(localPath)
	if err != nil {
		log.Printf("Warning: Could not open mock image %s: %v", localPath, err)
		return ""
	}
	defer f.Close()

	filename := filepath.Base(localPath)
	url, err := storageSvc.UploadPublicFile(f, filename, "image/jpeg")
	if err != nil {
		log.Printf("Warning: Failed to upload image %s to MinIO/S3: %v", localPath, err)
		return ""
	}
	log.Printf("Uploaded image %s -> %s", filename, url)
	return url
}

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

	storageSvc, err := storage.InitStorage(cfg)
	if err != nil {
		log.Printf("Notice: Could not initialize storage service: %v", err)
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

	// 2. Seed Restaurants & Upload Banner Images
	res1Image := uploadLocalImage(storageSvc, "mock/images/res1-somtum.jpg")
	r1 := models.Restaurant{
		OwnerID:               owner1.ID,
		Name:                  "Somtum & Grill Express",
		Description:           "Authentic Isan street food & grilled chicken",
		Address:               "Victory Monument, Bangkok",
		Lat:                   13.7649,
		Lng:                   100.5383,
		ImagePath:             res1Image,
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
		if res1Image != "" {
			db.Model(&r1).Update("image_path", res1Image)
			log.Printf("Updated image for restaurant: %s", r1.Name)
		}
	}

	res2Image := uploadLocalImage(storageSvc, "mock/images/res2-burger.jpg")
	r2 := models.Restaurant{
		OwnerID:               owner2.ID,
		Name:                  "Bangkok Burger & Wings",
		Description:           "Smash burgers, crispy wings, and craft shakes",
		Address:               "Siam Paragon Area, Bangkok",
		Lat:                   13.7460,
		Lng:                   100.5347,
		ImagePath:             res2Image,
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
		if res2Image != "" {
			db.Model(&r2).Update("image_path", res2Image)
			log.Printf("Updated image for restaurant: %s", r2.Name)
		}
	}

	// 3. Seed Menu Items & Upload Food Images
	menuItemsR1 := []struct {
		Item      models.MenuItem
		LocalPath string
	}{
		{
			Item: models.MenuItem{
				RestaurantID: r1.ID,
				Name:         "Somtum Thai (Papaya Salad)",
				Description:  "Spicy papaya salad with peanuts and dried shrimp",
				PriceSatang:  6000, // 60 THB
				IsAvailable:  true,
			},
			LocalPath: "mock/images/somtam.jpg",
		},
		{
			Item: models.MenuItem{
				RestaurantID: r1.ID,
				Name:         "Grilled Chicken Half",
				Description:  "Juicy grilled chicken marinated in lemongrass and garlic",
				PriceSatang:  12000, // 120 THB
				IsAvailable:  true,
			},
			LocalPath: "mock/images/grill-chicken.jpg",
		},
		{
			Item: models.MenuItem{
				RestaurantID: r1.ID,
				Name:         "Sticky Rice",
				Description:  "Warm bamboo-steamed sticky rice",
				PriceSatang:  1500, // 15 THB
				IsAvailable:  true,
			},
			LocalPath: "mock/images/stickyrice.jpg",
		},
	}

	for _, entry := range menuItemsR1 {
		item := entry.Item
		imgURL := uploadLocalImage(storageSvc, entry.LocalPath)
		if imgURL != "" {
			item.ImagePath = imgURL
		}

		var existing models.MenuItem
		if err := db.Where("restaurant_id = ? AND name = ?", item.RestaurantID, item.Name).First(&existing).Error; err != nil {
			db.Create(&item)
			log.Printf("Seeded menu item: %s (%s)", item.Name, r1.Name)
		} else if imgURL != "" {
			db.Model(&existing).Update("image_path", imgURL)
			log.Printf("Updated image for menu item: %s", item.Name)
		}
	}

	menuItemsR2 := []struct {
		Item      models.MenuItem
		LocalPath string
	}{
		{
			Item: models.MenuItem{
				RestaurantID: r2.ID,
				Name:         "Double Cheeseburger",
				Description:  "Two beef patties, cheddar cheese, special sauce",
				PriceSatang:  18900, // 189 THB
				IsAvailable:  true,
			},
			LocalPath: "mock/images/cheeseburger.jpg",
		},
		{
			Item: models.MenuItem{
				RestaurantID: r2.ID,
				Name:         "Crispy Buffalo Wings (6pcs)",
				Description:  "Tossed in spicy buffalo glaze with ranch dip",
				PriceSatang:  14900, // 149 THB
				IsAvailable:  true,
			},
			LocalPath: "mock/images/buffalowings.jpg",
		},
		{
			Item: models.MenuItem{
				RestaurantID: r2.ID,
				Name:         "French Fries",
				Description:  "Golden sea salted fries",
				PriceSatang:  5900, // 59 THB
				IsAvailable:  true,
			},
			LocalPath: "mock/images/frenchfries.jpg",
		},
	}

	for _, entry := range menuItemsR2 {
		item := entry.Item
		imgURL := uploadLocalImage(storageSvc, entry.LocalPath)
		if imgURL != "" {
			item.ImagePath = imgURL
		}

		var existing models.MenuItem
		if err := db.Where("restaurant_id = ? AND name = ?", item.RestaurantID, item.Name).First(&existing).Error; err != nil {
			db.Create(&item)
			log.Printf("Seeded menu item: %s (%s)", item.Name, r2.Name)
		} else if imgURL != "" {
			db.Model(&existing).Update("image_path", imgURL)
			log.Printf("Updated image for menu item: %s", item.Name)
		}
	}

	log.Println("Seeder completed successfully!")
}

func floatPtr(v float64) *float64 {
	return &v
}
