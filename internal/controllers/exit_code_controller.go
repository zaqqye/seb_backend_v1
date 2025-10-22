package controllers

import (
    "errors"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgconn"
    "gorm.io/gorm/clause"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
    "github.com/zaqqye/seb_backend_v1/internal/utils"
)

type ExitCodeController struct {
    DB *gorm.DB
}

// Helper: return allowed room IDs for a user. Admins: nil means all.
func (ec *ExitCodeController) allowedRoomIDsFor(user models.User) ([]uint, bool, error) {
    if user.Role == "admin" {
        return nil, true, nil // all rooms allowed
    }
    if user.Role == "pengawas" {
        var m []models.RoomSupervisor
        if err := ec.DB.Where("user_id_ref = ?", user.ID).Find(&m).Error; err != nil {
            return nil, false, err
        }
        ids := make([]uint, 0, len(m))
        for _, r := range m {
            ids = append(ids, r.RoomIDRef)
        }
        return ids, false, nil
    }
    return []uint{}, false, nil // siswa: no access
}

type generateExitCodeRequest struct {
    RoomID *uint `json:"room_id"` // optional for admin; required for pengawas
    Length int   `json:"length"`  // optional length of code; default 6
}

func (ec *ExitCodeController) Generate(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)

    var req generateExitCodeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    codestr, err := utils.GenerateCode(req.Length)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
        return
    }

    allowedRooms, isAdmin, err := ec.allowedRoomIDsFor(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Validate room permission for pengawas
    if !isAdmin {
        if req.RoomID == nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "room_id is required for pengawas"})
            return
        }
        permitted := false
        for _, rid := range allowedRooms {
            if rid == *req.RoomID {
                permitted = true
                break
            }
        }
        if !permitted {
            c.JSON(http.StatusForbidden, gin.H{"error": "not allowed for this room"})
            return
        }
    }

    // If provided, ensure room exists
    if req.RoomID != nil {
        var room models.Room
        if err := ec.DB.First(&room, *req.RoomID).Error; err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
            return
        }
    }

    rec := models.ExitCode{
        UserIDRef: uint(user.ID),
        RoomIDRef: req.RoomID,
        Code:      codestr,
    }
    if err := ec.DB.Create(&rec).Error; err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            c.JSON(http.StatusConflict, gin.H{"error": "code already exists, retry"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "id":         rec.ID,
        "code":       rec.Code,
        "room_id":    rec.RoomIDRef,
        "created_at": rec.CreatedAt,
    })
}

