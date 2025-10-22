package models

import "time"

type Major struct {
    ID        uint      `gorm:"primaryKey"`
    Code      string    `gorm:"uniqueIndex"`
    Name      string
    CreatedAt time.Time
    UpdatedAt time.Time
}

