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

type PaginationMeta struct {
	CurrentPage int
	TotalPages  int
	TotalItems  int64
	HasPrev     bool
	HasNext     bool
	PrevPage    int
	NextPage    int
	PageSize    int
}

func calcPagination(total int64, page, pageSize int) PaginationMeta {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	return PaginationMeta{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalItems:  total,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
		PageSize:    pageSize,
	}
}

type AdminHandler struct {
	storageSvc *storage.StorageService
}

func NewAdminHandler(storageSvc *storage.StorageService) *AdminHandler {
	return &AdminHandler{storageSvc: storageSvc}
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	sessUser := session.GetUser(c)

	var totalUsers, totalRestaurants, totalOrders int64
	var totalRevenueSatang int64

	database.DB.Model(&models.User{}).Count(&totalUsers)
	database.DB.Model(&models.Restaurant{}).Count(&totalRestaurants)
	database.DB.Model(&models.Order{}).Count(&totalOrders)
	database.DB.Model(&models.Order{}).Where("status = ?", models.StatusDelivered).Select("COALESCE(SUM(total_satang), 0)").Scan(&totalRevenueSatang)

	render.HTML(c, http.StatusOK, "admin/dashboard.html", gin.H{
		"TotalUsers":         totalUsers,
		"TotalRestaurants":   totalRestaurants,
		"TotalOrders":        totalOrders,
		"TotalRevenueSatang": totalRevenueSatang,
		"User":               sessUser,
	})
}

func (h *AdminHandler) Users(c *gin.Context) {
	sessUser := session.GetUser(c)

	q := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.Query("limit"))
	if pageSize < 1 {
		pageSize = 10
	}

	userDb := database.DB.Model(&models.User{})
	if q != "" {
		like := "%" + q + "%"
		userDb = userDb.Where("name ILIKE ? OR email ILIKE ? OR phone ILIKE ? OR role ILIKE ? OR address ILIKE ?", like, like, like, like, like)
	}

	var totalUsers int64
	userDb.Count(&totalUsers)
	paging := calcPagination(totalUsers, page, pageSize)

	var users []models.User
	userDb.Order("created_at DESC").Offset((paging.CurrentPage - 1) * pageSize).Limit(pageSize).Find(&users)

	render.HTML(c, http.StatusOK, "admin/users.html", gin.H{
		"Users":  users,
		"Q":      q,
		"Paging": paging,
		"User":   sessUser,
	})
}

func (h *AdminHandler) EditUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, _ := strconv.Atoi(userIDStr)

	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	phone := strings.TrimSpace(c.PostForm("phone"))
	role := c.PostForm("role")
	address := strings.TrimSpace(c.PostForm("address"))
	latStr := strings.TrimSpace(c.PostForm("lat"))
	lngStr := strings.TrimSpace(c.PostForm("lng"))

	var latVal, lngVal *float64
	if latStr != "" {
		if v, err := strconv.ParseFloat(latStr, 64); err == nil {
			latVal = &v
		}
	}
	if lngStr != "" {
		if v, err := strconv.ParseFloat(lngStr, 64); err == nil {
			lngVal = &v
		}
	}

	updates := map[string]interface{}{
		"name":    name,
		"email":   email,
		"phone":   phone,
		"role":    role,
		"address": address,
		"lat":     latVal,
		"lng":     lngVal,
	}

	database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, _ := strconv.Atoi(userIDStr)
	newRole := c.PostForm("role")

	validRoles := map[string]bool{
		models.RoleAdmin:      true,
		models.RoleStaff:      true,
		models.RoleRestaurant: true,
		models.RoleRider:      true,
		models.RoleCustomer:   true,
	}

	if validRoles[newRole] {
		database.DB.Model(&models.User{}).Where("id = ?", userID).Update("role", newRole)
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

func (h *AdminHandler) Restaurants(c *gin.Context) {
	sessUser := session.GetUser(c)

	q := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.Query("limit"))
	if pageSize < 1 {
		pageSize = 10
	}

	restDb := database.DB.Model(&models.Restaurant{})
	if q != "" {
		like := "%" + q + "%"
		restDb = restDb.Where("name ILIKE ? OR address ILIKE ? OR description ILIKE ?", like, like, like)
	}

	var totalRestaurants int64
	restDb.Count(&totalRestaurants)
	paging := calcPagination(totalRestaurants, page, pageSize)

	var restaurants []models.Restaurant
	restDb.Preload("Owner").Order("created_at DESC").Offset((paging.CurrentPage - 1) * pageSize).Limit(pageSize).Find(&restaurants)

	render.HTML(c, http.StatusOK, "admin/restaurants.html", gin.H{
		"Restaurants": restaurants,
		"Q":           q,
		"Paging":      paging,
		"User":        sessUser,
	})
}

