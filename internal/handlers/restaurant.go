package handlers

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/render"
	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/dreamoutbox/delivery-app/internal/storage"
	"github.com/gin-gonic/gin"
)

type RestaurantHandler struct {
	storageSvc *storage.StorageService
}

func NewRestaurantHandler(storageSvc *storage.StorageService) *RestaurantHandler {
	return &RestaurantHandler{storageSvc: storageSvc}
}

func (h *RestaurantHandler) Dashboard(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var restaurant models.Restaurant
	err := database.DB.Preload("MenuItems").Where("owner_id = ?", sessUser.ID).First(&restaurant).Error

	var hasRestaurant = true
	if err != nil {
		hasRestaurant = false
	}

	var orders []models.Order
	if hasRestaurant {
		database.DB.Preload("Customer").Preload("OrderItems").
			Where("restaurant_id = ?", restaurant.ID).
			Order("created_at DESC").
			Find(&orders)
	}

	render.HTML(c, http.StatusOK, "restaurant/dashboard.html", gin.H{
		"HasRestaurant": hasRestaurant,
		"Restaurant":    restaurant,
		"Orders":        orders,
		"User":          sessUser,
	})
}

func (h *RestaurantHandler) SaveRestaurant(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	description := strings.TrimSpace(c.PostForm("description"))
	address := strings.TrimSpace(c.PostForm("address"))
	latStr := strings.TrimSpace(c.PostForm("lat"))
	lngStr := strings.TrimSpace(c.PostForm("lng"))
	baseFeeStr := strings.TrimSpace(c.PostForm("delivery_base_fee_thb"))
	perKmStr := strings.TrimSpace(c.PostForm("delivery_per_km_thb"))
	isOpen := c.PostForm("is_open") == "on" || c.PostForm("is_open") == "true"

	lat, _ := strconv.ParseFloat(latStr, 64)
	lng, _ := strconv.ParseFloat(lngStr, 64)

	baseFeeTHB, _ := strconv.ParseFloat(baseFeeStr, 64)
	perKmTHB, _ := strconv.ParseFloat(perKmStr, 64)

	baseFeeSatang := int(math.Round(baseFeeTHB * 100))
	perKmSatang := int(math.Round(perKmTHB * 100))

	var restaurant models.Restaurant
	err := database.DB.Where("owner_id = ?", sessUser.ID).First(&restaurant).Error

	imageURL := restaurant.ImagePath
	file, header, errFile := c.Request.FormFile("image")
	if errFile == nil && file != nil && h.storageSvc != nil {
		defer file.Close()
		uploadedURL, uploadErr := h.storageSvc.UploadPublicFile(file, header.Filename, header.Header.Get("Content-Type"))
		if uploadErr == nil {
			imageURL = uploadedURL
		}
	}

	if err != nil {
		restaurant = models.Restaurant{
			OwnerID:                sessUser.ID,
			Name:                   name,
			Description:            description,
			Address:                address,
			Lat:                    lat,
			Lng:                    lng,
			ImagePath:              imageURL,
			DeliveryBaseFeeSatang: baseFeeSatang,
			DeliveryPerKmSatang:   perKmSatang,
			IsOpen:                 isOpen,
		}
		database.DB.Create(&restaurant)
	} else {
		updates := map[string]interface{}{
			"name":                       name,
			"description":                description,
			"address":                    address,
			"lat":                        lat,
			"lng":                        lng,
			"delivery_base_fee_satang": baseFeeSatang,
			"delivery_per_km_satang":   perKmSatang,
			"is_open":                    isOpen,
		}
		if imageURL != "" {
			updates["image_path"] = imageURL
		}
		database.DB.Model(&restaurant).Updates(updates)
	}

	c.Redirect(http.StatusFound, "/restaurant")
}

func (h *RestaurantHandler) SaveMenuItem(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var restaurant models.Restaurant
	if err := database.DB.Where("owner_id = ?", sessUser.ID).First(&restaurant).Error; err != nil {
		render.HTML(c, http.StatusBadRequest, "error.html", gin.H{"Title": "No Restaurant", "Message": "Create a restaurant first."})
		return
	}

	itemIDStr := c.PostForm("item_id")
	name := strings.TrimSpace(c.PostForm("name"))
	description := strings.TrimSpace(c.PostForm("description"))
	priceTHBStr := strings.TrimSpace(c.PostForm("price_thb"))
	isAvailable := c.PostForm("is_available") == "on" || c.PostForm("is_available") == "true"

	priceTHB, _ := strconv.ParseFloat(priceTHBStr, 64)
	priceSatang := int(math.Round(priceTHB * 100))

	var imageURL string
	file, header, errFile := c.Request.FormFile("image")
	if errFile == nil && file != nil && h.storageSvc != nil {
		defer file.Close()
		if uploadedURL, uploadErr := h.storageSvc.UploadPublicFile(file, header.Filename, header.Header.Get("Content-Type")); uploadErr == nil {
			imageURL = uploadedURL
		}
	}

	if itemIDStr != "" {
		itemID, _ := strconv.Atoi(itemIDStr)
		var item models.MenuItem
		if err := database.DB.Where("id = ? AND restaurant_id = ?", itemID, restaurant.ID).First(&item).Error; err == nil {
			updates := map[string]interface{}{
				"name":         name,
				"description":  description,
				"price_satang": priceSatang,
				"is_available": isAvailable,
			}
			if imageURL != "" {
				updates["image_path"] = imageURL
			}
			database.DB.Model(&item).Updates(updates)
		}
	} else {
		item := models.MenuItem{
			RestaurantID: restaurant.ID,
			Name:         name,
			Description:  description,
			PriceSatang:  priceSatang,
			ImagePath:    imageURL,
			IsAvailable:  isAvailable,
		}
		database.DB.Create(&item)
	}

	c.Redirect(http.StatusFound, "/restaurant")
}

func (h *RestaurantHandler) DeleteMenuItem(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	itemIDStr := c.Param("id")
	itemID, _ := strconv.Atoi(itemIDStr)

	var restaurant models.Restaurant
	if err := database.DB.Where("owner_id = ?", sessUser.ID).First(&restaurant).Error; err == nil {
		database.DB.Where("id = ? AND restaurant_id = ?", itemID, restaurant.ID).Delete(&models.MenuItem{})
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/restaurant")
}

func (h *RestaurantHandler) UpdateOrderStatus(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	orderIDStr := c.Param("id")
	orderID, _ := strconv.Atoi(orderIDStr)
	newStatus := c.PostForm("status")

	var restaurant models.Restaurant
	if err := database.DB.Where("owner_id = ?", sessUser.ID).First(&restaurant).Error; err != nil {
		c.String(http.StatusForbidden, "Forbidden")
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND restaurant_id = ?", orderID, restaurant.ID).First(&order).Error; err == nil {
		validStatuses := map[string]bool{
			models.StatusConfirmed:      true,
			models.StatusReadyForPickup: true,
			models.StatusCancelled:      true,
		}
		if validStatuses[newStatus] {
			database.DB.Model(&order).Update("status", newStatus)
		}
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/restaurant")
}
