package render

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	templateCache = make(map[string]*template.Template)
	cacheMutex    sync.RWMutex
)

// HTML renders a page template combined with the base layout.
func HTML(c *gin.Context, status int, pageFile string, data gin.H) {
	var tmpl *template.Template

	funcs := template.FuncMap{
		"formatTHB": func(v interface{}) string {
			var satang float64
			switch val := v.(type) {
			case int:
				satang = float64(val)
			case int64:
				satang = float64(val)
			case float64:
				satang = val
			case float32:
				satang = float64(val)
			case uint:
				satang = float64(val)
			case uint64:
				satang = float64(val)
			default:
				satang = 0
			}
			baht := satang / 100.0
			return fmt.Sprintf("฿%.2f", baht)
		},
		"formatTHBRaw": func(v interface{}) string {
			var satang float64
			switch val := v.(type) {
			case int:
				satang = float64(val)
			case int64:
				satang = float64(val)
			case float64:
				satang = val
			case float32:
				satang = float64(val)
			case uint:
				satang = float64(val)
			case uint64:
				satang = float64(val)
			default:
				satang = 0
			}
			baht := satang / 100.0
			return fmt.Sprintf("%.2f", baht)
		},
		"mul": func(a, b int) int {
			return a * b
		},
	}

	basePath := filepath.Join("templates", "layout", "base.html")
	targetPath := filepath.Join("templates", pageFile)

	t, err := template.New("layout/base").Funcs(funcs).ParseFiles(basePath, targetPath)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Template parse error (%s): %v", pageFile, err))
		return
	}
	tmpl = t

	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(c.Writer, "layout/base", data); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Template execution error (%s): %v", pageFile, err))
	}
}
