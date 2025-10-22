package database

import (
    "fmt"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/config"
    "github.com/zaqqye/seb_backend_v1/internal/models"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%s sslmode=%s client_encoding=UTF8 TimeZone=UTC",
        cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
    )
    return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func Migrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.User{},
        &models.Room{},
        &models.Major{},
        &models.ExitCode{},
        &models.RoomSupervisor{},
        &models.RoomStudent{},
        &models.SduiScreen{},
    )
}
