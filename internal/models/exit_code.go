package models

import "time"

type ExitCode struct {
    ID               uint       `gorm:"primaryKey"`
    UserIDRef        string
    StudentUserIDRef string     `gorm:"index"`
    RoomIDRef        *uint
    Code             string     `gorm:"uniqueIndex"`
    UsedAt           *time.Time `gorm:"index"`
    CreatedAt        time.Time
}

// RoomSupervisor maps a pengawas/admin user to rooms they supervise.
// Admins are allowed everywhere by role; this mapping is primarily for pengawas scope.
type RoomSupervisor struct {
    ID        uint      `gorm:"primaryKey"`
    UserIDRef string    `gorm:"uniqueIndex:uniq_user_room"`
    RoomIDRef uint      `gorm:"uniqueIndex:uniq_user_room"`
    CreatedAt time.Time
}

// RoomStudent maps a siswa user to rooms they belong to.
type RoomStudent struct {
    ID        uint      `gorm:"primaryKey"`
    UserIDRef string    `gorm:"uniqueIndex:uniq_student_room"`
    RoomIDRef uint      `gorm:"uniqueIndex:uniq_student_room"`
    CreatedAt time.Time
}
