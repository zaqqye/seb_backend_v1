package routes

import (
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/config"
    "github.com/zaqqye/seb_backend_v1/internal/controllers"
    "github.com/zaqqye/seb_backend_v1/internal/middleware"
)

func Register(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
    // Controllers
    expiresMins, err := time.ParseDuration(cfg.JWTExpiresIn + "m")
    if err != nil || expiresMins == 0 {
        expiresMins = 60 * time.Minute
    }
    authCtrl := &controllers.AuthController{DB: db, JWTSecret: cfg.JWTSecret, ExpiresIn: expiresMins}
    adminCtrl := &controllers.AdminController{DB: db}
    roomCtrl := &controllers.RoomController{DB: db}
    majorCtrl := &controllers.MajorController{DB: db}

    // Public
    auth := r.Group("/api/v1/auth")
    {
        // Registration restricted to admin; moved under /api/admin/users
        auth.POST("/login", authCtrl.Login)
    }

    // Public SDUI and Config (non-auth; some screens will 401 if role missing)
    sduiCtrl := &controllers.SDUIController{DB: db, Cfg: cfg}
    r.GET("/api/v1/sdui/screens/:name", sduiCtrl.GetScreen)

    cfgCtrl := &controllers.ConfigController{DB: db, Cfg: cfg}
    r.GET("/api/v1/config/public", cfgCtrl.Get)

    // Protected
    authMW := middleware.AuthMiddleware(db, middleware.AuthConfig{
        JWTSecret:    cfg.JWTSecret,
        JWTExpiresIn: expiresMins,
    })
    api := r.Group("/api/v1", authMW)
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

            // Rooms (Kelas) CRUD
            admin.GET("/rooms", roomCtrl.ListRooms)
            admin.POST("/rooms", roomCtrl.CreateRoom)
            admin.GET("/rooms/:id", roomCtrl.GetRoom)
            admin.PUT("/rooms/:id", roomCtrl.UpdateRoom)
            admin.DELETE("/rooms/:id", roomCtrl.DeleteRoom)

            // Majors (Jurusan) CRUD
            admin.GET("/majors", majorCtrl.ListMajors)
            admin.POST("/majors", majorCtrl.CreateMajor)
            admin.GET("/majors/:id", majorCtrl.GetMajor)
            admin.PUT("/majors/:id", majorCtrl.UpdateMajor)
            admin.DELETE("/majors/:id", majorCtrl.DeleteMajor)

            // Assignments: supervisors and students to rooms
            assignCtrl := &controllers.AssignmentController{DB: db}
            admin.POST("/rooms/:id/supervisors", assignCtrl.AssignSupervisor)
            admin.DELETE("/rooms/:id/supervisors/:user_id", assignCtrl.UnassignSupervisor)
            admin.GET("/rooms/:id/supervisors", assignCtrl.ListSupervisors)
            admin.POST("/rooms/:id/students", assignCtrl.AssignStudent)
            admin.DELETE("/rooms/:id/students/:user_id", assignCtrl.UnassignStudent)
            admin.GET("/rooms/:id/students", assignCtrl.ListStudents)

            // SDUI screens admin CRUD
            sduiAdmin := &controllers.SDUIAdminController{DB: db}
            admin.GET("/sdui/screens", sduiAdmin.List)
            admin.POST("/sdui/screens", sduiAdmin.Create)
            admin.GET("/sdui/screens/:id", sduiAdmin.Get)
            admin.PUT("/sdui/screens/:id", sduiAdmin.Update)
            admin.DELETE("/sdui/screens/:id", sduiAdmin.Delete)
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

        // Exit Codes (admin + pengawas)
        exitCtrl := &controllers.ExitCodeController{DB: db}
        exit := api.Group("/exit-codes", middleware.RequireRoles("admin", "pengawas"))
        {
            exit.POST("/generate", exitCtrl.Generate)
            exit.GET("", exitCtrl.List)
            exit.POST(":id/revoke", exitCtrl.Revoke)
            // Consume endpoint for mobile app
            exit.POST("/consume", exitCtrl.Consume)
        }

        // SDUI and Config with auth context (role-aware)
        api.GET("/sdui/auth/screens/:name", sduiCtrl.GetScreen)
        api.GET("/config", cfgCtrl.Get)
    }
}
