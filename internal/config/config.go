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
    JWTExpiresIn string // minutes (legacy; used as default Access TTL)
    AdminEmail    string
    AdminPassword string
    AdminFullName string
    // SDUI / Remote Config
    LayoutVersion        string
    MinAppVersionAndroid string
    MinAppVersionIOS     string
    SDUIHMACSecret       string
    // Token settings
    AccessTokenTTLMinutes  string // minutes
    RefreshTokenTTLDays    string // days
    RefreshJWTSecret       string
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
        LayoutVersion:        getenv("LAYOUT_VERSION", "1"),
        MinAppVersionAndroid: getenv("MIN_APP_VERSION_ANDROID", "1"),
        MinAppVersionIOS:     getenv("MIN_APP_VERSION_IOS", "1"),
        SDUIHMACSecret:       getenv("SDUI_HMAC_SECRET", ""),
        AccessTokenTTLMinutes: getenv("ACCESS_TOKEN_TTL_MINUTES", getenv("JWT_EXPIRES_IN", "15")),
        RefreshTokenTTLDays:   getenv("REFRESH_TOKEN_TTL_DAYS", "30"),
        RefreshJWTSecret:      getenv("REFRESH_JWT_SECRET", getenv("JWT_SECRET", "supersecret_change_me")),
    }
}

func getenv(key, fallback string) string {
    v := os.Getenv(key)
    if v == "" {
        return fallback
    }
    return v
}
