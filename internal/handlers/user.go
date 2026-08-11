package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/gin-gonic/gin"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) ShowProfile(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var user models.User
	if err := database.DB.First(&user, sessUser.ID).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "User Not Found", "Message": "User account could not be loaded."})
		return
	}

	c.HTML(http.StatusOK, "profile.html", gin.H{
		"User": user,
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	sessUser := session.GetUser(c)
	if sessUser == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	phone := strings.TrimSpace(c.PostForm("phone"))
	address := strings.TrimSpace(c.PostForm("address"))
	latStr := strings.TrimSpace(c.PostForm("lat"))
	lngStr := strings.TrimSpace(c.PostForm("lng"))

	if name == "" {
		c.HTML(http.StatusBadRequest, "profile.html", gin.H{
			"Error": "Name cannot be empty.",
			"User":  models.User{ID: sessUser.ID, Name: name, Phone: phone, Address: address},
		})
		return
	}

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
		"phone":   phone,
		"address": address,
		"lat":     latVal,
		"lng":     lngVal,
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", sessUser.ID).Updates(updates).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "profile.html", gin.H{
			"Error": "Failed to update profile.",
		})
		return
	}

	// Update session name if changed
	sessUser.Name = name
	var updatedUser models.User
	database.DB.First(&updatedUser, sessUser.ID)
	_ = session.SetUser(c, &updatedUser)

	if c.GetHeader("HX-Request") == "true" {
		c.Writer.Header().Set("Content-Type", "text/html")
		c.String(http.StatusOK, `<div class="alert alert-success alert-dismissible fade show" role="alert">
			<i class="bi bi-check-circle-fill me-2"></i>Profile & address updated successfully!
			<button type="button" class="btn-close" data-bs-dismiss="alert"></button>
		</div>`)
		return
	}

	c.HTML(http.StatusOK, "profile.html", gin.H{
		"Success": "Profile and address updated successfully!",
		"User":    updatedUser,
	})
}
