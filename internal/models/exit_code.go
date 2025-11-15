package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type ExitCode struct {
    ID               string     `gorm:"type:uuid;primaryKey"`
    UserIDRef        string     `gorm:"type:uuid;index"`
    StudentUserIDRef *string    `gorm:"type:uuid;index"`
    RoomIDRef        *string    `gorm:"type:uuid;index"`
    Code             string     `gorm:"uniqueIndex"`
    Reusable         bool       `gorm:"index"`
    UsedAt           *time.Time `gorm:"index"`
    CreatedAt        time.Time
}

func (e *ExitCode) BeforeCreate(tx *gorm.DB) (err error) {
    if e.ID == "" {
        e.ID = uuid.NewString()
    }
    return nil
}

// RoomSupervisor maps a pengawas/admin user to rooms they supervise.
// Admins are allowed everywhere by role; this mapping is primarily for pengawas scope.
type RoomSupervisor struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	UserIDRef string    `gorm:"type:uuid;uniqueIndex:uniq_user_room"`
	RoomIDRef string    `gorm:"type:uuid;uniqueIndex:uniq_user_room;index"`
	CreatedAt time.Time
}

func (rs *RoomSupervisor) BeforeCreate(tx *gorm.DB) (err error) {
    if rs.ID == "" {
        rs.ID = uuid.NewString()
    }
    return nil
}

// RoomStudent maps a siswa user to rooms they belong to.
type RoomStudent struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	UserIDRef string    `gorm:"type:uuid;uniqueIndex:uniq_student_room"`
	RoomIDRef string    `gorm:"type:uuid;uniqueIndex:uniq_student_room;index"`
	CreatedAt time.Time
}

func (rs *RoomStudent) BeforeCreate(tx *gorm.DB) (err error) {
    if rs.ID == "" {
        rs.ID = uuid.NewString()
    }
    return nil
}
