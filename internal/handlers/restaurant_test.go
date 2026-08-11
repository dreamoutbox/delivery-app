package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dreamoutbox/delivery-app/internal/config"
	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRestaurantDashboardRedirectUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{SessionSecret: "test-secret-key-32-bytes-long-12345"}
	session.InitStore(cfg)

	r := gin.New()
	restHandler := NewRestaurantHandler(nil)
	r.GET("/restaurant", restHandler.Dashboard)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/restaurant", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}
