package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/render"
	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/dreamoutbox/delivery-app/internal/storage"
	"github.com/gin-gonic/gin"
)

type RiderHandler struct {
	storageSvc *storage.StorageService
}

func NewRiderHandler(storageSvc *storage.StorageService) *RiderHandler {
	return &RiderHandler{storageSvc: storageSvc}
}

func (h *RiderHandler) Dashboard(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var availableOrders []models.Order
	database.DB.Preload("Restaurant").Preload("Customer").Preload("OrderItems").
		Where("status IN (?, ?) AND (rider_id IS NULL OR rider_id = 0)", models.StatusConfirmed, models.StatusReadyForPickup).
		Order("created_at ASC").
		Find(&availableOrders)

	var myActiveOrders []models.Order
	database.DB.Preload("Restaurant").Preload("Customer").Preload("OrderItems").
		Where("rider_id = ? AND status IN (?, ?, ?)", sessUser.ID, models.StatusConfirmed, models.StatusReadyForPickup, models.StatusInDelivery).
		Order("created_at ASC").
		Find(&myActiveOrders)

	var myCompletedOrders []models.Order
	database.DB.Preload("Restaurant").Preload("Customer").
		Where("rider_id = ? AND status = ?", sessUser.ID, models.StatusDelivered).
		Order("updated_at DESC").
		Limit(10).
		Find(&myCompletedOrders)

	render.HTML(c, http.StatusOK, "rider/dashboard.html", gin.H{
		"AvailableOrders":   availableOrders,
		"MyActiveOrders":    myActiveOrders,
		"MyCompletedOrders": myCompletedOrders,
		"User":              sessUser,
	})
}

func (h *RiderHandler) AcceptOrder(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	orderIDStr := c.Param("id")
	orderID, _ := strconv.Atoi(orderIDStr)

	var order models.Order
	if err := database.DB.Where("id = ? AND (rider_id IS NULL OR rider_id = 0)", orderID).First(&order).Error; err == nil {
		riderID := sessUser.ID
		database.DB.Model(&order).Update("rider_id", riderID)
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/rider")
}

func (h *RiderHandler) StartDelivery(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	orderIDStr := c.Param("id")
	orderID, _ := strconv.Atoi(orderIDStr)

	file, header, err := c.Request.FormFile("proof_image")
	if err != nil || file == nil {
		render.HTML(c, http.StatusBadRequest, "error.html", gin.H{"Title": "Image Required", "Message": "Proof photo is required before starting delivery."})
		return
	}
	defer file.Close()

	proofKey, uploadErr := h.storageSvc.UploadPrivateFile(file, header.Filename, header.Header.Get("Content-Type"))
	if uploadErr != nil {
		render.HTML(c, http.StatusInternalServerError, "error.html", gin.H{"Title": "Upload Failed", "Message": "Failed to upload proof photo."})
		return
	}

	database.DB.Model(&models.Order{}).Where("id = ? AND rider_id = ?", orderID, sessUser.ID).Updates(map[string]interface{}{
		"status":             models.StatusInDelivery,
		"rider_pickup_proof": proofKey,
	})

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/rider")
}

func (h *RiderHandler) CompleteDelivery(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	orderIDStr := c.Param("id")
	orderID, _ := strconv.Atoi(orderIDStr)

	file, header, err := c.Request.FormFile("proof_image")
	if err != nil || file == nil {
		render.HTML(c, http.StatusBadRequest, "error.html", gin.H{"Title": "Image Required", "Message": "Proof photo is required before completing delivery."})
		return
	}
	defer file.Close()

	proofKey, uploadErr := h.storageSvc.UploadPrivateFile(file, header.Filename, header.Header.Get("Content-Type"))
	if uploadErr != nil {
		render.HTML(c, http.StatusInternalServerError, "error.html", gin.H{"Title": "Upload Failed", "Message": "Failed to upload proof photo."})
		return
	}

	database.DB.Model(&models.Order{}).Where("id = ? AND rider_id = ?", orderID, sessUser.ID).Updates(map[string]interface{}{
		"status":               models.StatusDelivered,
		"rider_delivery_proof": proofKey,
	})

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/rider")
}

func (h *RiderHandler) ServeProofPhoto(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	orderIDStr := c.Param("id")
	proofType := c.Param("type")
	orderID, _ := strconv.Atoi(orderIDStr)

	var order models.Order
	if err := database.DB.First(&order, orderID).Error; err != nil {
		c.String(http.StatusNotFound, "Order not found")
		return
	}

	allowed := false
	if sessUser.Role == models.RoleAdmin || sessUser.Role == models.RoleStaff {
		allowed = true
	} else if order.CustomerID == sessUser.ID {
		allowed = true
	} else if order.RiderID != nil && *order.RiderID == sessUser.ID {
		allowed = true
	} else {
		var rest models.Restaurant
		if err := database.DB.First(&rest, order.RestaurantID).Error; err == nil && rest.OwnerID == sessUser.ID {
			allowed = true
		}
	}

	if !allowed {
		c.String(http.StatusForbidden, "Forbidden")
		return
	}

	var proofKey string
	if proofType == "pickup" {
		proofKey = order.RiderPickupProof
	} else {
		proofKey = order.RiderDeliveryProof
	}

	if proofKey == "" {
		c.String(http.StatusNotFound, "No proof photo uploaded")
		return
	}

	body, contentType, err := h.storageSvc.GetPrivateFile(proofKey)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error fetching proof: %v", err))
		return
	}
	defer body.Close()

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "private, max-age=3600")
	_, _ = io.Copy(c.Writer, body)
}
