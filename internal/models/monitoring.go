package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// StudentStatus stores per-student runtime/app status for monitoring.
// One row per siswa user.
type StudentStatus struct {
    ID              string     `gorm:"type:uuid;primaryKey"`
    UserIDRef       string     `gorm:"type:uuid;uniqueIndex"`
    AppVersion      string     `gorm:"size:64"`
    Locked          bool       `gorm:"index"`
    BlockedFromExam bool       `gorm:"index"`
    ForceLogoutAt   *time.Time `gorm:"index"`
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

func (s *StudentStatus) BeforeCreate(tx *gorm.DB) (err error) {
    if s.ID == "" {
        s.ID = uuid.NewString()
    }
    return nil
}
