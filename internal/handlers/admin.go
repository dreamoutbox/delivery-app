package handlers

import (
	"net/http"
	"strconv"

	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/gin-gonic/gin"
)

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

	var users []models.User
	database.DB.Order("created_at DESC").Limit(20).Find(&users)

	var restaurants []models.Restaurant
	database.DB.Preload("Owner").Order("created_at DESC").Limit(20).Find(&restaurants)

	var orders []models.Order
	database.DB.Preload("Customer").Preload("Restaurant").Preload("Rider").Order("created_at DESC").Limit(20).Find(&orders)

	c.HTML(http.StatusOK, "admin/dashboard.html", gin.H{
		"TotalUsers":          totalUsers,
		"TotalRestaurants":    totalRestaurants,
		"TotalOrders":         totalOrders,
		"TotalRevenueSatang": totalRevenueSatang,
		"Users":              users,
		"Restaurants":        restaurants,
		"Orders":             orders,
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
