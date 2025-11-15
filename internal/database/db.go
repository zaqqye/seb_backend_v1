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
    if err := db.AutoMigrate(
        &models.User{},
        &models.Room{},
        &models.Major{},
        &models.ExitCode{},
        &models.RoomSupervisor{},
        &models.RoomStudent{},
        &models.SduiScreen{},
        &models.RefreshToken{},
        &models.StudentStatus{},
        &models.AppConfig{},
    ); err != nil {
        return err
    }
    return createIndexes(db)
}

// createIndexes creates additional, idempotent indexes and extensions that
// GORM AutoMigrate cannot express (partial indexes, trigram, composites used by queries).
func createIndexes(db *gorm.DB) error {
    stmts := []string{
        // Extensions
        `CREATE EXTENSION IF NOT EXISTS pg_trgm`,

        // Exit codes
        `CREATE INDEX IF NOT EXISTS idx_exit_codes_room ON exit_codes (room_id_ref)`,
        `CREATE INDEX IF NOT EXISTS idx_exit_codes_student ON exit_codes (student_user_id_ref)`,
        `CREATE INDEX IF NOT EXISTS idx_exit_codes_user ON exit_codes (user_id_ref)`,
        `CREATE INDEX IF NOT EXISTS idx_exit_codes_used ON exit_codes (used_at)`,
        `CREATE INDEX IF NOT EXISTS idx_exit_codes_unused_code ON exit_codes USING btree (code) WHERE used_at IS NULL`,

        // Room assignments / supervisors
        `CREATE INDEX IF NOT EXISTS idx_room_students_room ON room_students (room_id_ref)`,
        `CREATE INDEX IF NOT EXISTS idx_room_supervisors_room_user ON room_supervisors (room_id_ref, user_id_ref)`,

        // Student statuses
        `CREATE INDEX IF NOT EXISTS idx_student_statuses_user ON student_statuses (user_id_ref)`,
        `CREATE INDEX IF NOT EXISTS idx_student_statuses_flags ON student_statuses (locked, blocked_from_exam)`,
        `CREATE INDEX IF NOT EXISTS idx_student_statuses_updated ON student_statuses (updated_at)`,

        // Users
        `CREATE INDEX IF NOT EXISTS idx_users_role ON users (role)`,
        `CREATE INDEX IF NOT EXISTS idx_users_active ON users (active)`,
        `CREATE INDEX IF NOT EXISTS idx_users_kelas ON users (kelas)`,
        `CREATE INDEX IF NOT EXISTS idx_users_jurusan ON users (jurusan)`,
        `CREATE INDEX IF NOT EXISTS idx_users_fullname_trgm ON users USING GIN (lower(full_name) gin_trgm_ops)`,
        `CREATE INDEX IF NOT EXISTS idx_users_email_trgm ON users USING GIN (lower(email) gin_trgm_ops)`,

        // Rooms
        `CREATE INDEX IF NOT EXISTS idx_rooms_active ON rooms (active)`,
        `CREATE INDEX IF NOT EXISTS idx_rooms_name_trgm ON rooms USING GIN (lower(name) gin_trgm_ops)`,

        // SDUI
        `CREATE INDEX IF NOT EXISTS idx_sdui_active_name_platform ON sdui_screens (active, name, platform)`,
    }
    for _, s := range stmts {
        if err := db.Exec(s).Error; err != nil {
            return err
        }
    }
    return nil
}
