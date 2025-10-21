package routes

import (
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/example/seb_backend_v1/internal/config"
    "github.com/example/seb_backend_v1/internal/controllers"
    "github.com/example/seb_backend_v1/internal/middleware"
)

func Register(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
    // Controllers
    expiresMins, err := time.ParseDuration(cfg.JWTExpiresIn + "m")
    if err != nil || expiresMins == 0 {
        expiresMins = 60 * time.Minute
    }
    authCtrl := &controllers.AuthController{DB: db, JWTSecret: cfg.JWTSecret, ExpiresIn: expiresMins}
    adminCtrl := &controllers.AdminController{DB: db}

    // Public
    auth := r.Group("/api/auth")
    {
        // Registration restricted to admin; moved under /api/admin/users
        auth.POST("/login", authCtrl.Login)
    }

    // Protected
    authMW := middleware.AuthMiddleware(db, middleware.AuthConfig{
        JWTSecret:    cfg.JWTSecret,
        JWTExpiresIn: expiresMins,
    })
    api := r.Group("/api", authMW)
    {
        api.GET("/auth/me", authCtrl.Me)
        api.POST("/auth/logout", authCtrl.Logout)

        // Admin-only
        admin := api.Group("/admin", middleware.RequireRoles("admin"))
        {
            admin.GET("/users", adminCtrl.ListUsers)
            admin.POST("/users", authCtrl.Register) // admin-only registration (supports role/active)
            admin.GET("/users/:user_id", adminCtrl.GetUser)
            admin.PUT("/users/:user_id", adminCtrl.UpdateUser)
            admin.DELETE("/users/:user_id", adminCtrl.DeleteUser)
        }

        // Pengawas area (and admin)
        pengawas := api.Group("/pengawas", middleware.RequireRoles("pengawas", "admin"))
        {
            pengawas.GET("/panel", func(c *gin.Context) {
                c.JSON(200, gin.H{"message": "pengawas panel"})
            })
        }

        // Siswa area (and admin)
        siswa := api.Group("/siswa", middleware.RequireRoles("siswa", "admin"))
        {
            siswa.GET("/panel", func(c *gin.Context) {
                c.JSON(200, gin.H{"message": "siswa panel"})
            })
        }
    }
}
