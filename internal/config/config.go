package config

import (
    "os"
)

type Config struct {
    Port         string
    DBHost       string
    DBPort       string
    DBUser       string
    DBPassword   string
    DBName       string
    DBSSLMode    string
    JWTSecret    string
    JWTExpiresIn string // minutes
    AdminEmail    string
    AdminPassword string
    AdminFullName string
}

func Load() *Config {
    return &Config{
        Port:         getenv("PORT", "8080"),
        DBHost:       getenv("DB_HOST", "localhost"),
        DBPort:       getenv("DB_PORT", "5432"),
        DBUser:       getenv("DB_USER", "postgres"),
        DBPassword:   getenv("DB_PASSWORD", "postgres"),
        DBName:       getenv("DB_NAME", "seb_db"),
        DBSSLMode:    getenv("DB_SSLMODE", "disable"),
        JWTSecret:    getenv("JWT_SECRET", "supersecret_change_me"),
        JWTExpiresIn: getenv("JWT_EXPIRES_IN", "60"),
        AdminEmail:    getenv("ADMIN_EMAIL", "admin@example.com"),
        AdminPassword: getenv("ADMIN_PASSWORD", "admin123"),
        AdminFullName: getenv("ADMIN_FULL_NAME", "Administrator"),
    }
}

func getenv(key, fallback string) string {
    v := os.Getenv(key)
    if v == "" {
        return fallback
    }
    return v
}
