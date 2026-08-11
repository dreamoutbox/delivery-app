package handlers

import (
	"fmt"
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
	var err error

	idStr := c.Query("id")
	if (sessUser.Role == models.RoleAdmin || sessUser.Role == models.RoleStaff) && idStr != "" {
		restID, _ := strconv.Atoi(idStr)
		err = database.DB.Preload("MenuItems").First(&restaurant, restID).Error
	} else {
		err = database.DB.Preload("MenuItems").Where("owner_id = ?", sessUser.ID).First(&restaurant).Error
	}

	var hasRestaurant = true
	if err != nil {
		hasRestaurant = false
	}

	render.HTML(c, http.StatusOK, "restaurant/dashboard.html", gin.H{
		"HasRestaurant":    hasRestaurant,
		"Restaurant":       restaurant,
		"User":             sessUser,
		"IsRestaurantView": true,
	})
}

func (h *RestaurantHandler) Orders(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var restaurant models.Restaurant
	var err error

	idStr := c.Query("id")
	if (sessUser.Role == models.RoleAdmin || sessUser.Role == models.RoleStaff) && idStr != "" {
		restID, _ := strconv.Atoi(idStr)
		err = database.DB.First(&restaurant, restID).Error
	} else {
		err = database.DB.Where("owner_id = ?", sessUser.ID).First(&restaurant).Error
	}

	if err != nil {
		render.HTML(c, http.StatusBadRequest, "error.html", gin.H{"Title": "No Restaurant Found", "Message": "Restaurant account could not be found."})
		return
	}

	statusFilter := strings.TrimSpace(c.Query("status"))
	if statusFilter == "" {
		statusFilter = "all"
	}

	q := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize := 10

	dbQuery := database.DB.Model(&models.Order{}).Where("restaurant_id = ?", restaurant.ID)

	if statusFilter == "active" {
		dbQuery = dbQuery.Where("status IN (?, ?, ?, ?)", models.StatusPending, models.StatusConfirmed, models.StatusReadyForPickup, models.StatusInDelivery)
	} else if statusFilter == "completed" {
		dbQuery = dbQuery.Where("status = ?", models.StatusDelivered)
	} else if statusFilter != "all" && statusFilter != "" {
		dbQuery = dbQuery.Where("status = ?", statusFilter)
	}

	if q != "" {
		like := "%" + q + "%"
		dbQuery = dbQuery.Where("delivery_address ILIKE ? OR CAST(id AS TEXT) LIKE ?", like, like)
	}

	var totalItems int64
	dbQuery.Count(&totalItems)

	paging := calcPagination(totalItems, page, pageSize)

	var orders []models.Order
	dbQuery.Preload("Customer").Preload("OrderItems").Preload("Rider").
		Order("created_at DESC").
		Offset((paging.CurrentPage - 1) * pageSize).
		Limit(pageSize).
		Find(&orders)

	render.HTML(c, http.StatusOK, "restaurant/orders.html", gin.H{
		"Restaurant":       restaurant,
		"Orders":           orders,
		"StatusFilter":     statusFilter,
		"Q":                q,
		"Paging":           paging,
		"User":             sessUser,
		"IsRestaurantView": true,
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

	targetIDStr := c.PostForm("restaurant_id")
	if targetIDStr == "" {
		targetIDStr = c.PostForm("id")
	}
	if targetIDStr == "" {
		targetIDStr = c.Query("id")
	}

	var restaurant models.Restaurant
	var err error

	if targetIDStr != "" {
		targetID, _ := strconv.Atoi(targetIDStr)
		err = database.DB.First(&restaurant, targetID).Error
	} else {
		err = database.DB.Where("owner_id = ?", sessUser.ID).First(&restaurant).Error
	}

	imageURL := restaurant.ImagePath
	file, header, errFile := c.Request.FormFile("image")
	if errFile == nil && file != nil && h.storageSvc != nil {
		defer file.Close()
		uploadedURL, uploadErr := h.storageSvc.UploadPublicFile(file, header.Filename, header.Header.Get("Content-Type"))
		if uploadErr == nil {
			imageURL = uploadedURL
		}
	}

	if err != nil && restaurant.ID == 0 {
		restaurant = models.Restaurant{
			OwnerID:               sessUser.ID,
			Name:                  name,
			Description:           description,
			Address:               address,
			Lat:                   lat,
			Lng:                   lng,
			ImagePath:             imageURL,
			DeliveryBaseFeeSatang: baseFeeSatang,
			DeliveryPerKmSatang:   perKmSatang,
			IsOpen:                isOpen,
		}
		database.DB.Create(&restaurant)
	} else {
		updates := map[string]interface{}{
			"name":                  name,
			"description":           description,
			"address":               address,
			"lat":                   lat,
			"lng":                   lng,
			"delivery_base_fee_satang": baseFeeSatang,
			"delivery_per_km_satang":   perKmSatang,
			"is_open":               isOpen,
		}
		if imageURL != "" {
			updates["image_path"] = imageURL
		}
		database.DB.Model(&restaurant).Updates(updates)
	}

	redirectPath := "/restaurant"
	if sessUser.Role == models.RoleAdmin || sessUser.Role == models.RoleStaff {
		redirectPath = fmt.Sprintf("/restaurant?id=%d", restaurant.ID)
	}

	c.Redirect(http.StatusFound, redirectPath)
}

func (h *RestaurantHandler) SaveMenuItem(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var restaurant models.Restaurant
	var err error

	targetIDStr := c.PostForm("restaurant_id")
	if (sessUser.Role == models.RoleAdmin || sessUser.Role == models.RoleStaff) && targetIDStr != "" {
		targetID, _ := strconv.Atoi(targetIDStr)
		err = database.DB.First(&restaurant, targetID).Error
	} else {
		err = database.DB.Where("owner_id = ?", sessUser.ID).First(&restaurant).Error
	}

	if err != nil {
		render.HTML(c, http.StatusBadRequest, "error.html", gin.H{"Title": "No Restaurant", "Message": "Restaurant not found."})
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

	redirectPath := "/restaurant"
	if sessUser.Role == models.RoleAdmin || sessUser.Role == models.RoleStaff {
		redirectPath = fmt.Sprintf("/restaurant?id=%d", restaurant.ID)
	}

	c.Redirect(http.StatusFound, redirectPath)
}

func (h *RestaurantHandler) DeleteMenuItem(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	itemIDStr := c.Param("id")
	itemID, _ := strconv.Atoi(itemIDStr)

	var item models.MenuItem
	if err := database.DB.First(&item, itemID).Error; err == nil {
		var restaurant models.Restaurant
		if err := database.DB.First(&restaurant, item.RestaurantID).Error; err == nil {
			if restaurant.OwnerID == sessUser.ID || sessUser.Role == models.RoleAdmin || sessUser.Role == models.RoleStaff {
				database.DB.Delete(&item)
			}
		}
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

	var order models.Order
	if err := database.DB.First(&order, orderID).Error; err == nil {
		var restaurant models.Restaurant
		if err := database.DB.First(&restaurant, order.RestaurantID).Error; err == nil {
			if restaurant.OwnerID == sessUser.ID || sessUser.Role == models.RoleAdmin || sessUser.Role == models.RoleStaff {
				validStatuses := map[string]bool{
					models.StatusConfirmed:      true,
					models.StatusReadyForPickup: true,
					models.StatusCancelled:      true,
				}
				if validStatuses[newStatus] {
					database.DB.Model(&order).Update("status", newStatus)
				}
			}
		}
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/restaurant/orders")
}
