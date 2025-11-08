package routes

import (
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/config"
    "github.com/zaqqye/seb_backend_v1/internal/controllers"
    "github.com/zaqqye/seb_backend_v1/internal/middleware"
    "github.com/zaqqye/seb_backend_v1/internal/ws"
)

func Register(r *gin.Engine, db *gorm.DB, cfg *config.Config, hubs *ws.Hubs) {
    // Controllers
    expiresMins, err := time.ParseDuration(cfg.JWTExpiresIn + "m")
    if err != nil || expiresMins == 0 {
        expiresMins = 60 * time.Minute
    }
    // Token TTLs
    accessTTL, err2 := time.ParseDuration(cfg.AccessTokenTTLMinutes + "m")
    if err2 != nil || accessTTL <= 0 {
        accessTTL = 15 * time.Minute
    }
    refreshDays, err3 := time.ParseDuration(cfg.RefreshTokenTTLDays + "24h")
    if err3 != nil || refreshDays <= 0 {
        refreshDays = 30 * 24 * time.Hour
    }
    authCtrl := &controllers.AuthController{
        DB:            db,
        AccessSecret:  cfg.JWTSecret,
        RefreshSecret: cfg.RefreshJWTSecret,
        AccessTTL:     accessTTL,
        RefreshTTL:    refreshDays,
    }
    adminCtrl := &controllers.AdminController{DB: db}
    roomCtrl := &controllers.RoomController{DB: db}
    majorCtrl := &controllers.MajorController{DB: db}
    studentStatusCtrl := &controllers.StudentStatusController{DB: db, Hubs: hubs}
    monCtrl := &controllers.MonitoringController{DB: db, Hubs: hubs}

    // Public
    auth := r.Group("/api/v1/auth")
    {
        // Registration restricted to admin; moved under /api/admin/users
        auth.POST("/login", authCtrl.Login)
        auth.POST("/refresh", authCtrl.Refresh)
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
            admin.POST("/users/import", adminCtrl.ImportUsers)

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
        siswa := api.Group("/siswa", middleware.RequireRoles("siswa", "admin", "pengawas"))
        {
            siswa.GET("/panel", func(c *gin.Context) {
                c.JSON(200, gin.H{"message": "siswa panel"})
            })
            // Student status update/read for monitoring
            siswa.GET("/status", studentStatusCtrl.GetSelf)
            siswa.POST("/status", studentStatusCtrl.UpdateSelf)
        }

        // Exit Codes (admin + pengawas)
        exitCtrl := &controllers.ExitCodeController{DB: db, Hubs: hubs}
        exit := api.Group("/exit-codes", middleware.RequireRoles("admin", "pengawas"))
        {
            exit.POST("/generate", exitCtrl.Generate)
            exit.GET("", exitCtrl.List)
            exit.POST(":id/revoke", exitCtrl.Revoke)
            // Consume endpoint for mobile app
            // Note: route consume untuk siswa/pengawas/admin didefinisikan di luar group ini
        }

        // Siswa, pengawas, dan admin boleh consume exit code (auth wajib)
        api.POST("/exit-codes/consume", middleware.RequireRoles("siswa", "pengawas", "admin"), exitCtrl.Consume)

        // Monitoring (admin + pengawas)
        monitoring := api.Group("/monitoring", middleware.RequireRoles("admin", "pengawas"))
        {
            monitoring.GET("/students", monCtrl.ListStudents)
            monitoring.POST("/students/:id/logout", monCtrl.ForceLogout)
            monitoring.POST("/students/:id/allow", monCtrl.AllowExam)
        }

        // SDUI and Config with auth context (role-aware)
        api.GET("/sdui/auth/screens/:name", sduiCtrl.GetScreen)
        api.GET("/config", cfgCtrl.Get)
    }
    wsGroup := r.Group("/ws", authMW)
    {
        wsGroup.GET("/monitoring", middleware.RequireRoles("admin", "pengawas"), ws.MonitoringHandler(db, hubs.Monitoring))
        wsGroup.GET("/siswa/status", middleware.RequireRoles("siswa"), ws.StudentHandler(hubs))
    }
}
