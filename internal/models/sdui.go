package models

import (
    "time"

    "gorm.io/datatypes"
)

// SduiScreen stores server-driven UI screen payloads.
// Role is optional; empty string means applies to any role.
type SduiScreen struct {
    ID            uint           `gorm:"primaryKey"`
    Name          string         `gorm:"index:uniq_sdui,priority:1"`
    Platform      string         `gorm:"index:uniq_sdui,priority:2"`
    Role          string         `gorm:"index:uniq_sdui,priority:3"`
    SchemaVersion int            `gorm:"default:1"`
    ScreenVersion int            `gorm:"default:1"`
    Active        bool           `gorm:"default:true"`
    Payload       datatypes.JSON `gorm:"type:jsonb"`
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

