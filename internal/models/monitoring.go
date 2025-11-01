package models

import "time"

// StudentStatus stores per-student runtime/app status for monitoring.
// One row per siswa user.
type StudentStatus struct {
    ID              uint       `gorm:"primaryKey"`
    UserIDRef       uint       `gorm:"uniqueIndex"`
    AppVersion      string     `gorm:"size:64"`
    Locked          bool       `gorm:"index"`
    BlockedFromExam bool       `gorm:"index"`
    ForceLogoutAt   *time.Time `gorm:"index"`
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