func (h *AdminHandler) EditRestaurant(c *gin.Context) {
	restIDStr := c.Param("id")
	restID, _ := strconv.Atoi(restIDStr)

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
	_ = database.DB.First(&restaurant, restID)

	imageURL := restaurant.ImagePath
	file, header, errFile := c.Request.FormFile("image")
	if errFile == nil && file != nil && h.storageSvc != nil {
		defer file.Close()
		if uploadedURL, uploadErr := h.storageSvc.UploadPublicFile(file, header.Filename, header.Header.Get("Content-Type")); uploadErr == nil {
			imageURL = uploadedURL
		}
	}

	updates := map[string]interface{}{
		"name":                     name,
		"description":              description,
		"address":                  address,
		"lat":                      lat,
		"lng":                      lng,
		"delivery_base_fee_satang": baseFeeSatang,
		"delivery_per_km_satang":   perKmSatang,
		"is_open":                  isOpen,
	}
	if imageURL != "" {
		updates["image_path"] = imageURL
	}

	database.DB.Model(&models.Restaurant{}).Where("id = ?", restID).Updates(updates)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/admin/restaurants")
}

func (h *AdminHandler) Orders(c *gin.Context) {
	sessUser := session.GetUser(c)

	q := strings.TrimSpace(c.Query("q"))
	statusFilter := strings.TrimSpace(c.Query("status"))
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.Query("limit"))
	if pageSize < 1 {
		pageSize = 10
	}

	orderDb := database.DB.Model(&models.Order{}).
		Joins("Customer").
		Joins("Restaurant")

	if statusFilter != "" && statusFilter != "all" {
		orderDb = orderDb.Where("orders.status = ?", statusFilter)
	}

	if q != "" {
		like := "%" + q + "%"
		orderDb = orderDb.Where(`
			CAST(orders.id AS TEXT) LIKE ? OR 
			"Customer".name ILIKE ? OR 
			"Customer".email ILIKE ? OR 
			"Customer".phone ILIKE ? OR 
			"Restaurant".name ILIKE ? OR 
			orders.delivery_address ILIKE ? OR 
			orders.status::text ILIKE ? OR 
			CAST(orders.total_satang AS TEXT) LIKE ? OR 
			CAST(orders.total_satang / 100 AS TEXT) LIKE ? OR 
			TO_CHAR(orders.created_at, 'YYYY-MM-DD HH24:MI:SS DD Mon YYYY') ILIKE ?
		`, like, like, like, like, like, like, like, like, like, like)
	}

	var totalOrders int64
	orderDb.Count(&totalOrders)
	paging := calcPagination(totalOrders, page, pageSize)

	var orders []models.Order
	orderDb.Preload("Customer").Preload("Restaurant").Preload("Rider").
		Order("orders.created_at DESC").
		Offset((paging.CurrentPage - 1) * pageSize).
		Limit(pageSize).
		Find(&orders)

	render.HTML(c, http.StatusOK, "admin/orders.html", gin.H{
		"Orders":       orders,
		"Q":            q,
		"StatusFilter": statusFilter,
		"Paging":       paging,
		"User":         sessUser,
	})
}

func (h *AdminHandler) UpdateOrderStatus(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, _ := strconv.Atoi(orderIDStr)
	newStatus := c.PostForm("status")

	validStatuses := map[string]bool{
		models.StatusPending:        true,
		models.StatusConfirmed:      true,
		models.StatusReadyForPickup: true,
		models.StatusInDelivery:     true,
		models.StatusDelivered:      true,
		models.StatusCancelled:      true,
	}

	if validStatuses[newStatus] {
		database.DB.Model(&models.Order{}).Where("id = ?", orderID).Update("status", newStatus)
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/admin/orders")
}

func (h *AdminHandler) UpdateOrderDetail(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, _ := strconv.Atoi(orderIDStr)

	newStatus := c.PostForm("status")
	deliveryAddress := strings.TrimSpace(c.PostForm("delivery_address"))
	latStr := strings.TrimSpace(c.PostForm("delivery_lat"))
	lngStr := strings.TrimSpace(c.PostForm("delivery_lng"))
	note := strings.TrimSpace(c.PostForm("note"))

	latVal, _ := strconv.ParseFloat(latStr, 64)
	lngVal, _ := strconv.ParseFloat(lngStr, 64)

	updates := map[string]interface{}{
		"status":           newStatus,
		"delivery_address": deliveryAddress,
		"delivery_lat":     latVal,
		"delivery_lng":     lngVal,
		"note":             note,
	}

	database.DB.Model(&models.Order{}).Where("id = ?", orderID).Updates(updates)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/orders/%d", orderID))
}
