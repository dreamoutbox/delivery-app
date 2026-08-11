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
}

func calcPagination(total int64, page, pageSize int) PaginationMeta {
	if page < 1 {
		page = 1
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
	}
}

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	sessUser := session.GetUser(c)

	var totalUsers, totalRestaurants, totalOrders int64
	var totalRevenueSatang int64

	database.DB.Model(&models.User{}).Count(&totalUsers)
	database.DB.Model(&models.Restaurant{}).Count(&totalRestaurants)
	database.DB.Model(&models.Order{}).Count(&totalOrders)
	database.DB.Model(&models.Order{}).Where("status = ?", models.StatusDelivered).Select("COALESCE(SUM(total_satang), 0)").Scan(&totalRevenueSatang)

	pageSize := 5

	// 1. Users Search & Pagination
	userQ := strings.TrimSpace(c.Query("user_q"))
	userPage, _ := strconv.Atoi(c.Query("user_page"))
	if userPage < 1 {
		userPage = 1
	}

	userDb := database.DB.Model(&models.User{})
	if userQ != "" {
		like := "%" + userQ + "%"
		userDb = userDb.Where("name ILIKE ? OR email ILIKE ? OR phone ILIKE ? OR role ILIKE ?", like, like, like, like)
	}
	var countUsers int64
	userDb.Count(&countUsers)
	userPaging := calcPagination(countUsers, userPage, pageSize)

	var users []models.User
	userDb.Order("created_at DESC").Offset((userPaging.CurrentPage - 1) * pageSize).Limit(pageSize).Find(&users)

	// 2. Restaurants Search & Pagination
	restQ := strings.TrimSpace(c.Query("rest_q"))
	restPage, _ := strconv.Atoi(c.Query("rest_page"))
	if restPage < 1 {
		restPage = 1
	}

	restDb := database.DB.Model(&models.Restaurant{})
	if restQ != "" {
		like := "%" + restQ + "%"
		restDb = restDb.Where("name ILIKE ? OR address ILIKE ? OR description ILIKE ?", like, like, like)
	}
	var countRestaurants int64
	restDb.Count(&countRestaurants)
	restPaging := calcPagination(countRestaurants, restPage, pageSize)

	var restaurants []models.Restaurant
	restDb.Preload("Owner").Order("created_at DESC").Offset((restPaging.CurrentPage - 1) * pageSize).Limit(pageSize).Find(&restaurants)

	// 3. Orders Search & Pagination
	orderQ := strings.TrimSpace(c.Query("order_q"))
	orderPage, _ := strconv.Atoi(c.Query("order_page"))
	if orderPage < 1 {
		orderPage = 1
	}

	orderDb := database.DB.Model(&models.Order{})
	if orderQ != "" {
		like := "%" + orderQ + "%"
		orderDb = orderDb.Where("status ILIKE ? OR delivery_address ILIKE ? OR CAST(id AS TEXT) LIKE ?", like, like, like)
	}
	var countOrders int64
	orderDb.Count(&countOrders)
	orderPaging := calcPagination(countOrders, orderPage, pageSize)

	var orders []models.Order
	orderDb.Preload("Customer").Preload("Restaurant").Preload("Rider").Order("created_at DESC").Offset((orderPaging.CurrentPage - 1) * pageSize).Limit(pageSize).Find(&orders)

	render.HTML(c, http.StatusOK, "admin/dashboard.html", gin.H{
		"TotalUsers":          totalUsers,
		"TotalRestaurants":    totalRestaurants,
		"TotalOrders":         totalOrders,
		"TotalRevenueSatang": totalRevenueSatang,
		"Users":              users,
		"UserQ":              userQ,
		"UserPaging":         userPaging,
		"Restaurants":        restaurants,
		"RestQ":              restQ,
		"RestPaging":         restPaging,
		"Orders":             orders,
		"OrderQ":             orderQ,
		"OrderPaging":        orderPaging,
		"User":               sessUser,
	})
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

	c.Redirect(http.StatusFound, "/admin")
}

func (h *AdminHandler) EditUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, _ := strconv.Atoi(userIDStr)

	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	phone := strings.TrimSpace(c.PostForm("phone"))
	role := c.PostForm("role")

	updates := map[string]interface{}{
		"name":  name,
		"email": email,
		"phone": phone,
		"role":  role,
	}

	database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/admin")
}

func (h *AdminHandler) ToggleRestaurantStatus(c *gin.Context) {
	restIDStr := c.Param("id")
	restID, _ := strconv.Atoi(restIDStr)

	var rest models.Restaurant
	if err := database.DB.First(&rest, restID).Error; err == nil {
		database.DB.Model(&rest).Update("is_open", !rest.IsOpen)
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/admin")
}

func (h *AdminHandler) EditRestaurant(c *gin.Context) {
	restIDStr := c.Param("id")
	restID, _ := strconv.Atoi(restIDStr)

	name := strings.TrimSpace(c.PostForm("name"))
	address := strings.TrimSpace(c.PostForm("address"))
	baseFeeStr := strings.TrimSpace(c.PostForm("delivery_base_fee_thb"))
	perKmStr := strings.TrimSpace(c.PostForm("delivery_per_km_thb"))

	baseFeeTHB, _ := strconv.ParseFloat(baseFeeStr, 64)
	perKmTHB, _ := strconv.ParseFloat(perKmStr, 64)

	baseFeeSatang := int(math.Round(baseFeeTHB * 100))
	perKmSatang := int(math.Round(perKmTHB * 100))

	updates := map[string]interface{}{
		"name":                       name,
		"address":                    address,
		"delivery_base_fee_satang": baseFeeSatang,
		"delivery_per_km_satang":   perKmSatang,
	}

	database.DB.Model(&models.Restaurant{}).Where("id = ?", restID).Updates(updates)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/admin")
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

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin#order-%d", orderID))
}
