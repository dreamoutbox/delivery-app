package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	"testing"

	"github.com/dreamoutbox/delivery-app/internal/config"
	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{SessionSecret: "test-secret-key-32-bytes-long-12345"}
	session.InitStore(cfg)

	r := gin.New()
	authHandler := NewAuthHandler()

	r.GET("/login", authHandler.ShowLogin)
	r.POST("/login", authHandler.Login)
	r.POST("/logout", authHandler.Logout)

	return r
}

func TestLoginValidation(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	form := url.Values{}
	form.Add("email", "")
	form.Add("password", "")

	req, _ := http.NewRequest("POST", "/login", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
