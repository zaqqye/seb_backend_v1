package models

import "time"

type Room struct {
    ID        uint      `gorm:"primaryKey"`
    Name      string    `gorm:"uniqueIndex"`
    Active    bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
