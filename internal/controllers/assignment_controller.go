package controllers

import (
    "errors"
    "fmt"
    "net/http"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
)

type AssignmentController struct {
    DB *gorm.DB
}

type assignRequest struct {
    UserID string `json:"user_id" binding:"required"`
}

// AssignSupervisor assigns a pengawas user to a room
func (ac *AssignmentController) AssignSupervisor(c *gin.Context) {
    roomID := strings.TrimSpace(c.Param("id"))
    if roomID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
        return
    }
    // Ensure room exists
    var room models.Room
    if err := ac.DB.Where("id = ?", roomID).First(&room).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
        return
    }

    var req assignRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    var user models.User
    if err := ac.DB.Where("id = ?", strings.TrimSpace(req.UserID)).First(&user).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }
    if user.Role != "pengawas" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "user is not pengawas"})
        return
    }
    rec := models.RoomSupervisor{UserIDRef: user.ID, RoomIDRef: room.ID}
    if err := ac.DB.Where("user_id_ref = ? AND room_id_ref = ?", rec.UserIDRef, rec.RoomIDRef).FirstOrCreate(&rec).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "assigned"})
}

// UnassignSupervisor removes a pengawas from a room
func (ac *AssignmentController) UnassignSupervisor(c *gin.Context) {
    roomID := strings.TrimSpace(c.Param("id"))
    if roomID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
        return
    }
    userID := strings.TrimSpace(c.Param("user_id"))
    if userID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
        return
    }
    if err := ac.DB.Where("user_id_ref = ? AND room_id_ref = ?", userID, roomID).Delete(&models.RoomSupervisor{}).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "unassigned"})
}

// AssignStudent assigns a siswa user to a room
func (ac *AssignmentController) AssignStudent(c *gin.Context) {
    roomID := strings.TrimSpace(c.Param("id"))
    if roomID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
        return
    }
    var room models.Room
    if err := ac.DB.Where("id = ?", roomID).First(&room).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
        return
    }
    var req assignRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    var user models.User
    if err := ac.DB.Where("id = ?", strings.TrimSpace(req.UserID)).First(&user).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }
    if user.Role != "siswa" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "user is not siswa"})
        return
    }
    if err := ac.DB.Transaction(func(tx *gorm.DB) error {
        rec := models.RoomStudent{UserIDRef: user.ID, RoomIDRef: room.ID}
        if err := tx.Where("user_id_ref = ? AND room_id_ref = ?", rec.UserIDRef, rec.RoomIDRef).FirstOrCreate(&rec).Error; err != nil {
            return err
        }
        var st models.StudentStatus
        if err := tx.Where("user_id_ref = ?", user.ID).First(&st).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                st = models.StudentStatus{UserIDRef: user.ID}
                if err := tx.Create(&st).Error; err != nil {
                    return err
                }
            } else {
                return err
            }
        }
        return nil
    }); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "assigned"})
}

// UnassignStudent removes a siswa from a room
func (ac *AssignmentController) UnassignStudent(c *gin.Context) {
    roomID := strings.TrimSpace(c.Param("id"))
    if roomID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
        return
    }
    userID := strings.TrimSpace(c.Param("user_id"))
    if userID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
        return
    }
    if err := ac.DB.Transaction(func(tx *gorm.DB) error {
        if err := tx.Where("user_id_ref = ? AND room_id_ref = ?", userID, roomID).Delete(&models.RoomStudent{}).Error; err != nil {
            return err
        }
        if err := tx.Where("user_id_ref = ?", userID).Delete(&models.StudentStatus{}).Error; err != nil {
            return err
        }
        return nil
    }); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "unassigned"})
}

