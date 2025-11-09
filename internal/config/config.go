package config

import "os"

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
    // Moodle SSO / OAuth
    MoodleSSOClientID     string
    MoodleSSOClientSecret string
    MoodleSSOLoginURL     string
    MoodleSSOSecret       string
}

func Load() *Config {
    return &Config{
        Port:                 os.Getenv("PORT"),
        DBHost:               os.Getenv("DB_HOST"),
        DBPort:               os.Getenv("DB_PORT"),
        DBUser:               os.Getenv("DB_USER"),
        DBPassword:           os.Getenv("DB_PASSWORD"),
        DBName:               os.Getenv("DB_NAME"),
        DBSSLMode:            os.Getenv("DB_SSLMODE"),
        JWTSecret:            os.Getenv("JWT_SECRET"),
        JWTExpiresIn:         os.Getenv("JWT_EXPIRES_IN"),
        AdminEmail:           os.Getenv("ADMIN_EMAIL"),
        AdminPassword:        os.Getenv("ADMIN_PASSWORD"),
        AdminFullName:        os.Getenv("ADMIN_FULL_NAME"),
        LayoutVersion:        os.Getenv("LAYOUT_VERSION"),
        MinAppVersionAndroid: os.Getenv("MIN_APP_VERSION_ANDROID"),
        MinAppVersionIOS:     os.Getenv("MIN_APP_VERSION_IOS"),
        SDUIHMACSecret:       os.Getenv("SDUI_HMAC_SECRET"),
        AccessTokenTTLMinutes: firstNonEmpty(
            os.Getenv("ACCESS_TOKEN_TTL_MINUTES"),
            os.Getenv("JWT_EXPIRES_IN"),
        ),
        RefreshTokenTTLDays: os.Getenv("REFRESH_TOKEN_TTL_DAYS"),
        RefreshJWTSecret: firstNonEmpty(
            os.Getenv("REFRESH_JWT_SECRET"),
            os.Getenv("JWT_SECRET"),
        ),
        MoodleSSOClientID:     os.Getenv("MOODLE_SSO_CLIENT_ID"),
        MoodleSSOClientSecret: os.Getenv("MOODLE_SSO_CLIENT_SECRET"),
        MoodleSSOLoginURL:     os.Getenv("MOODLE_SSO_LOGIN_URL"),
        MoodleSSOSecret:       firstNonEmpty(os.Getenv("MOODLE_SSO_SECRET"), os.Getenv("JWT_SECRET")),
    }
}

func firstNonEmpty(values ...string) string {
    for _, v := range values {
        if v != "" {
            return v
        }
    }
    return ""
}
