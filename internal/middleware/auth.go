package middleware

import (
	"net/http"

	"github.com/dreamoutbox/delivery-app/internal/session"
	"github.com/gin-gonic/gin"
)

// InjectUser adds the session user to Gin Context if logged in
func InjectUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetUser(c)
		if user != nil {
			c.Set("user", user)
		}
		c.Next()
	}
}

// RequireLogin ensures the user is authenticated
func RequireLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetUser(c)
		if user == nil {
			if c.GetHeader("HX-Request") == "true" {
				c.Header("HX-Redirect", "/login")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

// RequireRole ensures the logged-in user has one of the required roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userVal, exists := c.Get("user")
		if !exists || userVal == nil {
			if c.GetHeader("HX-Request") == "true" {
				c.Header("HX-Redirect", "/login")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user := userVal.(*session.SessionUser)
		allowed := false
		for _, r := range roles {
			if user.Role == r {
				allowed = true
				break
			}
		}

		if !allowed {
			if c.GetHeader("HX-Request") == "true" {
				c.String(http.StatusForbidden, "<div class='alert alert-danger'>Access forbidden: insufficient role permission.</div>")
				c.Abort()
				return
			}
			c.HTML(http.StatusForbidden, "error.html", gin.H{
				"Title":   "Forbidden",
				"Message": "You do not have permission to access this page.",
				"User":    user,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
