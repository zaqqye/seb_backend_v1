package controllers

import (
	"errors"
	"log"

	"gorm.io/gorm"

	"github.com/zaqqye/seb_backend_v1/internal/models"
	"github.com/zaqqye/seb_backend_v1/internal/ws"
)

func broadcastStudentStatus(db *gorm.DB, hub *ws.MonitoringHub, studentID string) {
	if hub == nil {
		return
	}
	var st models.StudentStatus
	if err := db.Where("user_id_ref = ?", studentID).First(&st).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("monitoring broadcast: %v", err)
		}
		return
	}
	var room models.RoomStudent
	var roomID *string
	if err := db.Where("user_id_ref = ?", studentID).First(&room).Error; err == nil {
		roomID = &room.RoomIDRef
	}
	payload := ws.MonitoringPayload{
		StudentID:       studentID,
		RoomID:          roomID,
		Locked:          st.Locked,
		BlockedFromExam: st.BlockedFromExam,
		UpdatedAt:       st.UpdatedAt,
		ForceLogoutAt:   st.ForceLogoutAt,
		LastAppVersion:  st.AppVersion,
	}
	hub.Broadcast(payload)
}
