package database

import (
	"fmt"
	"log"

	"github.com/dreamoutbox/delivery-app/internal/config"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ensureDatabaseExists(cfg *config.Config) error {
	// Connect to default 'postgres' database to check/create target DB
	defaultDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable TimeZone=Asia/Bangkok",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword)

	sysDB, err := gorm.Open(postgres.Open(defaultDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to default postgres DB: %w", err)
	}

	sqlDB, err := sysDB.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	var count int
	checkSQL := "SELECT count(*) FROM pg_database WHERE datname = ?"
	if err := sysDB.Raw(checkSQL, cfg.DBName).Scan(&count).Error; err != nil {
		return fmt.Errorf("failed to query pg_database: %w", err)
	}

	if count == 0 {
		log.Printf("Database '%s' does not exist. Creating...", cfg.DBName)
		createSQL := fmt.Sprintf("CREATE DATABASE \"%s\"", cfg.DBName)
		if err := sysDB.Exec(createSQL).Error; err != nil {
			return fmt.Errorf("failed to create database '%s': %w", cfg.DBName, err)
		}
		log.Printf("Database '%s' created successfully.", cfg.DBName)
	}

	return nil
}

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	if err := ensureDatabaseExists(cfg); err != nil {
		log.Printf("Notice during ensureDatabaseExists: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database '%s': %w", cfg.DBName, err)
	}

	log.Println("Database connection established. Ensuring PostgreSQL ENUM types & AutoMigrating models...")

	// Ensure PostgreSQL ENUM type 'order_status' exists
	db.Exec(`DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status') THEN
			CREATE TYPE order_status AS ENUM ('pending', 'confirmed', 'ready_for_pickup', 'in_delivery', 'delivered', 'cancelled');
		END IF;
	END $$;`)

	// If orders table exists with non-enum status column, alter column type to order_status
	db.Exec(`DO $$ BEGIN
		IF EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'orders' AND column_name = 'status' AND udt_name != 'order_status'
		) THEN
			ALTER TABLE orders ALTER COLUMN status DROP DEFAULT;
			ALTER TABLE orders ALTER COLUMN status TYPE order_status USING status::order_status;
			ALTER TABLE orders ALTER COLUMN status SET DEFAULT 'pending'::order_status;
		END IF;
	END $$;`)

	err = db.AutoMigrate(
		&models.User{},
		&models.Restaurant{},
		&models.MenuItem{},
		&models.Order{},
		&models.OrderItem{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	DB = db
	return db, nil
}
