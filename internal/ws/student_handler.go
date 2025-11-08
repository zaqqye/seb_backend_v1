package ws

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zaqqye/seb_backend_v1/internal/models"
)

func StudentHandler(hubs *Hubs) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hubs == nil || hubs.Student == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "realtime not available"})
			return
		}
		uVal, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		user := uVal.(models.User)
		if strings.ToLower(user.Role) != "siswa" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := newStudentClient(hubs.Student, conn, user.ID)
		hubs.Student.register <- client

		go client.writePump()
		client.readPump()
	}
}
