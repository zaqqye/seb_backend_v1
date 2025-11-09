package controllers

import (
    "errors"
    "fmt"
    "net/http"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgconn"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
)

type RoomController struct {
    DB *gorm.DB
}

type createRoomRequest struct {
    Name   string `json:"name" binding:"required"`
    Active *bool  `json:"active"`
}

type updateRoomRequest struct {
    Name   *string `json:"name"`
    Active *bool   `json:"active"`
}

func (rc *RoomController) ListRooms(c *gin.Context) {
    uVal, ok := c.Get("user")
    var currentUser models.User
    var isAdmin bool
    if ok {
        currentUser = uVal.(models.User)
        isAdmin = strings.ToLower(currentUser.Role) == "admin"
    }
    // Pagination/sort same pattern as users; add filters: q (name), active
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
        "name":       "name",
        "active":     "active",
    }
    sortCol, ok := allowedSorts[sortBy]
    if !ok {
        sortCol = "created_at"
    }
    order := fmt.Sprintf("%s %s", sortCol, sortDir)
    
    // Filters
    qText := strings.TrimSpace(c.Query("q"))
    activeStr := strings.TrimSpace(strings.ToLower(c.Query("active")))

    base := rc.DB.Model(&models.Room{})
    if !isAdmin && currentUser.ID != "" {
        sub := rc.DB.Table("room_supervisors").Select("room_id_ref").Where("user_id_ref = ?", currentUser.ID)
        base = base.Where("id IN (?)", sub)
    }
    if qText != "" {
        like := "%" + qText + "%"
        base = base.Where("name ILIKE ?", like)
    }
    if activeStr != "" {
        switch activeStr {
        case "true", "1":
            base = base.Where("active = ?", true)
        case "false", "0":
            base = base.Where("active = ?", false)
        default:
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid active value"})
            return
        }
    }

    var total int64
    if err := base.Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var rooms []models.Room
    listQ := rc.DB.Model(&models.Room{}).Order(order)
    if !isAdmin && currentUser.ID != "" {
        sub := rc.DB.Table("room_supervisors").Select("room_id_ref").Where("user_id_ref = ?", currentUser.ID)
        listQ = listQ.Where("id IN (?)", sub)
    }
    // reapply filters on list query
    if qText != "" {
        like := "%" + qText + "%"
        listQ = listQ.Where("name ILIKE ?", like)
    }
    if activeStr != "" {
        switch activeStr {
        case "true", "1":
            listQ = listQ.Where("active = ?", true)
        case "false", "0":
            listQ = listQ.Where("active = ?", false)
        }
    }
    if !all {
        offset := (page - 1) * limit
        listQ = listQ.Offset(offset).Limit(limit)
    }
    if err := listQ.Find(&rooms).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    out := make([]gin.H, 0, len(rooms))
    for _, r := range rooms {
        out = append(out, gin.H{
            "id":         r.ID,
            "name":       r.Name,
            "active":     r.Active,
            "created_at": r.CreatedAt,
            "updated_at": r.UpdatedAt,
        })
    }
    meta := gin.H{"total": total, "all": all}
    if !all {
        meta["limit"] = limit
        meta["page"] = page
        meta["sort_by"] = sortCol
        meta["sort_dir"] = sortDir
    }
    if qText != "" {
        meta["q"] = qText
    }
    if activeStr != "" {
        meta["active"] = activeStr
    }
    c.JSON(http.StatusOK, gin.H{"data": out, "meta": meta})
}

func (rc *RoomController) CreateRoom(c *gin.Context) {
    var req createRoomRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    active := true
    if req.Active != nil {
        active = *req.Active
    }
    room := models.Room{Name: req.Name, Active: active}
    if err := rc.DB.Create(&room).Error; err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            c.JSON(http.StatusConflict, gin.H{"error": "room name already exists"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"message": "created", "id": room.ID})
}

func (rc *RoomController) GetRoom(c *gin.Context) {
    id := strings.TrimSpace(c.Param("id"))
    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    var room models.Room
    if err := rc.DB.Where("id = ?", id).First(&room).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
        return
    }
    c.JSON(http.StatusOK, room)
}

func (rc *RoomController) UpdateRoom(c *gin.Context) {
    id := strings.TrimSpace(c.Param("id"))
    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    var room models.Room
    if err := rc.DB.Where("id = ?", id).First(&room).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
        return
    }
    var req updateRoomRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if req.Name != nil {
        room.Name = *req.Name
    }
    if req.Active != nil {
        room.Active = *req.Active
    }
    if err := rc.DB.Save(&room).Error; err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            c.JSON(http.StatusConflict, gin.H{"error": "room name already exists"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (rc *RoomController) DeleteRoom(c *gin.Context) {
    id := strings.TrimSpace(c.Param("id"))
    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    if err := rc.DB.Where("id = ?", id).Delete(&models.Room{}).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
