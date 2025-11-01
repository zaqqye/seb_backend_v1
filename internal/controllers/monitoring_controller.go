package controllers

import (
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
)

type MonitoringController struct {
    DB *gorm.DB
}

// allowedRoomIDsFor returns room IDs supervised by user; admin -> (nil, true)
func (mc *MonitoringController) allowedRoomIDsFor(user models.User) ([]uint, bool, error) {
    if user.Role == "admin" {
        return nil, true, nil
    }
    if user.Role == "pengawas" {
        var m []models.RoomSupervisor
        if err := mc.DB.Where("user_id_ref = ?", user.ID).Find(&m).Error; err != nil {
            return nil, false, err
        }
        ids := make([]uint, 0, len(m))
        for _, r := range m { ids = append(ids, r.RoomIDRef) }
        return ids, false, nil
    }
    return []uint{}, false, nil
}

// ListStudents returns monitoring rows scoped by role.
func (mc *MonitoringController) ListStudents(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)

    all := strings.EqualFold(c.Query("all"), "true") || c.Query("all") == "1"
    limit := 20
    page := 1
    if v := c.Query("limit"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { limit = n } }
    if v := c.Query("page"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { page = n } }
    sortBy := strings.ToLower(c.DefaultQuery("sort_by", "updated_at"))
    sortDir := strings.ToUpper(c.DefaultQuery("sort_dir", "DESC"))
    if sortDir != "ASC" && sortDir != "DESC" { sortDir = "DESC" }
    allowedSorts := map[string]string{
        "updated_at": "COALESCE(ss.updated_at, u.updated_at)",
        "full_name":  "u.full_name",
        "email":      "u.email",
        "kelas":      "u.kelas",
        "jurusan":    "u.jurusan",
        "locked":     "ss.locked",
    }
    sortCol, ok := allowedSorts[sortBy]; if !ok { sortCol = allowedSorts["updated_at"] }
    order := sortCol + " " + sortDir

    qText := strings.TrimSpace(c.Query("q"))
    roomStr := strings.TrimSpace(c.Query("room_id"))
    var roomID int
    if roomStr != "" { if n, err := strconv.Atoi(roomStr); err == nil { roomID = n } }

    allowedRooms, isAdmin, err := mc.allowedRoomIDsFor(user)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }

    // Base query from users (siswa only)
    type row struct {
        UserID   uint      `json:"id"`
        FullName string    `json:"full_name"`
        Email    string    `json:"email"`
        Kelas    string    `json:"kelas"`
        Jurusan  string    `json:"jurusan"`
        AppVer   string    `json:"app_version"`
        Locked   bool      `json:"locked"`
        Blocked  bool      `json:"blocked_from_exam"`
        Updated  *time.Time `json:"updated_at"`
    }

    base := mc.DB.Table("users AS u").
        Select("u.id, u.full_name, u.email, u.kelas, u.jurusan, COALESCE(ss.app_version, '') AS app_version, COALESCE(ss.locked, FALSE) AS locked, COALESCE(ss.blocked_from_exam, FALSE) AS blocked_from_exam, ss.updated_at").
        Joins("LEFT JOIN student_statuses ss ON ss.user_id_ref = u.id").
        Where("u.role = ?", "siswa")

    // Apply search
    if qText != "" {
        like := "%" + qText + "%"
        base = base.Where("u.full_name ILIKE ? OR u.email ILIKE ?", like, like)
    }
    // Apply room scoping and filtering
    if !isAdmin {
        if len(allowedRooms) == 0 {
            c.JSON(http.StatusOK, gin.H{"data": []any{}, "meta": gin.H{"total": 0, "all": all}})
            return
        }
        base = base.Joins("JOIN room_students rs ON rs.user_id_ref = u.id").Where("rs.room_id_ref IN ?", allowedRooms)
    }
    if roomID > 0 {
        base = base.Joins("JOIN room_students frs ON frs.user_id_ref = u.id").Where("frs.room_id_ref = ?", roomID)
    }

    var total int64
    if err := base.Distinct("u.id").Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
    }

    listQ := base.Distinct("u.id, u.full_name, u.email, u.kelas, u.jurusan, ss.app_version, ss.locked, ss.blocked_from_exam, ss.updated_at").Order(order)
    if !all { listQ = listQ.Offset((page-1)*limit).Limit(limit) }
    var rows []row
    if err := listQ.Scan(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
    }
    meta := gin.H{"total": total, "all": all}
    if !all { meta["limit"] = limit; meta["page"] = page; meta["sort_by"] = sortBy; meta["sort_dir"] = sortDir }
    c.JSON(http.StatusOK, gin.H{"data": rows, "meta": meta})
}

// ForceLogout marks the student to be logged out and blocks exam until allowed again.
func (mc *MonitoringController) ForceLogout(c *gin.Context) {
    uVal, _ := c.Get("user")
    actor := uVal.(models.User)
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }

    var target models.User
    if err := mc.DB.First(&target, id).Error; err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "user not found"}); return }
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
}

// AllowExam clears the block so the student can start exam again.
func (mc *MonitoringController) AllowExam(c *gin.Context) {
    uVal, _ := c.Get("user")
    actor := uVal.(models.User)
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }

    var target models.User
    if err := mc.DB.First(&target, id).Error; err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "user not found"}); return }
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
}
