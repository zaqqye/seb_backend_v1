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
    screens := []seedDef{
        {
            Name: "login", Platform: "android", Role: "", SchemaVersion: 1, ScreenVersion: 1,
            Payload: `{"schema_version":1,"screen_version":1,"name":"login","platform":"android","components":[{"id":"title","type":"text","props":{"text":"Masuk","style":"h1"}},{"id":"email","type":"input","props":{"name":"email","hint":"Email","keyboard":"email"}},{"id":"password","type":"input","props":{"name":"password","hint":"Password","secure":true}},{"id":"loginBtn","type":"button","props":{"text":"Login","style":"primary"},"action":{"type":"http","method":"POST","url":"/api/v1/auth/login","bodyBindings":{"email":"email","password":"password"},"onSuccess":[{"type":"storeToken","from":"response.access_token"},{"type":"navigate","to":"dashboard"}],"onError":[{"type":"toast","message":"Email/Password salah"}]}}]}`,
        },
        {
            Name: "login", Platform: "ios", Role: "", SchemaVersion: 1, ScreenVersion: 1,
            Payload: `{"schema_version":1,"screen_version":1,"name":"login","platform":"ios","components":[{"id":"title","type":"text","props":{"text":"Masuk","style":"h1"}},{"id":"email","type":"input","props":{"name":"email","hint":"Email","keyboard":"email"}},{"id":"password","type":"input","props":{"name":"password","hint":"Password","secure":true}},{"id":"loginBtn","type":"button","props":{"text":"Login","style":"primary"},"action":{"type":"http","method":"POST","url":"/api/v1/auth/login","bodyBindings":{"email":"email","password":"password"},"onSuccess":[{"type":"storeToken","from":"response.access_token"},{"type":"navigate","to":"dashboard"}],"onError":[{"type":"toast","message":"Email/Password salah"}]}}]}`,
        },
        {
            Name: "dashboard", Platform: "android", Role: "", SchemaVersion: 1, ScreenVersion: 1,
            Payload: `{"schema_version":1,"screen_version":1,"name":"dashboard","platform":"android","components":[{"id":"title","type":"text","props":{"text":"Dashboard","style":"h1"}},{"id":"me","type":"button","props":{"text":"Profil Saya"},"action":{"type":"http","auth":"bearer","method":"GET","url":"/api/v1/auth/me"}}]}`,
        },
        {
            Name: "dashboard", Platform: "ios", Role: "", SchemaVersion: 1, ScreenVersion: 1,
            Payload: `{"schema_version":1,"screen_version":1,"name":"dashboard","platform":"ios","components":[{"id":"title","type":"text","props":{"text":"Dashboard","style":"h1"}},{"id":"me","type":"button","props":{"text":"Profil Saya"},"action":{"type":"http","auth":"bearer","method":"GET","url":"/api/v1/auth/me"}}]}`,
        },
        {
            Name: "exit_code", Platform: "android", Role: "", SchemaVersion: 1, ScreenVersion: 1,
            Payload: `{"schema_version":1,"screen_version":1,"name":"exit_code","platform":"android","components":[{"id":"code","type":"input","props":{"name":"code","hint":"Masukkan Exit Code"}},{"id":"submit","type":"button","props":{"text":"Gunakan Kode"},"action":{"type":"http","method":"POST","url":"/api/v1/exit-codes/consume","auth":"bearer","bodyBindings":{"code":"code"},"onSuccess":[{"type":"toast","message":"Kode diterima"},{"type":"navigate","to":"dashboard"}],"onError":[{"type":"toast","message":"Kode tidak valid/terpakai"}]}}]}`,
        },
        {
            Name: "exit_code", Platform: "ios", Role: "", SchemaVersion: 1, ScreenVersion: 1,
            Payload: `{"schema_version":1,"screen_version":1,"name":"exit_code","platform":"ios","components":[{"id":"code","type":"input","props":{"name":"code","hint":"Masukkan Exit Code"}},{"id":"submit","type":"button","props":{"text":"Gunakan Kode"},"action":{"type":"http","method":"POST","url":"/api/v1/exit-codes/consume","auth":"bearer","bodyBindings":{"code":"code"},"onSuccess":[{"type":"toast","message":"Kode diterima"},{"type":"navigate","to":"dashboard"}],"onError":[{"type":"toast","message":"Kode tidak valid/terpakai"}]}}]}`,
        },
    }

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
    log.Println("Seeded SDUI screens (login, dashboard, exit_code) for android & ios")
    return nil
}
