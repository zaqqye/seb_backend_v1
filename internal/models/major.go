package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type Major struct {
    ID        string    `gorm:"type:uuid;primaryKey"`
    Code      string    `gorm:"uniqueIndex"`
    Name      string
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (m *Major) BeforeCreate(tx *gorm.DB) (err error) {
    if m.ID == "" {
        m.ID = uuid.NewString()
    }
    return nil
}