func (ec *ExitCodeController) List(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)

    // Query params: limit, page, all, sort_by, sort_dir, room_id, used
    all := strings.EqualFold(c.Query("all"), "true") || c.Query("all") == "1"
    limit := 20
    page := 1
    if v := c.Query("limit"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 {
            limit = n
        }
    }
    if v := c.Query("page"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 {
            page = n
        }
    }
    sortBy := strings.ToLower(c.DefaultQuery("sort_by", "created_at"))
    sortDir := strings.ToUpper(c.DefaultQuery("sort_dir", "DESC"))
    if sortDir != "ASC" && sortDir != "DESC" {
        sortDir = "DESC"
    }
    allowedSorts := map[string]string{
        "id":         "id",
        "created_at": "created_at",
        "used_at":    "used_at",
        "code":       "code",
    }
    sortCol, ok := allowedSorts[sortBy]
    if !ok {
        sortCol = "created_at"
    }
    order := sortCol + " " + sortDir

    allowedRooms, isAdmin, err := ec.allowedRoomIDsFor(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    base := ec.DB.Model(&models.ExitCode{})
    // Scope by role
    if !isAdmin {
        if len(allowedRooms) == 0 {
            // pengawas with no rooms: return empty list
            c.JSON(http.StatusOK, gin.H{"data": []any{}, "meta": gin.H{"total": 0, "all": all}})
            return
        }
        base = base.Where("room_id_ref IN ?", allowedRooms)
    }

    // Filters
    if roomStr := strings.TrimSpace(c.Query("room_id")); roomStr != "" {
        if roomID, err := strconv.Atoi(roomStr); err == nil && roomID > 0 {
            base = base.Where("room_id_ref = ?", roomID)
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
            return
        }
    }
    // used filter; default show only unused (used_at is NULL)
    used := strings.TrimSpace(strings.ToLower(c.DefaultQuery("used", "false")))
    switch used {
    case "true", "1":
        base = base.Where("used_at IS NOT NULL")
    case "false", "0":
        base = base.Where("used_at IS NULL")
    case "all":
        // no filter
    default:
        base = base.Where("used_at IS NULL")
    }

    var total int64
    if err := base.Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var items []models.ExitCode
    listQ := ec.DB.Order(order)
    if !isAdmin {
        listQ = listQ.Where("room_id_ref IN ?", allowedRooms)
    }
    // Reapply filters similarly as base
    if roomStr := strings.TrimSpace(c.Query("room_id")); roomStr != "" {
        if roomID, err := strconv.Atoi(roomStr); err == nil && roomID > 0 {
            listQ = listQ.Where("room_id_ref = ?", roomID)
        }
    }
    switch used {
    case "true", "1":
        listQ = listQ.Where("used_at IS NOT NULL")
    case "false", "0":
        listQ = listQ.Where("used_at IS NULL")
    case "all":
        // none
    default:
        listQ = listQ.Where("used_at IS NULL")
    }
    if !all {
        offset := (page - 1) * limit
        listQ = listQ.Offset(offset).Limit(limit)
    }
    if err := listQ.Find(&items).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    out := make([]gin.H, 0, len(items))
    for _, e := range items {
        out = append(out, gin.H{
            "id":         e.ID,
            "room_id":    e.RoomIDRef,
            "code":       e.Code,
            "used_at":    e.UsedAt,
            "created_at": e.CreatedAt,
            "created_by": e.UserIDRef,
        })
    }
    meta := gin.H{"total": total, "all": all}
    if !all {
        meta["limit"] = limit
        meta["page"] = page
        meta["sort_by"] = sortCol
        meta["sort_dir"] = sortDir
    }
    c.JSON(http.StatusOK, gin.H{"data": out, "meta": meta})
}

func (ec *ExitCodeController) Revoke(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)

    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    var rec models.ExitCode
    if err := ec.DB.First(&rec, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "exit code not found"})
        return
    }

    // Scope check for pengawas
    if user.Role == "pengawas" {
        var count int64
        if rec.RoomIDRef == nil {
            // Pengawas cannot revoke non-room-specific codes
            c.JSON(http.StatusForbidden, gin.H{"error": "not allowed to revoke this code"})
            return
        }
        if err := ec.DB.Model(&models.RoomSupervisor{}).
            Where("user_id_ref = ? AND room_id_ref = ?", user.ID, *rec.RoomIDRef).
            Count(&count).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        if count == 0 {
            c.JSON(http.StatusForbidden, gin.H{"error": "not allowed for this room"})
            return
        }
    }

    if rec.UsedAt == nil {
        now := time.Now().UTC()
        rec.UsedAt = &now
        if err := ec.DB.Save(&rec).Error; err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    }
    c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}

// Consume marks a code as used (single-use). If already used, returns 409.
type consumeRequest struct {
    Code   string `json:"code" binding:"required"`
    RoomID *uint  `json:"room_id"`
}

func (ec *ExitCodeController) Consume(c *gin.Context) {
    var req consumeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    now := time.Now().UTC()

    err := ec.DB.Transaction(func(tx *gorm.DB) error {
        var rec models.ExitCode
        q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ? AND used_at IS NULL", req.Code)
        if req.RoomID != nil {
            q = q.Where("(room_id_ref = ?)", *req.RoomID)
        }
        if err := q.First(&rec).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                return err
            }
            return err
        }
        rec.UsedAt = &now
        if err := tx.Save(&rec).Error; err != nil {
            return err
        }
        return nil
    })
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusConflict, gin.H{"error": "code not found or already used"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "consumed"})
}
