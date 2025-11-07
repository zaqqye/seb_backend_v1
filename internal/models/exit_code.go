package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type ExitCode struct {
    ID               string     `gorm:"type:uuid;primaryKey"`
    UserIDRef        string
    StudentUserIDRef string     `gorm:"index"`
    RoomIDRef        *string
    Code             string     `gorm:"uniqueIndex"`
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
    UserIDRef string    `gorm:"uniqueIndex:uniq_user_room"`
    RoomIDRef string    `gorm:"uniqueIndex:uniq_user_room"`
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
    UserIDRef string    `gorm:"uniqueIndex:uniq_student_room"`
    RoomIDRef string    `gorm:"uniqueIndex:uniq_student_room"`
    CreatedAt time.Time
}

func (rs *RoomStudent) BeforeCreate(tx *gorm.DB) (err error) {
    if rs.ID == "" {
        rs.ID = uuid.NewString()
    }
    return nil
}
