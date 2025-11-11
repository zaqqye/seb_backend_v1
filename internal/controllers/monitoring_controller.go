package controllers

import (
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
    "github.com/zaqqye/seb_backend_v1/internal/ws"
)

type MonitoringController struct {
    DB   *gorm.DB
    Hubs *ws.Hubs
}

// allowedRoomIDsFor returns room IDs supervised by user; admin -> (nil, true)
func (mc *MonitoringController) allowedRoomIDsFor(user models.User) ([]string, bool, error) {
    if user.Role == "admin" {
        return nil, true, nil
    }
    if user.Role == "pengawas" {
        var m []models.RoomSupervisor
        if err := mc.DB.Where("user_id_ref = ?", user.ID).Find(&m).Error; err != nil {
            return nil, false, err
        }
        ids := make([]string, 0, len(m))
        for _, r := range m { ids = append(ids, r.RoomIDRef) }
        return ids, false, nil
    }
    return []string{}, false, nil
}

// ListStudents returns monitoring rows scoped by role.
func (mc *MonitoringController) ListStudents(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)

    all := strings.EqualFold(c.Query("all"), "true") || c.Query("all") == "1"
    limit := 50
    page := 1
    if v := c.Query("limit"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { limit = n } }
    if v := c.Query("page"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { page = n } }
    sortBy := strings.ToLower(c.DefaultQuery("sort_by", "updated_at"))
    sortDir := strings.ToUpper(c.DefaultQuery("sort_dir", "DESC"))
    if sortDir != "ASC" && sortDir != "DESC" { sortDir = "DESC" }
    allowedSorts := map[string]string{
        "updated_at": "monitoring_updated_at",
        "full_name":  "full_name",
        "email":      "email",
        "kelas":      "kelas",
        "jurusan":    "jurusan",
        "locked":     "monitoring_locked",
    }
    sortCol, ok := allowedSorts[sortBy]; if !ok { sortCol = allowedSorts["updated_at"] }
    order := sortCol + " " + sortDir

    qText := strings.TrimSpace(c.Query("q"))
    roomID := strings.TrimSpace(c.Query("room_id"))

    allowedRooms, isAdmin, err := mc.allowedRoomIDsFor(user)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }

    // Base query from users (siswa only)
    type row struct {
        UserID               string     `gorm:"column:user_id"`
        FullName             string     `gorm:"column:full_name"`
        Email                string     `gorm:"column:email"`
        Kelas                string     `gorm:"column:kelas"`
        Jurusan              string     `gorm:"column:jurusan"`
        StatusID             *string    `gorm:"column:status_id"`
        AppVersion           string     `gorm:"column:app_version"`
        MonitoringLocked     bool       `gorm:"column:monitoring_locked"`
        BlockedFromExam      bool       `gorm:"column:blocked_from_exam"`
        ForceLogoutAt        *time.Time `gorm:"column:force_logout_at"`
        MonitoringUpdatedAt  *time.Time `gorm:"column:monitoring_updated_at"`
        RoomID               *string    `gorm:"column:room_id"`
        RoomName             *string    `gorm:"column:room_name"`
    }

    applyFilters := func(q *gorm.DB) *gorm.DB {
        if qText != "" {
            like := "%" + qText + "%"
            q = q.Where("u.full_name ILIKE ? OR u.email ILIKE ?", like, like)
        }
        if !isAdmin {
            if len(allowedRooms) == 0 {
                return q.Where("1 = 0")
            }
            q = q.Where("rs.room_id_ref::text IN ?", allowedRooms)
        }
        if roomID != "" {
            q = q.Where("rs.room_id_ref = ?", roomID)
        }
        return q
    }

    base := mc.DB.Table("users AS u").
        Select(`
            u.id AS user_id,
            u.full_name AS full_name,
            u.email AS email,
            u.kelas AS kelas,
            u.jurusan AS jurusan,
            ss.id AS status_id,
            COALESCE(ss.app_version, '') AS app_version,
            COALESCE(ss.locked, FALSE) AS monitoring_locked,
            COALESCE(ss.blocked_from_exam, FALSE) AS blocked_from_exam,
            ss.force_logout_at AS force_logout_at,
            COALESCE(ss.updated_at, u.updated_at) AS monitoring_updated_at,
            r.id AS room_id,
            r.name AS room_name`).
        Joins("LEFT JOIN student_statuses ss ON ss.user_id_ref = u.id").
        Joins("LEFT JOIN room_students rs ON rs.user_id_ref = u.id").
        Joins("LEFT JOIN rooms r ON r.id = rs.room_id_ref").
        Where("u.role = ?", "siswa")
    base = applyFilters(base)
    if !isAdmin && len(allowedRooms) == 0 {
        c.JSON(http.StatusOK, gin.H{"data": []any{}, "meta": gin.H{"total": 0, "all": all}})
        return
    }

    countQ := mc.DB.Table("users AS u").
        Joins("LEFT JOIN student_statuses ss ON ss.user_id_ref = u.id").
        Joins("LEFT JOIN room_students rs ON rs.user_id_ref = u.id").
        Where("u.role = ?", "siswa")
    countQ = applyFilters(countQ)

    var total int64
    if err := countQ.Distinct("u.id").Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
    }

    listQ := base.Order(order)
    if !all { listQ = listQ.Offset((page-1)*limit).Limit(limit) }
    var rows []row
    if err := listQ.Find(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
    }

    strOrEmpty := func(val *string) string {
        if val == nil { return "" }
        return *val
    }

    type monitoringBlock struct {
        ID              string     `json:"id"`
        AppVersion      string     `json:"app_version"`
        Locked          bool       `json:"locked"`
        BlockedFromExam bool       `json:"blocked_from_exam"`
        ForceLogoutAt   *time.Time `json:"force_logout_at"`
        UpdatedAt       *time.Time `json:"updated_at"`
    }
    type roomBlock struct {
        ID       string `json:"id"`
        RoomName string `json:"room_name"`
    }
    type response struct {
        ID         string          `json:"id"`
        FullName   string          `json:"full_name"`
        Email      string          `json:"email"`
        Kelas      string          `json:"kelas"`
        Jurusan    string          `json:"jurusan"`
        Monitoring monitoringBlock `json:"monitoring"`
        Room       roomBlock       `json:"room"`
    }

    data := make([]response, 0, len(rows))
    for _, r := range rows {
        data = append(data, response{
            ID:       r.UserID,
            FullName: r.FullName,
            Email:    r.Email,
            Kelas:    r.Kelas,
            Jurusan:  r.Jurusan,
            Monitoring: monitoringBlock{
                ID:              strOrEmpty(r.StatusID),
                AppVersion:      r.AppVersion,
                Locked:          r.MonitoringLocked,
                BlockedFromExam: r.BlockedFromExam,
                ForceLogoutAt:   r.ForceLogoutAt,
                UpdatedAt:       r.MonitoringUpdatedAt,
            },
            Room: roomBlock{
                ID:       strOrEmpty(r.RoomID),
                RoomName: strOrEmpty(r.RoomName),
            },
        })
    }

    meta := gin.H{"total": total, "all": all}
    if !all { meta["limit"] = limit; meta["page"] = page; meta["sort_by"] = sortBy; meta["sort_dir"] = sortDir }
    c.JSON(http.StatusOK, gin.H{"data": data, "meta": meta})
}

