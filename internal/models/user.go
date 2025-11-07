package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type User struct {
    ID        string    `gorm:"type:uuid;primaryKey"`
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

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
    if u.ID == "" {
        u.ID = uuid.NewString()
    }
    return nil
}
