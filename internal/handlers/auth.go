package handlers

import (
	"net/http"
	"strings"

	"github.com/dreamoutbox/delivery-app/internal/database"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/dreamoutbox/delivery-app/internal/render"
	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func getRoleRedirectURL(role string) string {
	switch role {
	case models.RoleAdmin, models.RoleStaff:
		return "/admin"
	case models.RoleRestaurant:
		return "/restaurant"
	case models.RoleRider:
		return "/rider"
	default:
		return "/"
	}
}

func (h *AuthHandler) ShowRegister(c *gin.Context) {
	if u := session.GetUser(c); u != nil {
		c.Redirect(http.StatusFound, getRoleRedirectURL(u.Role))
		return
	}
	render.HTML(c, http.StatusOK, "auth/register.html", gin.H{})
}

func (h *AuthHandler) Register(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	password := c.PostForm("password")
	role := c.PostForm("role")

	if name == "" || email == "" || password == "" {
		render.HTML(c, http.StatusBadRequest, "auth/register.html", gin.H{
			"Error": "Name, Email, and Password are required.",
			"Name":  name,
			"Email": email,
		})
		return
	}

	validRoles := map[string]bool{
		models.RoleCustomer:   true,
		models.RoleRestaurant: true,
		models.RoleRider:      true,
	}
	if !validRoles[role] {
		role = models.RoleCustomer
	}

	var count int64
	database.DB.Model(&models.User{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		render.HTML(c, http.StatusConflict, "auth/register.html", gin.H{
			"Error": "Email address is already registered.",
			"Name":  name,
			"Email": email,
		})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		render.HTML(c, http.StatusInternalServerError, "auth/register.html", gin.H{"Error": "Failed to encrypt password."})
		return
	}

	user := models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashed),
		Role:         role,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		render.HTML(c, http.StatusInternalServerError, "auth/register.html", gin.H{"Error": "Failed to create user account."})
		return
	}

	_ = session.SetUser(c, &user)

	redirectURL := getRoleRedirectURL(user.Role)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", redirectURL)
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, redirectURL)
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	if u := session.GetUser(c); u != nil {
		c.Redirect(http.StatusFound, getRoleRedirectURL(u.Role))
		return
	}
	render.HTML(c, http.StatusOK, "auth/login.html", gin.H{})
}

func (h *AuthHandler) Login(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	password := c.PostForm("password")

	if email == "" || password == "" {
		render.HTML(c, http.StatusBadRequest, "auth/login.html", gin.H{
			"Error": "Email and Password are required.",
			"Email": email,
		})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		render.HTML(c, http.StatusUnauthorized, "auth/login.html", gin.H{
			"Error": "Invalid email or password.",
			"Email": email,
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		render.HTML(c, http.StatusUnauthorized, "auth/login.html", gin.H{
			"Error": "Invalid email or password.",
			"Email": email,
		})
		return
	}

	_ = session.SetUser(c, &user)

	redirectURL := getRoleRedirectURL(user.Role)

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", redirectURL)
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, redirectURL)
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