// ListSupervisors lists supervisors assigned to a room with pagination/sort.
func (ac *AssignmentController) ListSupervisors(c *gin.Context) {
    roomID := strings.TrimSpace(c.Param("id"))
    if roomID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
        return
    }

    all := strings.EqualFold(c.Query("all"), "true") || c.Query("all") == "1"
    limit := 20
    page := 1
    if v := c.Query("limit"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { limit = n } }
    if v := c.Query("page"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { page = n } }
    sortBy := strings.ToLower(c.DefaultQuery("sort_by", "created_at"))
    sortDir := strings.ToUpper(c.DefaultQuery("sort_dir", "DESC"))
    if sortDir != "ASC" && sortDir != "DESC" { sortDir = "DESC" }
    allowedSorts := map[string]string{
        "created_at": "rs.created_at",
        "full_name":  "u.full_name",
        "email":      "u.email",
    }
    sortCol, ok := allowedSorts[sortBy]; if !ok { sortCol = "rs.created_at" }
    order := fmt.Sprintf("%s %s", sortCol, sortDir)

    var total int64
    if err := ac.DB.Model(&models.RoomSupervisor{}).Where("room_id_ref = ?", roomID).Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    type row struct {
        UserID    string `json:"user_id"`
        FullName  string `json:"full_name"`
        Email     string `json:"email"`
        CreatedAt string `json:"created_at"`
    }
    q := ac.DB.Table("room_supervisors AS rs").
        Select("u.id AS user_id, u.full_name, u.email, rs.created_at").
        Joins("JOIN users u ON u.id = rs.user_id_ref").
        Where("rs.room_id_ref = ?", roomID).
        Order(order)
    if !all { q = q.Offset((page-1)*limit).Limit(limit) }
    var rows []row
    if err := q.Scan(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    meta := gin.H{"total": total, "all": all}
    if !all { meta["limit"] = limit; meta["page"] = page; meta["sort_by"] = sortBy; meta["sort_dir"] = sortDir }
    c.JSON(http.StatusOK, gin.H{"data": rows, "meta": meta})
}

// ListStudents lists students assigned to a room with pagination/sort.
func (ac *AssignmentController) ListStudents(c *gin.Context) {
    roomID := strings.TrimSpace(c.Param("id"))
    if roomID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
        return
    }

    uVal, ok := c.Get("user")
    if ok {
        user := uVal.(models.User)
        if strings.ToLower(user.Role) == "pengawas" {
            var count int64
            if err := ac.DB.Model(&models.RoomSupervisor{}).
                Where("user_id_ref = ? AND room_id_ref = ?", user.ID, roomID).
                Count(&count).Error; err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
            if count == 0 {
                c.JSON(http.StatusForbidden, gin.H{"error": "not allowed for this room"})
                return
            }
        }
    }

    all := strings.EqualFold(c.Query("all"), "true") || c.Query("all") == "1"
    limit := 20
    page := 1
    if v := c.Query("limit"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { limit = n } }
    if v := c.Query("page"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { page = n } }
    sortBy := strings.ToLower(c.DefaultQuery("sort_by", "created_at"))
    sortDir := strings.ToUpper(c.DefaultQuery("sort_dir", "DESC"))
    if sortDir != "ASC" && sortDir != "DESC" { sortDir = "DESC" }
    allowedSorts := map[string]string{
        "created_at": "rs.created_at",
        "full_name":  "u.full_name",
        "email":      "u.email",
        "kelas":      "u.kelas",
        "jurusan":    "u.jurusan",
    }
    sortCol, ok := allowedSorts[sortBy]; if !ok { sortCol = "rs.created_at" }
    order := fmt.Sprintf("%s %s", sortCol, sortDir)

    var total int64
    if err := ac.DB.Model(&models.RoomStudent{}).Where("room_id_ref = ?", roomID).Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    type row struct {
        UserID    string `json:"user_id"`
        FullName  string `json:"full_name"`
        Email     string `json:"email"`
        Kelas     string `json:"kelas"`
        Jurusan   string `json:"jurusan"`
        CreatedAt string `json:"created_at"`
    }
    q := ac.DB.Table("room_students AS rs").
        Select("u.id AS user_id, u.full_name, u.email, u.kelas, u.jurusan, rs.created_at").
        Joins("JOIN users u ON u.id = rs.user_id_ref").
        Where("rs.room_id_ref = ?", roomID).
        Order(order)
    if !all { q = q.Offset((page-1)*limit).Limit(limit) }
    var rows []row
    if err := q.Scan(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    meta := gin.H{"total": total, "all": all}
    if !all { meta["limit"] = limit; meta["page"] = page; meta["sort_by"] = sortBy; meta["sort_dir"] = sortDir }
    c.JSON(http.StatusOK, gin.H{"data": rows, "meta": meta})
}
