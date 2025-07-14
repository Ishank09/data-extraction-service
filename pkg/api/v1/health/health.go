package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Health struct{}

func New() Health {
	return Health{}
}

func (s Health) Handler(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]any{
		"message": "healthy",
	})
}
