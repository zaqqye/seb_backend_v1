package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type Room struct {
    ID        string    `gorm:"type:uuid;primaryKey"`
    Name      string    `gorm:"uniqueIndex"`
    Active    bool
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (r *Room) BeforeCreate(tx *gorm.DB) (err error) {
    if r.ID == "" {
        r.ID = uuid.NewString()
    }
    return nil
}
