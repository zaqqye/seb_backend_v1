package models

import (
    "time"
)

type User struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    string    `gorm:"uniqueIndex"`
    FullName  string
    Email     string    `gorm:"uniqueIndex"`
    Password  string
    Role      string
    Kelas     string
    Jurusan   string
    Active    bool
    CreatedAt time.Time
    UpdatedAt time.Time
}

