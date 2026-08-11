package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/render"
	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/dreamoutbox/delivery-app/pkg/geo"
	"github.com/gin-gonic/gin"
)

type CustomerHandler struct{}

func NewCustomerHandler() *CustomerHandler {
	return &CustomerHandler{}
}

func (h *CustomerHandler) ShowRestaurant(c *gin.Context) {
	idStr := c.Param("id")
	restID, err := strconv.Atoi(idStr)
	if err != nil {
		render.HTML(c, http.StatusNotFound, "error.html", gin.H{"Title": "Not Found", "Message": "Invalid restaurant ID."})
		return
	}

	var restaurant models.Restaurant
	if err := database.DB.Preload("MenuItems").First(&restaurant, restID).Error; err != nil {
		render.HTML(c, http.StatusNotFound, "error.html", gin.H{"Title": "Not Found", "Message": "Restaurant not found."})
		return
	}

	basketRestID, basketItems := session.GetBasket(c)
	basketCount := session.GetBasketCount(c)

	render.HTML(c, http.StatusOK, "customer/restaurant_detail.html", gin.H{
		"Restaurant":   restaurant,
		"BasketRestID": basketRestID,
		"BasketItems":  basketItems,
		"BasketCount":  basketCount,
		"User":         session.GetUser(c),
	})
}

