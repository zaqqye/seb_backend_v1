package main

import (
    "log"
    "os"

    "github.com/joho/godotenv"

    "github.com/gin-gonic/gin"

    "github.com/zaqqye/seb_backend_v1/internal/config"
    "github.com/zaqqye/seb_backend_v1/internal/database"
    "github.com/zaqqye/seb_backend_v1/internal/routes"
)

func main() {
    // Load .env (non-fatal if missing in production)
    _ = godotenv.Load()

    cfg := config.Load()

    db, err := database.Connect(cfg)
    if err != nil {
        log.Fatalf("database connection failed: %v", err)
    }

    if err := database.Migrate(db); err != nil {
        log.Fatalf("database migration failed: %v", err)
    }

    if err := database.SeedAdmin(db, cfg); err != nil {
        log.Fatalf("admin seed failed: %v", err)
    }

    if err := database.SeedSDUIScreens(db); err != nil {
        log.Fatalf("sdui seed failed: %v", err)
    }

    r := gin.Default()
    routes.Register(r, db, cfg)

    port := cfg.Port
    if port == "" {
        port = "8080"
    }

    if err := r.Run(":" + port); err != nil {
        log.Println("server exited with error:", err)
        os.Exit(1)
    }
}
