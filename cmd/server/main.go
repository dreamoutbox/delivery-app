package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/dreamoutbox/delivery-app/internal/config"
	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Initializing FastBite Delivery App...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	_ = db

	storageSvc, err := storage.InitStorage(cfg)
	if err != nil {
		log.Printf("MinIO Storage notice: %v", err)
	} else {
		_ = storageSvc
		log.Println("MinIO Storage initialized.")
	}

	r := gin.Default()

	// Load HTML templates with custom functions
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"formatTHB": func(satang int) string {
			baht := float64(satang) / 100.0
			return fmt.Sprintf("฿%.2f", baht)
		},
	}).ParseGlob("templates/**/*.html"))
	r.SetHTMLTemplate(tmpl)

	// Static files
	r.Static("/static", "./static")

	// Routes
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "pong"})
	})

	r.GET("/", func(c *gin.Context) {
		var restaurants []models.Restaurant
		database.DB.Where("is_open = ?", true).Find(&restaurants)

		c.HTML(http.StatusOK, "home.html", gin.H{
			"Restaurants": restaurants,
			"User":        nil, // Will be populated when Auth is ready
		})
	})

	log.Printf("Server starting on port %s...", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
