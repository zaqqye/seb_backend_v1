package models

import "time"

// AppConfig stores arbitrary key/value settings managed via backend or admin UI.
type AppConfig struct {
	Key         string    `gorm:"size:128;primaryKey"`
	Value       string    `gorm:"type:text"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
