package controllers

import (
	"errors"
	"log"

	"gorm.io/gorm"

	"github.com/zaqqye/seb_backend_v1/internal/models"
	"github.com/zaqqye/seb_backend_v1/internal/ws"
)

func broadcastStudentStatus(db *gorm.DB, hubs *ws.Hubs, studentID string) {
	if hubs == nil {
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
	var roomIDPtr *string
	var roomIDValue string
	roomBlock := ws.MonitoringRoom{}
	if err := db.Where("user_id_ref = ?", studentID).First(&room).Error; err == nil {
		roomIDValue = room.RoomIDRef
		roomIDPtr = &roomIDValue
		roomBlock.ID = room.RoomIDRef
		var roomModel models.Room
		if err := db.Select("name").Where("id = ?", room.RoomIDRef).First(&roomModel).Error; err == nil {
			roomBlock.RoomName = roomModel.Name
		}
	}
	updatedAt := st.UpdatedAt
	payload := ws.MonitoringPayload{
		ID:              studentID,
		StudentID:       studentID,
		RoomID:          roomIDPtr,
		Locked:          st.Locked,
		BlockedFromExam: st.BlockedFromExam,
		UpdatedAt:       st.UpdatedAt,
		ForceLogoutAt:   st.ForceLogoutAt,
		LastAppVersion:  st.AppVersion,
		Monitoring: ws.MonitoringSnapshot{
			ID:              st.ID,
			AppVersion:      st.AppVersion,
			Locked:          st.Locked,
			BlockedFromExam: st.BlockedFromExam,
			ForceLogoutAt:   st.ForceLogoutAt,
			UpdatedAt:       &updatedAt,
		},
		Room: roomBlock,
	}
	if hubs.Monitoring != nil {
		hubs.Monitoring.Broadcast(payload)
	}
	if hubs.Student != nil {
		msg := ws.StudentMessage{
			Type:            "status_update",
			Locked:          st.Locked,
			BlockedFromExam: st.BlockedFromExam,
			ForceLogoutAt:   st.ForceLogoutAt,
			AppVersion:      st.AppVersion,
		}
		hubs.Student.Notify(studentID, msg)
	}
}
