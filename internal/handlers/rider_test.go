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

func TestRiderDashboardUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{SessionSecret: "test-secret-key-32-bytes-long-12345"}
	session.InitStore(cfg)

	r := gin.New()
	riderHandler := NewRiderHandler(nil)
	r.GET("/rider", riderHandler.Dashboard)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/rider", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}
