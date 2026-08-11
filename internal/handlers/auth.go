package handlers

import (
	"net/http"
	"strings"

	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) ShowRegister(c *gin.Context) {
	if session.GetUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "register.html", gin.H{})
}

func (h *AuthHandler) Register(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	password := c.PostForm("password")
	role := c.PostForm("role")

	if name == "" || email == "" || password == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Error": "Name, Email, and Password are required.",
			"Name":  name,
			"Email": email,
		})
		return
	}

	// Validate role (default to customer)
	validRoles := map[string]bool{
		models.RoleCustomer:   true,
		models.RoleRestaurant: true,
		models.RoleRider:      true,
	}
	if !validRoles[role] {
		role = models.RoleCustomer
	}

	// Check email uniqueness
	var count int64
	database.DB.Model(&models.User{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		c.HTML(http.StatusConflict, "register.html", gin.H{
			"Error": "Email address is already registered.",
			"Name":  name,
			"Email": email,
		})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{"Error": "Failed to encrypt password."})
		return
	}

	user := models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashed),
		Role:         role,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{"Error": "Failed to create user account."})
		return
	}

	// Log user in automatically
	_ = session.SetUser(c, &user)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	if session.GetUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

func (h *AuthHandler) Login(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	password := c.PostForm("password")

	if email == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"Error": "Email and Password are required.",
			"Email": email,
		})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Error": "Invalid email or password.",
			"Email": email,
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Error": "Invalid email or password.",
			"Email": email,
		})
		return
	}

	_ = session.SetUser(c, &user)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	_ = session.Clear(c)
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/login")
		c.Status(http.StatusOK)
		return
	}
	c.Redirect(http.StatusFound, "/login")
}
