package models

import "time"

type RefreshToken struct {
    ID               uint      `gorm:"primaryKey"`
    TokenID          string    `gorm:"index"` // jti
    UserIDRef        string    `gorm:"index"`
    TokenHash        string    `gorm:"uniqueIndex"`
    ExpiresAt        time.Time `gorm:"index"`
    RevokedAt        *time.Time
    ReplacedByTokenID *string
    CreatedAt        time.Time
}
