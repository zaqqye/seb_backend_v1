package database

import (
    "log"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/example/seb_backend_v1/internal/config"
    "github.com/example/seb_backend_v1/internal/models"
    "github.com/example/seb_backend_v1/internal/utils"
)

func SeedAdmin(db *gorm.DB, cfg *config.Config) error {
    var count int64
    if err := db.Model(&models.User{}).Where("role = ?", "admin").Count(&count).Error; err != nil {
        return err
    }
    if count > 0 {
        return nil
    }

    email := cfg.AdminEmail
    if email == "" {
        email = "admin@example.com"
    }
    fullName := cfg.AdminFullName
    if fullName == "" {
        fullName = "Administrator"
    }
    password := cfg.AdminPassword
    if password == "" {
        password = "admin123"
    }
    hashed, err := utils.HashPassword(password)
    if err != nil {
        return err
    }

    admin := models.User{
        UserID:   uuid.NewString(),
        FullName: fullName,
        Email:    email,
        Password: hashed,
        Role:     "admin",
        Active:   true,
    }
    if err := db.Create(&admin).Error; err != nil {
        return err
    }
    log.Println("Seeded initial admin:", email)
    return nil
}

