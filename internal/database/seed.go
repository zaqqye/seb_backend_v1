package database

import (
    "log"

    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/config"
    "github.com/zaqqye/seb_backend_v1/internal/models"
    "github.com/zaqqye/seb_backend_v1/internal/utils"
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

func SeedSDUIScreens(db *gorm.DB) error {
    type seedDef struct{
        Name string
        Platform string
        Role string
        SchemaVersion int
        ScreenVersion int
        Payload string
    }
    screens := []seedDef{}

    for _, s := range screens {
        var count int64
        if err := db.Model(&models.SduiScreen{}).
            Where("name = ? AND platform = ? AND role = ? AND screen_version = ?", s.Name, s.Platform, s.Role, s.ScreenVersion).
            Count(&count).Error; err != nil {
            return err
        }
        if count > 0 {
            continue
        }
        rec := models.SduiScreen{
            Name: s.Name,
            Platform: s.Platform,
            Role: s.Role,
            SchemaVersion: s.SchemaVersion,
            ScreenVersion: s.ScreenVersion,
            Active: true,
            Payload: []byte(s.Payload),
        }
        if err := db.Create(&rec).Error; err != nil {
            return err
        }
    }
    log.Println("No SDUI screens seeded (managed externally)")
    return nil
}