// ForceLogout marks the student to be logged out and blocks exam until allowed again.
func (mc *MonitoringController) ForceLogout(c *gin.Context) {
    uVal, _ := c.Get("user")
    actor := uVal.(models.User)
    idStr := strings.TrimSpace(c.Param("id"))
    if idStr == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }

    var target models.User
    if err := mc.DB.Where("id = ?", idStr).First(&target).Error; err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "user not found"}); return }
    if strings.ToLower(target.Role) != "siswa" { c.JSON(http.StatusBadRequest, gin.H{"error": "target is not siswa"}); return }

    // Scope check for pengawas
    if actor.Role == "pengawas" {
        var count int64
        mc.DB.Table("room_students").Where("user_id_ref = ? AND room_id_ref IN (?)", target.ID, mc.DB.Table("room_supervisors").Select("room_id_ref").Where("user_id_ref = ?", actor.ID)).Count(&count)
        if count == 0 { c.JSON(http.StatusForbidden, gin.H{"error": "not allowed for this student"}); return }
    }

    now := time.Now().UTC()
    var st models.StudentStatus
    if err := mc.DB.Where("user_id_ref = ?", target.ID).First(&st).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            st = models.StudentStatus{UserIDRef: target.ID, BlockedFromExam: true, ForceLogoutAt: &now}
            if err := mc.DB.Create(&st).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        } else { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    } else {
        st.BlockedFromExam = true
        st.ForceLogoutAt = &now
        st.Locked = false
        if err := mc.DB.Save(&st).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    }
    c.JSON(http.StatusOK, gin.H{"message": "student logged out and blocked"})
    go broadcastStudentStatus(mc.DB, mc.Hubs, target.ID)
}

// AllowExam clears the block so the student can start exam again.
func (mc *MonitoringController) AllowExam(c *gin.Context) {
    uVal, _ := c.Get("user")
    actor := uVal.(models.User)
    idStr := strings.TrimSpace(c.Param("id"))
    if idStr == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }

    var target models.User
    if err := mc.DB.Where("id = ?", idStr).First(&target).Error; err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "user not found"}); return }
    if strings.ToLower(target.Role) != "siswa" { c.JSON(http.StatusBadRequest, gin.H{"error": "target is not siswa"}); return }

    if actor.Role == "pengawas" {
        var count int64
        mc.DB.Table("room_students").Where("user_id_ref = ? AND room_id_ref IN (?)", target.ID, mc.DB.Table("room_supervisors").Select("room_id_ref").Where("user_id_ref = ?", actor.ID)).Count(&count)
        if count == 0 { c.JSON(http.StatusForbidden, gin.H{"error": "not allowed for this student"}); return }
    }

    var st models.StudentStatus
    if err := mc.DB.Where("user_id_ref = ?", target.ID).First(&st).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            st = models.StudentStatus{UserIDRef: target.ID, BlockedFromExam: false}
            if err := mc.DB.Create(&st).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        } else { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    } else {
        st.BlockedFromExam = false
        if err := mc.DB.Save(&st).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    }
    c.JSON(http.StatusOK, gin.H{"message": "student allowed to start exam"})
    go broadcastStudentStatus(mc.DB, mc.Hubs, target.ID)
}
