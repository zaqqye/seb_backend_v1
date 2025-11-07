package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type RefreshToken struct {
    ID               string    `gorm:"type:uuid;primaryKey"`
    TokenID          string    `gorm:"index"` // jti
    UserIDRef        string    `gorm:"index"`
    TokenHash        string    `gorm:"uniqueIndex"`
    ExpiresAt        time.Time `gorm:"index"`
    RevokedAt        *time.Time
    ReplacedByTokenID *string
    CreatedAt        time.Time
}

func (rt *RefreshToken) BeforeCreate(tx *gorm.DB) (err error) {
    if rt.ID == "" {
        rt.ID = uuid.NewString()
    }
    return nil
}