func (h *CustomerHandler) AddToBasket(c *gin.Context) {
	itemIDStr := c.PostForm("menu_item_id")
	itemID, _ := strconv.Atoi(itemIDStr)
	action := c.PostForm("action")

	var menuItem models.MenuItem
	if err := database.DB.First(&menuItem, itemID).Error; err != nil && action != "clear" {
		c.String(http.StatusBadRequest, "Invalid item")
		return
	}

	currentRestID, items := session.GetBasket(c)

	if action == "clear" {
		_ = session.ClearBasket(c)
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	if currentRestID != 0 && currentRestID != menuItem.RestaurantID && len(items) > 0 {
		render.HTML(c, http.StatusConflict, "customer/switch_modal.html", gin.H{
			"NewItem":       menuItem,
			"NewRestID":     menuItem.RestaurantID,
			"CurrentRestID": currentRestID,
		})
		return
	}

	if items == nil {
		items = make(map[uint]int)
	}

	if action == "remove" {
		if qty, ok := items[uint(itemID)]; ok {
			if qty > 1 {
				items[uint(itemID)] = qty - 1
			} else {
				delete(items, uint(itemID))
			}
		}
	} else {
		items[uint(itemID)] += 1
	}

	if len(items) == 0 {
		_ = session.ClearBasket(c)
	} else {
		_ = session.SaveBasket(c, menuItem.RestaurantID, items)
	}

	if c.GetHeader("HX-Request") == "true" {
		basketCount := session.GetBasketCount(c)
		c.String(http.StatusOK, fmt.Sprintf(`<span class="position-absolute top-0 start-100 translate-middle badge rounded-pill bg-danger" id="basket-badge">%d</span>`, basketCount))
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/restaurant/%d", menuItem.RestaurantID))
}

type BasketItemView struct {
	MenuItem    models.MenuItem
	Quantity    int
	TotalSatang int
}

func (h *CustomerHandler) ShowCart(c *gin.Context) {
	sessUser := session.GetUser(c)

	restID, items := session.GetBasket(c)
	if restID == 0 || len(items) == 0 {
		render.HTML(c, http.StatusOK, "customer/cart.html", gin.H{
			"IsEmpty": true,
			"User":    sessUser,
		})
		return
	}

	var restaurant models.Restaurant
	database.DB.First(&restaurant, restID)

	var cartItems []BasketItemView
	subtotalSatang := 0

	for itemID, qty := range items {
		var menuItem models.MenuItem
		if err := database.DB.First(&menuItem, itemID).Error; err == nil {
			itemTotal := menuItem.PriceSatang * qty
			subtotalSatang += itemTotal
			cartItems = append(cartItems, BasketItemView{
				MenuItem:    menuItem,
				Quantity:    qty,
				TotalSatang: itemTotal,
			})
		}
	}

	var userModel models.User
	var deliveryFeeSatang int
	var distanceKm float64
	hasAddress := false

	if sessUser != nil {
		if err := database.DB.First(&userModel, sessUser.ID).Error; err == nil {
			if userModel.Address != "" && userModel.Lat != nil && userModel.Lng != nil {
				hasAddress = true
				distanceKm = geo.CalculateDistanceKm(restaurant.Lat, restaurant.Lng, *userModel.Lat, *userModel.Lng)
				deliveryFeeSatang = geo.CalculateDeliveryFeeSatang(
					restaurant.DeliveryBaseFeeSatang,
					restaurant.DeliveryPerKmSatang,
					restaurant.Lat, restaurant.Lng,
					*userModel.Lat, *userModel.Lng,
				)
			}
		}
	}

	totalSatang := subtotalSatang + deliveryFeeSatang

	render.HTML(c, http.StatusOK, "customer/cart.html", gin.H{
		"IsEmpty":           false,
		"Restaurant":        restaurant,
		"CartItems":         cartItems,
		"SubtotalSatang":    subtotalSatang,
		"DeliveryFeeSatang": deliveryFeeSatang,
		"TotalSatang":       totalSatang,
		"DistanceKm":        fmt.Sprintf("%.1f", distanceKm),
		"HasAddress":        hasAddress,
		"User":              userModel,
	})
}

func (h *CustomerHandler) PlaceOrder(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var user models.User
	if err := database.DB.First(&user, sessUser.ID).Error; err != nil || user.Address == "" || user.Lat == nil || user.Lng == nil {
		c.Redirect(http.StatusFound, "/profile")
		return
	}

	restID, items := session.GetBasket(c)
	if restID == 0 || len(items) == 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var restaurant models.Restaurant
	if err := database.DB.First(&restaurant, restID).Error; err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	subtotalSatang := 0
	var orderItems []models.OrderItem

	for itemID, qty := range items {
		var menuItem models.MenuItem
		if err := database.DB.First(&menuItem, itemID).Error; err == nil {
			itemTotal := menuItem.PriceSatang * qty
			subtotalSatang += itemTotal
			orderItems = append(orderItems, models.OrderItem{
				MenuItemID:  menuItem.ID,
				Name:        menuItem.Name,
				PriceSatang: menuItem.PriceSatang,
				Quantity:    qty,
			})
		}
	}

	deliveryFeeSatang := geo.CalculateDeliveryFeeSatang(
		restaurant.DeliveryBaseFeeSatang,
		restaurant.DeliveryPerKmSatang,
		restaurant.Lat, restaurant.Lng,
		*user.Lat, *user.Lng,
	)

	totalSatang := subtotalSatang + deliveryFeeSatang
	note := c.PostForm("note")

	order := models.Order{
		CustomerID:        user.ID,
		RestaurantID:      restaurant.ID,
		Status:            models.StatusPending,
		DeliveryAddress:   user.Address,
		DeliveryLat:       *user.Lat,
		DeliveryLng:       *user.Lng,
		SubtotalSatang:    subtotalSatang,
		DeliveryFeeSatang: deliveryFeeSatang,
		TotalSatang:       totalSatang,
		Note:              note,
		OrderItems:        orderItems,
	}

	if err := database.DB.Create(&order).Error; err != nil {
		render.HTML(c, http.StatusInternalServerError, "error.html", gin.H{"Title": "Order Failed", "Message": "Could not place order."})
		return
	}

	_ = session.ClearBasket(c)

	c.Redirect(http.StatusFound, fmt.Sprintf("/orders/%d", order.ID))
}

func (h *CustomerHandler) ListOrders(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var orders []models.Order
	database.DB.Preload("Restaurant").Preload("Rider").
		Where("customer_id = ?", sessUser.ID).
		Order("created_at DESC").
		Find(&orders)

	render.HTML(c, http.StatusOK, "customer/orders.html", gin.H{
		"Orders": orders,
		"User":   sessUser,
	})
}

func (h *CustomerHandler) OrderDetail(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	idStr := c.Param("id")
	orderID, _ := strconv.Atoi(idStr)

	var order models.Order
	if err := database.DB.Preload("Restaurant").Preload("Customer").Preload("Rider").Preload("OrderItems").First(&order, orderID).Error; err != nil {
		render.HTML(c, http.StatusNotFound, "error.html", gin.H{"Title": "Order Not Found", "Message": "Order details could not be found."})
		return
	}

	if sessUser.Role == models.RoleCustomer && order.CustomerID != sessUser.ID {
		render.HTML(c, http.StatusForbidden, "error.html", gin.H{"Title": "Forbidden", "Message": "You cannot view this order."})
		return
	}

	render.HTML(c, http.StatusOK, "customer/order_detail.html", gin.H{
		"Order": order,
		"User":  sessUser,
	})
}
