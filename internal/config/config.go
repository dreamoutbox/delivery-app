package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort            string
	AppEnv             string
	SessionSecret      string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	MinIOEndpoint      string
	MinIOAccessKey     string
	MinIOSecretKey     string
	MinIOUseSSL        bool
	MinIOPublicBucket  string
	MinIOPrivateBucket string
	MinIOPublicURL     string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load() // Load .env if present

	cfg := &Config{
		AppPort:            getEnv("APP_PORT", "8080"),
		AppEnv:             getEnv("APP_ENV", "development"),
		SessionSecret:      getEnv("SESSION_SECRET", "super-secret-key-32-bytes-long-12345"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "example"),
		DBName:             getEnv("DB_NAME", "delivery_app"),
		MinIOEndpoint:      getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:     getEnv("MINIO_ACCESS_KEY", "miniominio"),
		MinIOSecretKey:     getEnv("MINIO_SECRET_KEY", "miniominio"),
		MinIOUseSSL:        getEnv("MINIO_USE_SSL", "false") == "true",
		MinIOPublicBucket:  getEnv("MINIO_PUBLIC_BUCKET", "delivery-public"),
		MinIOPrivateBucket: getEnv("MINIO_PRIVATE_BUCKET", "delivery-private"),
		MinIOPublicURL:     getEnv("MINIO_PUBLIC_URL", "http://localhost:9000/delivery-public"),
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Bangkok",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
