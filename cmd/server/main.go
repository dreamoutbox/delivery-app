package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/dreamoutbox/delivery-app/internal/config"
	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/handlers"
	"github.com/dreamoutbox/delivery-app/internal/middleware"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/session"
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

	session.InitStore(cfg)

	storageSvc, err := storage.InitStorage(cfg)
	if err != nil {
		log.Printf("MinIO Storage notice: %v", err)
	} else {
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

	// Middleware
	r.Use(middleware.InjectUser())

	// Handlers
	authHandler := handlers.NewAuthHandler()
	userHandler := handlers.NewUserHandler()
	restHandler := handlers.NewRestaurantHandler(storageSvc)

	// Public Routes
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "pong"})
	})

	r.GET("/", func(c *gin.Context) {
		var restaurants []models.Restaurant
		database.DB.Where("is_open = ?", true).Find(&restaurants)

		user, _ := c.Get("user")

		c.HTML(http.StatusOK, "home.html", gin.H{
			"Restaurants": restaurants,
			"User":        user,
		})
	})

	r.GET("/login", authHandler.ShowLogin)
	r.POST("/login", authHandler.Login)
	r.GET("/register", authHandler.ShowRegister)
	r.POST("/register", authHandler.Register)
	r.POST("/logout", authHandler.Logout)

	// User Authenticated Routes
	authRoutes := r.Group("/")
	authRoutes.Use(middleware.RequireLogin())
	{
		authRoutes.GET("/profile", userHandler.ShowProfile)
		authRoutes.POST("/profile", userHandler.UpdateProfile)
	}

	// Restaurant Owner & Admin Routes
	restRoutes := r.Group("/restaurant")
	restRoutes.Use(middleware.RequireLogin(), middleware.RequireRole(models.RoleRestaurant, models.RoleAdmin))
	{
		restRoutes.GET("", restHandler.Dashboard)
		restRoutes.POST("/save", restHandler.SaveRestaurant)
		restRoutes.POST("/menu/save", restHandler.SaveMenuItem)
		restRoutes.POST("/menu/:id/delete", restHandler.DeleteMenuItem)
	}

	log.Printf("Server starting on port %s...", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
