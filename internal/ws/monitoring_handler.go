package ws

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"github.com/zaqqye/seb_backend_v1/internal/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins; rely on JWT auth.
		return true
	},
}

func MonitoringHandler(db *gorm.DB, hub *MonitoringHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		uVal, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		user := uVal.(models.User)
		role := strings.ToLower(user.Role)
		if role != "admin" && role != "pengawas" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		allowAll := role == "admin"
		allowedRooms := map[string]struct{}{}
		if !allowAll {
			var assignments []models.RoomSupervisor
			if err := db.Where("user_id_ref = ?", user.ID).Find(&assignments).Error; err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if len(assignments) == 0 {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no rooms assigned"})
				return
			}
			for _, asg := range assignments {
				allowedRooms[asg.RoomIDRef] = struct{}{}
			}
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := newMonitoringClient(hub, conn, allowedRooms, allowAll)
		hub.register <- client

		go client.writePump()
		client.readPump()
	}
}
